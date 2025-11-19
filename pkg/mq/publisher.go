package mq

import (
	"context"
	"encoding/json"
	"fmt"

	"mygoproject/pkg/otel"
	"mygoproject/pkg/trace"

	"github.com/rabbitmq/amqp091-go"
	otelstd "go.opentelemetry.io/otel"
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
	return p.PublishWithContext(context.Background(), routingKey, payload)
}

// PublishWithContext publishes an event with trace_id from context.
func (p *Publisher) PublishWithContext(ctx context.Context, routingKey string, payload any) error {
	// 创建 OpenTelemetry span
	ctx, span := otel.MQPublishSpan(ctx, routingKey, ExchangeName)
	defer span.End()

	body, err := json.Marshal(payload)
	if err != nil {
		span.RecordError(err)
		return err
	}

	// 从 context 中提取 trace context 并注入到消息头
	headers := amqp091.Table{}
	propagator := otelstd.GetTextMapPropagator()
	carrier := otel.NewMQHeaderCarrier(headers)
	propagator.Inject(ctx, carrier)

	// 向后兼容：也设置 x-trace-id（如果存在）
	if traceID := trace.FromContext(ctx); traceID != "" {
		headers["x-trace-id"] = traceID
	}

	err = p.channel.Publish(
		ExchangeName,
		routingKey,
		false,
		false,
		amqp091.Publishing{
			ContentType:  "application/json",
			Body:         body,
			DeliveryMode: amqp091.Persistent,
			Headers:      headers,
		},
	)

	if err != nil {
		span.RecordError(err)
	}
	return err
}
