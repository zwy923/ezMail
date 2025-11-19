package logger

import (
	"context"

	"go.uber.org/zap"
	"mygoproject/pkg/trace"
)

var Log *zap.Logger

func NewLogger() *zap.Logger {
	l, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	Log = l
	return l
}

// WithTrace 从 context 中提取 trace_id 并添加到 logger
func WithTrace(ctx context.Context, logger *zap.Logger) *zap.Logger {
	traceID := trace.FromContext(ctx)
	if traceID != "" {
		return logger.With(zap.String("trace_id", traceID))
	}
	return logger
}

