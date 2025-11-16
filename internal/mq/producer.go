package mq

import (
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "events"
)

type Producer struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

func NewProducer(url string) (*Producer, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// 声明 topic exchange
	err = ch.ExchangeDeclare(
		ExchangeName,
		"topic", // topic exchange 支持 routing key 模式匹配
		true,    // durable
		false,   // auto-deleted
		false,   // internal
		false,   // no-wait
		nil,     // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare exchange: %w", err)
	}

	return &Producer{
		conn:    conn,
		channel: ch,
	}, nil
}

func (p *Producer) Close() {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}

// Publish publishes an event to the exchange with the given routing key.
// routingKey: e.g., "email.received", "user.registered"
func (p *Producer) Publish(routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.channel.Publish(
		ExchangeName, // exchange
		routingKey,   // routing key
		false,        // mandatory
		false,        // immediate
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
		},
	)
}
