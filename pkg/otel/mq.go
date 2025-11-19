package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// MQPublishSpan 在 MQ 发布时创建 span
func MQPublishSpan(ctx context.Context, routingKey string, exchange string) (context.Context, trace.Span) {
	tracer := Tracer()
	ctx, span := tracer.Start(ctx, "mq.publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", exchange),
			attribute.String("messaging.destination_kind", "exchange"),
			attribute.String("messaging.rabbitmq.routing_key", routingKey),
		),
	)

	// 将 trace context 注入到 context 中（用于后续提取到消息头）
	return ctx, span
}

// MQConsumeSpan 在 MQ 消费时创建 span
func MQConsumeSpan(ctx context.Context, routingKey string, queue string) (context.Context, trace.Span) {
	// 注意：trace context 应该在调用此函数之前从消息头中提取

	tracer := Tracer()
	ctx, span := tracer.Start(ctx, "mq.consume",
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "rabbitmq"),
			attribute.String("messaging.destination", queue),
			attribute.String("messaging.destination_kind", "queue"),
			attribute.String("messaging.rabbitmq.routing_key", routingKey),
		),
	)

	return ctx, span
}

// MQHeaderCarrier 实现 TextMapCarrier 接口，用于从 RabbitMQ 消息头中提取/注入 trace context
type MQHeaderCarrier struct {
	headers map[string]interface{}
}

func NewMQHeaderCarrier(headers map[string]interface{}) *MQHeaderCarrier {
	if headers == nil {
		headers = make(map[string]interface{})
	}
	return &MQHeaderCarrier{
		headers: headers,
	}
}

func (c *MQHeaderCarrier) Get(key string) string {
	if c.headers == nil {
		return ""
	}
	// OpenTelemetry 使用 traceparent 和 tracestate
	if key == "traceparent" || key == "tracestate" {
		if val, ok := c.headers[key]; ok {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

func (c *MQHeaderCarrier) Set(key, value string) {
	if c.headers == nil {
		c.headers = make(map[string]interface{})
	}
	c.headers[key] = value
}

func (c *MQHeaderCarrier) Keys() []string {
	return []string{"traceparent", "tracestate"}
}

