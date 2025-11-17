package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type MessageHandler func(ctx context.Context, data json.RawMessage) error

type Consumer struct {
	channel    *amqp091.Channel
	queue      amqp091.Queue
	routingKey string
	handler    MessageHandler
	conn       *amqp091.Connection
	logger     *zap.Logger
}

// NewConsumer creates a consumer for a specific routing key.
func NewConsumer(url, queueName, routingKey string, logger *zap.Logger) (*Consumer, error) {
	conn, err := NewConnection(url)
	if err != nil {
		return nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	if err := DeclareExchange(ch); err != nil {
		ch.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

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

	logger.Info("Consumer initialized",
		zap.String("routing_key", routingKey),
		zap.String("queue", queueName),
		zap.String("exchange", ExchangeName),
	)

	return &Consumer{
		conn:       conn,
		channel:    ch,
		queue:      q,
		routingKey: routingKey,
		logger:     logger,
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
	for msg := range deliveries {
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

			// 执行业务处理
			if err := c.handler(ctx, msg.Body); err != nil {
				c.logger.Error("Handler error",
					zap.String("routing_key", c.routingKey),
					zap.String("queue", c.queue.Name),
					zap.Error(err),
				)
				// 业务失败 → 拒绝消息并重新入队，让 MQ 重试
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

	return nil
}

