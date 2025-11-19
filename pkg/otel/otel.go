package otel

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	tracer trace.Tracer
)

// Config OpenTelemetry 配置
type Config struct {
	ServiceName    string
	ServiceVersion string
	Endpoint       string // otel-collector endpoint (e.g., "otel-collector:4317")
	Enabled        bool
}

// Init 初始化 OpenTelemetry
func Init(cfg Config, logger *zap.Logger) (func(), error) {
	if !cfg.Enabled {
		logger.Info("OpenTelemetry is disabled")
		return func() {}, nil
	}

	if cfg.Endpoint == "" {
		cfg.Endpoint = "otel-collector:4317"
	}

	logger.Info("Initializing OpenTelemetry",
		zap.String("service", cfg.ServiceName),
		zap.String("endpoint", cfg.Endpoint),
	)

	// 创建资源
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(cfg.ServiceName),
			semconv.ServiceVersionKey.String(cfg.ServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建 OTLP exporter
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
		otlptracegrpc.WithInsecure(), // 开发环境使用，生产环境应使用 TLS
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // 开发环境采样所有，生产环境应使用概率采样
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(tp)

	// 设置全局 TextMapPropagator（用于跨服务传播 trace context）
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// 创建 tracer
	tracer = tp.Tracer(cfg.ServiceName)

	logger.Info("OpenTelemetry initialized successfully")

	// 返回清理函数
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown TracerProvider", zap.Error(err))
		}
	}, nil
}

// Tracer 返回全局 tracer
func Tracer() trace.Tracer {
	if tracer == nil {
		// 如果未初始化，返回 NoopTracer（避免 panic）
		return trace.NewNoopTracerProvider().Tracer("noop")
	}
	return tracer
}

// StartSpan 创建新的 span
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return Tracer().Start(ctx, name, opts...)
}

// GetTextMapPropagator 返回全局 TextMapPropagator
func GetTextMapPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}
