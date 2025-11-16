package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

type MessageHandler func(ctx context.Context, event Event) error

type Consumer struct {
	channel *amqp091.Channel
	queue   amqp091.Queue
	handler MessageHandler
	conn    *amqp091.Connection
}

func NewConsumer(url string) (*Consumer, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	q, err := ch.QueueDeclare(
		"events", // 通用事件队列！
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &Consumer{
		conn:    conn,
		channel: ch,
		queue:   q,
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

	fmt.Println("[RabbitMQ] Consumer started, waiting for messages...")

	go func() {
		for msg := range msgs {
			var evt Event
			if err := json.Unmarshal(msg.Body, &evt); err != nil {
				fmt.Println("[Consumer] invalid message:", err)
				// 无效消息，拒绝且不重新入队
				_ = msg.Nack(false, false)
				continue
			}

			ctx := context.Background()
			if err := c.handler(ctx, evt); err != nil {
				fmt.Println("[Consumer] handler error:", err)
				// 处理失败，拒绝并重新入队
				_ = msg.Nack(false, true)
				continue
			}

			// 处理成功，确认消息
			if err := msg.Ack(false); err != nil {
				fmt.Println("[Consumer] failed to ack message:", err)
			}
		}
	}()

	return nil
}
