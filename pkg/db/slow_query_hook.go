package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"mygoproject/pkg/metrics"
)

// SlowQueryTracer 慢查询监控 Tracer
type SlowQueryTracer struct {
	logger        *zap.Logger
	slowThreshold time.Duration // 慢查询阈值，默认 100ms
}

// NewSlowQueryTracer 创建慢查询 Tracer
func NewSlowQueryTracer(logger *zap.Logger, slowThreshold time.Duration) *SlowQueryTracer {
	if slowThreshold == 0 {
		slowThreshold = 100 * time.Millisecond
	}
	return &SlowQueryTracer{
		logger:        logger,
		slowThreshold: slowThreshold,
	}
}

// TraceQueryStart 查询开始时的钩子
func (t *SlowQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	// 在 context 中存储查询开始时间和 SQL
	ctx = context.WithValue(ctx, "query_start_time", time.Now())
	ctx = context.WithValue(ctx, "query_sql", data.SQL)
	return ctx
}

// TraceQueryEnd 查询结束时的钩子
func (t *SlowQueryTracer) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	// 获取查询开始时间
	startTime, ok := ctx.Value("query_start_time").(time.Time)
	if !ok {
		return
	}

	duration := time.Since(startTime)

	// 如果查询时间超过阈值，记录警告日志和指标
	if duration > t.slowThreshold {
		// 从 TraceQueryStartData 获取 SQL（需要通过 context 传递）
		// 注意：pgx v5 的 TraceQueryEndData 不包含 SQL，需要从 context 获取
		sql := t.getSQLFromContext(ctx)
		if sql == "" {
			sql = "unknown"
		}

		// 截断 SQL 语句（避免日志过长）
		sqlTruncated := sql
		if len(sqlTruncated) > 200 {
			sqlTruncated = sqlTruncated[:200] + "..."
		}

		// 记录警告日志
		t.logger.Warn("slow-query",
			zap.String("sql", sqlTruncated),
			zap.Duration("took", duration),
			zap.String("command_tag", data.CommandTag.String()),
		)

		// 记录 Prometheus 指标
		metrics.IncrementSlowQuery(sqlTruncated, duration)
	}
}

// getSQLFromContext 从 context 中获取 SQL（如果之前存储过）
func (t *SlowQueryTracer) getSQLFromContext(ctx context.Context) string {
	if sql, ok := ctx.Value("query_sql").(string); ok {
		return sql
	}
	return ""
}

