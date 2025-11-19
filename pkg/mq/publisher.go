package mq

import (
	"encoding/json"
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	conn    *amqp091.Connection
	channel *amqp091.Channel
}

func NewPublisher(url string) (*Publisher, error) {
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

	return &Publisher{
		conn:    conn,
		channel: ch,
	}, nil
}

func (p *Publisher) Close() {
	if p.channel != nil {
		_ = p.channel.Close()
	}
	if p.conn != nil {
		_ = p.conn.Close()
	}
}

// IsConnected checks if the publisher connection is still alive
func (p *Publisher) IsConnected() bool {
	if p.conn == nil || p.channel == nil {
		return false
	}
	// Check if connection is closed
	if p.conn.IsClosed() {
		return false
	}
	return true
}

// Publish publishes an event to the exchange with the given routing key.
func (p *Publisher) Publish(routingKey string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return p.channel.Publish(
		ExchangeName,
		routingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
		},
	)
}

