package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

type MessageHandler func(ctx context.Context, data json.RawMessage) error

type Consumer struct {
	channel      *amqp091.Channel
	queue        amqp091.Queue
	routingKey   string
	handler      MessageHandler
	conn         *amqp091.Connection
	logger       *zap.Logger
	dlqPublisher *Publisher // 用于发送到死信队列
	maxRetries   int        // 最大重试次数
	ctx          context.Context
	cancel       context.CancelFunc
	stopChan     chan struct{}
}

// NewConsumer creates a consumer for a specific routing key.
func NewConsumer(url, queueName, routingKey string, logger *zap.Logger) (*Consumer, error) {
	return NewConsumerWithRetry(url, queueName, routingKey, logger, 3)
}

// NewConsumerWithRetry creates a consumer with retry and DLQ support.
func NewConsumerWithRetry(url, queueName, routingKey string, logger *zap.Logger, maxRetries int) (*Consumer, error) {
	conn, err := NewConnection(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// 声明主 Exchange
	if err := DeclareExchange(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// 声明 DLQ Exchange
	if err := DeclareDLQExchange(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ exchange: %w", err)
	}

	// 声明 DLQ Queue
	_, err = DeclareDLQQueue(ch, routingKey)
	if err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare DLQ queue: %w", err)
	}

	// 声明主 Queue
	q, err := ch.QueueDeclare(
		queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	err = ch.QueueBind(
		q.Name,
		routingKey,
		ExchangeName,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to bind queue: %w", err)
	}

	// 创建 DLQ Publisher（复用同一个连接）
	dlqPublisher := &Publisher{
		conn:    conn,
		channel: ch,
	}

	logger.Info("Consumer initialized",
		zap.String("routing_key", routingKey),
		zap.String("queue", queueName),
		zap.String("exchange", ExchangeName),
		zap.Int("max_retries", maxRetries),
	)

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		conn:         conn,
		channel:     ch,
		queue:       q,
		routingKey:  routingKey,
		logger:      logger,
		dlqPublisher: dlqPublisher,
		maxRetries:  maxRetries,
		ctx:         ctx,
		cancel:      cancel,
		stopChan:    make(chan struct{}),
	}, nil
}

func (c *Consumer) SetHandler(h MessageHandler) {
	c.handler = h
}

func (c *Consumer) Close() {
	if c.channel != nil {
		_ = c.channel.Close()
	}
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

// IsConnected checks if the consumer connection is still alive
func (c *Consumer) IsConnected() bool {
	if c.conn == nil || c.channel == nil {
		return false
	}
	// Check if connection is closed
	if c.conn.IsClosed() {
		return false
	}
	return true
}

// StartConsuming starts consuming messages. This method blocks and should be called in a goroutine.
func (c *Consumer) StartConsuming() error {
	if c.handler == nil {
		return fmt.Errorf("consumer handler not set")
	}

	deliveries, err := c.channel.Consume(
		c.queue.Name,
		"worker",
		false, // 手动ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register consumer: %w", err)
	}

	c.logger.Info("Consumer started consuming messages",
		zap.String("routing_key", c.routingKey),
		zap.String("queue", c.queue.Name),
	)

	// 最安全的消费模型：保证每条消息都会被 ack 或 nack
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("Consumer stopping, waiting for current messages to complete...",
				zap.String("routing_key", c.routingKey),
			)
			close(c.stopChan)
			return nil
		case msg, ok := <-deliveries:
			if !ok {
				c.logger.Info("Consumer channel closed",
					zap.String("routing_key", c.routingKey),
				)
				close(c.stopChan)
				return nil
			}
			func() {
				ctx := context.Background()

				c.logger.Debug("Received message",
					zap.String("routing_key", c.routingKey),
					zap.String("queue", c.queue.Name),
					zap.Int("message_size", len(msg.Body)),
				)

			// Panic 恢复：确保即使 handler panic 也能正确处理消息
			defer func() {
				if r := recover(); r != nil {
					c.logger.Error("Handler panic recovered",
						zap.String("routing_key", c.routingKey),
						zap.String("queue", c.queue.Name),
						zap.Any("panic", r),
					)
					// Panic → 拒绝消息并重新入队
					if err := msg.Nack(false, true); err != nil {
						c.logger.Error("Failed to nack message after panic",
							zap.String("routing_key", c.routingKey),
							zap.Error(err),
						)
					}
				}
			}()

			// 获取重试次数（从消息头）
			retryCount := 0
			if retryHeader, ok := msg.Headers["x-retry-count"]; ok {
				if count, ok := retryHeader.(int64); ok {
					retryCount = int(count)
				}
			}

			// 执行业务处理
			if err := c.handler(ctx, msg.Body); err != nil {
				c.logger.Error("Handler error",
					zap.String("routing_key", c.routingKey),
					zap.String("queue", c.queue.Name),
					zap.Int("retry_count", retryCount),
					zap.Error(err),
				)

				// 检查错误类型：JSON 解析错误直接发送到 DLQ，不重试
				errStr := err.Error()
				if contains(errStr, "json_unmarshal_error") || contains(errStr, "json:") {
					c.logger.Warn("JSON unmarshal error, sending directly to DLQ",
						zap.String("routing_key", c.routingKey),
						zap.Error(err),
					)
					
					// 发送到死信队列
					if dlqErr := c.dlqPublisher.PublishToDLQ(c.routingKey, msg.Body, err.Error()); dlqErr != nil {
						c.logger.Error("Failed to publish to DLQ",
							zap.String("routing_key", c.routingKey),
							zap.Error(dlqErr),
						)
					}
					
					// Ack 掉原消息（已发送到 DLQ）
					if err := msg.Ack(false); err != nil {
						c.logger.Error("Failed to ack message after DLQ",
							zap.String("routing_key", c.routingKey),
							zap.Error(err),
						)
					}
					return
				}

				// 检查是否超过最大重试次数
				if retryCount >= c.maxRetries {
					c.logger.Warn("Max retries exceeded, sending to DLQ",
						zap.String("routing_key", c.routingKey),
						zap.Int("retry_count", retryCount),
						zap.Int("max_retries", c.maxRetries),
					)
					
					// 发送到死信队列
					if dlqErr := c.dlqPublisher.PublishToDLQ(c.routingKey, msg.Body, err.Error()); dlqErr != nil {
						c.logger.Error("Failed to publish to DLQ",
							zap.String("routing_key", c.routingKey),
							zap.Error(dlqErr),
						)
					}
					
					// Ack 掉原消息（已发送到 DLQ）
					if err := msg.Ack(false); err != nil {
						c.logger.Error("Failed to ack message after DLQ",
							zap.String("routing_key", c.routingKey),
							zap.Error(err),
						)
					}
					return
				}

				// 未超过最大重试次数 → 拒绝消息并重新入队
				// 注意：RabbitMQ 不会自动增加重试计数，需要手动设置
				if err := msg.Nack(false, true); err != nil {
					c.logger.Error("Failed to nack message",
						zap.String("routing_key", c.routingKey),
						zap.Error(err),
					)
				}
				return
			}

			// Handler 成功 → 确认消息
			if err := msg.Ack(false); err != nil {
				c.logger.Error("Failed to ack message",
					zap.String("routing_key", c.routingKey),
					zap.Error(err),
				)
			} else {
				c.logger.Debug("Message processed successfully",
					zap.String("routing_key", c.routingKey),
					zap.String("queue", c.queue.Name),
				)
			}
		}()
		}
	}
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() {
	c.logger.Info("Stopping consumer...",
		zap.String("routing_key", c.routingKey),
	)
	c.cancel()
	<-c.stopChan
	c.logger.Info("Consumer stopped",
		zap.String("routing_key", c.routingKey),
	)
}

