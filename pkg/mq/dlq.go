package mq

import (
	"fmt"

	"github.com/rabbitmq/amqp091-go"
)

const (
	DLQExchangeName = "email.received.dlq"
)

// DeclareDLQExchange declares the dead letter exchange.
func DeclareDLQExchange(ch *amqp091.Channel) error {
	return ch.ExchangeDeclare(
		DLQExchangeName,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
}

// DeclareDLQQueue declares a dead letter queue for a specific routing key.
func DeclareDLQQueue(ch *amqp091.Channel, routingKey string) (amqp091.Queue, error) {
	queueName := fmt.Sprintf("%s.dlq", routingKey)
	
	q, err := ch.QueueDeclare(
		queueName,
		true,  // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return amqp091.Queue{}, fmt.Errorf("failed to declare DLQ queue: %w", err)
	}

	// Bind queue to DLQ exchange
	err = ch.QueueBind(
		q.Name,
		routingKey,
		DLQExchangeName,
		false,
		nil,
	)
	if err != nil {
		return amqp091.Queue{}, fmt.Errorf("failed to bind DLQ queue: %w", err)
	}

	return q, nil
}

// PublishToDLQ publishes a message to the dead letter queue.
func (p *Publisher) PublishToDLQ(routingKey string, payload []byte, originalError string) error {
	// Add error information to message headers
	headers := amqp091.Table{
		"x-original-error": originalError,
		"x-failed-at":      "worker-service",
	}

	return p.channel.Publish(
		DLQExchangeName,
		routingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         payload,
			DeliveryMode: amqp091.Persistent,
			Headers:      headers,
		},
	)
}

