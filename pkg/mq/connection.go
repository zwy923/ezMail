package mq

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

const (
	ExchangeName = "events"
)

// NewConnection creates a new RabbitMQ connection.
func NewConnection(url string) (*amqp091.Connection, error) {
	conn, err := amqp091.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}
	return conn, nil
}

// DeclareExchange declares the events exchange.
func DeclareExchange(ch *amqp091.Channel) error {
	return ch.ExchangeDeclare(
		ExchangeName,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
}

