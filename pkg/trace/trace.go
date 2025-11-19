package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

const TraceIDKey = "trace_id"

// GenerateTraceID 生成一个新的 trace ID
func GenerateTraceID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// FromContext 从 context 中获取 trace_id
func FromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// WithContext 将 trace_id 添加到 context 中
func WithContext(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// FromHeader 从 HTTP header 中提取 trace_id（支持 X-Trace-ID 和 X-Request-ID）
func FromHeader(headerValue string) string {
	if headerValue != "" {
		return headerValue
	}
	return ""
}

// HeaderName 返回 trace ID 的 HTTP header 名称
func HeaderName() string {
	return "X-Trace-ID"
}
