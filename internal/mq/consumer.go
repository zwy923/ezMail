package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

type MessageHandler func(ctx context.Context, data json.RawMessage) error

type Consumer struct {
	channel    *amqp091.Channel
	queue      amqp091.Queue
	routingKey string
	handler    MessageHandler
	conn       *amqp091.Connection
}

// NewConsumer creates a consumer for a specific routing key.
// Each routing key gets its own queue, e.g., "email.received" -> "email.received.q"
func NewConsumer(url, queueName, routingKey string) (*Consumer, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// 声明 exchange
	err = ch.ExchangeDeclare(
		ExchangeName,
		"topic",
		true,  // durable
		false, // auto-delete
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	// 每个 worker 使用自己的 queueName（不可自动生成）
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	// 绑定队列到 exchange → 支持 fanout
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

	fmt.Printf("[RabbitMQ] Consumer started for routingKey=%s, queue=%s\n", routingKey, queueName)

	return &Consumer{
		conn:       conn,
		channel:    ch,
		queue:      q,
		routingKey: routingKey,
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

	msgs, err := c.channel.Consume(
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

	fmt.Printf("[RabbitMQ] Consumer started for routing key: %s, queue: %s\n", c.routingKey, c.queue.Name)

	// 直接在这里处理消息，不嵌套 goroutine
	for msg := range msgs {
		ctx := context.Background()

		// 直接使用消息体，不再解析 Event 结构
		if err := c.handler(ctx, msg.Body); err != nil {
			fmt.Printf("[Consumer] handler error for %s: %v\n", c.routingKey, err)
			// 处理失败，拒绝并重新入队
			_ = msg.Nack(false, true)
			continue
		}

		// 处理成功，确认消息
		if err := msg.Ack(false); err != nil {
			fmt.Printf("[Consumer] failed to ack message for %s: %v\n", c.routingKey, err)
		}
	}

	return nil
}
