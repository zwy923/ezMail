package otel

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// DBSpan 为数据库操作创建 span
func DBSpan(ctx context.Context, operation string, query string) (context.Context, trace.Span) {
	tracer := Tracer()
	ctx, span := tracer.Start(ctx, "db."+operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			semconv.DBSystemKey.String("postgresql"),
			semconv.DBOperationKey.String(operation),
			attribute.String("db.statement", query),
		),
	)
	return ctx, span
}

// DBQueryRow 包装 pgx QueryRow 操作
func DBQueryRow(ctx context.Context, operation string, query string, fn func(context.Context) error) error {
	ctx, span := DBSpan(ctx, operation, query)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}

// DBExec 包装 pgx Exec 操作
func DBExec(ctx context.Context, operation string, query string, fn func(context.Context) error) error {
	ctx, span := DBSpan(ctx, operation, query)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}

// DBQuery 包装 pgx Query 操作
func DBQuery(ctx context.Context, operation string, query string, fn func(context.Context) error) error {
	ctx, span := DBSpan(ctx, operation, query)
	defer span.End()

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}

// WrapDBError 记录数据库错误到 span
func WrapDBError(span trace.Span, err error) {
	if err != nil {
		if err == sql.ErrNoRows || err == pgx.ErrNoRows {
			span.SetStatus(codes.Ok, "no rows")
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	} else {
		span.SetStatus(codes.Ok, "")
	}
}

// QueryRow 包装 pgx QueryRow 操作，自动添加追踪
func QueryRow(ctx context.Context, operation string, query string, fn func(context.Context) error) error {
	ctx, span := DBSpan(ctx, operation, query)
	defer span.End()

	err := fn(ctx)
	WrapDBError(span, err)
	return err
}

// Exec 包装 pgx Exec 操作，自动添加追踪
func Exec(ctx context.Context, operation string, query string, fn func(context.Context) error) error {
	ctx, span := DBSpan(ctx, operation, query)
	defer span.End()

	err := fn(ctx)
	WrapDBError(span, err)
	return err
}

// Query 包装 pgx Query 操作，自动添加追踪
func Query(ctx context.Context, operation string, query string, fn func(context.Context) error) error {
	ctx, span := DBSpan(ctx, operation, query)
	defer span.End()

	err := fn(ctx)
	WrapDBError(span, err)
	return err
}

