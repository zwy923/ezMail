package otel

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	httpServerDuration     metric.Float64Histogram
	httpServerRequestSize  metric.Int64Histogram
	httpServerResponseSize metric.Int64Histogram
)

// InitHTTPMetrics 初始化 HTTP 指标
func InitHTTPMetrics(meter metric.Meter) error {
	var err error

	httpServerDuration, err = meter.Float64Histogram(
		"http.server.duration",
		metric.WithDescription("HTTP server request duration"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	httpServerRequestSize, err = meter.Int64Histogram(
		"http.server.request.size",
		metric.WithDescription("HTTP server request size"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return err
	}

	httpServerResponseSize, err = meter.Int64Histogram(
		"http.server.response.size",
		metric.WithDescription("HTTP server response size"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return err
	}

	return nil
}

// HTTPMiddleware HTTP 追踪中间件
func HTTPMiddleware(next http.Handler) http.Handler {
	propagator := otel.GetTextMapPropagator()
	tracer := Tracer()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从请求头中提取 trace context
		ctx := propagator.Extract(r.Context(), propagation.HeaderCarrier(r.Header))

		// 创建 span
		spanName := r.Method + " " + r.URL.Path
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(r.Method),
				semconv.HTTPURLKey.String(r.URL.String()),
				semconv.HTTPRouteKey.String(r.URL.Path),
				attribute.String("http.user_agent", r.UserAgent()),
			),
		)
		defer span.End()

		// 记录请求大小
		if httpServerRequestSize != nil {
			httpServerRequestSize.Record(ctx, r.ContentLength)
		}

		// 包装 ResponseWriter 以捕获状态码和响应大小
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     200,
		}

		start := time.Now()
		next.ServeHTTP(wrapped, r.WithContext(ctx))
		duration := time.Since(start)

		// 记录响应属性
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(wrapped.statusCode),
			attribute.Int64("http.response.size", wrapped.responseSize),
		)

		// 记录指标
		if httpServerDuration != nil {
			httpServerDuration.Record(ctx, float64(duration.Milliseconds()),
				metric.WithAttributes(
					semconv.HTTPMethodKey.String(r.Method),
					semconv.HTTPRouteKey.String(r.URL.Path),
					semconv.HTTPStatusCodeKey.Int(wrapped.statusCode),
				),
			)
		}

		if httpServerResponseSize != nil {
			httpServerResponseSize.Record(ctx, wrapped.responseSize,
				metric.WithAttributes(
					semconv.HTTPMethodKey.String(r.Method),
					semconv.HTTPRouteKey.String(r.URL.Path),
					semconv.HTTPStatusCodeKey.Int(wrapped.statusCode),
				),
			)
		}

		// 设置错误状态
		if wrapped.statusCode >= 400 {
			span.SetStatus(codes.Error, "HTTP "+strconv.Itoa(wrapped.statusCode))
			span.SetAttributes(attribute.String("error", "true"))
		}

		// 将 trace context 注入响应头（用于客户端追踪）
		propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))
	})
}

// responseWriter 包装 http.ResponseWriter 以捕获状态码和响应大小
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	responseSize int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.responseSize += int64(n)
	return n, err
}

// GinMiddleware Gin 框架的 HTTP 追踪中间件
func GinMiddleware() func(c *gin.Context) {
	propagator := otel.GetTextMapPropagator()
	tracer := Tracer()

	return func(c *gin.Context) {
		// 从请求头中提取 trace context
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 创建 span
		spanName := c.Request.Method + " " + c.FullPath()
		if spanName == " " {
			spanName = c.Request.Method + " " + c.Request.URL.Path
		}
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.HTTPMethodKey.String(c.Request.Method),
				semconv.HTTPURLKey.String(c.Request.URL.String()),
				semconv.HTTPRouteKey.String(c.FullPath()),
				attribute.String("http.user_agent", c.Request.UserAgent()),
			),
		)
		defer span.End()

		// 记录请求大小
		if httpServerRequestSize != nil {
			httpServerRequestSize.Record(ctx, c.Request.ContentLength)
		}

		// 更新 context
		c.Request = c.Request.WithContext(ctx)

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// 记录响应属性
		statusCode := c.Writer.Status()
		responseSize := int64(c.Writer.Size())
		span.SetAttributes(
			semconv.HTTPStatusCodeKey.Int(statusCode),
			attribute.Int64("http.response.size", responseSize),
		)

		// 记录指标
		if httpServerDuration != nil {
			httpServerDuration.Record(ctx, float64(duration.Milliseconds()),
				metric.WithAttributes(
					semconv.HTTPMethodKey.String(c.Request.Method),
					semconv.HTTPRouteKey.String(c.FullPath()),
					semconv.HTTPStatusCodeKey.Int(statusCode),
				),
			)
		}

		if httpServerResponseSize != nil {
			httpServerResponseSize.Record(ctx, responseSize,
				metric.WithAttributes(
					semconv.HTTPMethodKey.String(c.Request.Method),
					semconv.HTTPRouteKey.String(c.FullPath()),
					semconv.HTTPStatusCodeKey.Int(statusCode),
				),
			)
		}

		// 设置错误状态
		if statusCode >= 400 {
			span.SetStatus(codes.Error, "HTTP "+strconv.Itoa(statusCode))
			span.SetAttributes(attribute.String("error", "true"))
		}

		// 将 trace context 注入响应头
		propagator.Inject(ctx, propagation.HeaderCarrier(c.Writer.Header()))
	}
}
