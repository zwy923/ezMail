package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Event 表示一个待发布的事件
type Event struct {
	ID            int64
	AggregateType string
	AggregateID   *int64
	RoutingKey    string
	Payload       json.RawMessage
	Status        string
	RetryCount    int
	NextRetryAt   *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Repository 提供 Outbox 操作的接口
type Repository struct {
	db *pgxpool.Pool
}

// NewRepository 创建新的 Outbox Repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// InsertEvent 在事务中插入事件到 outbox
// 必须在事务中调用，确保与业务数据的一致性
func (r *Repository) InsertEvent(ctx context.Context, tx pgx.Tx, event *Event) error {
	query := `
		INSERT INTO outbox_events (aggregate_type, aggregate_id, routing_key, payload, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, updated_at
	`

	err := tx.QueryRow(ctx, query,
		event.AggregateType,
		event.AggregateID,
		event.RoutingKey,
		event.Payload,
		event.Status,
	).Scan(&event.ID, &event.CreatedAt, &event.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to insert outbox event: %w", err)
	}

	return nil
}

// GetPendingEvents 获取待发送的事件（用于 Dispatcher）
func (r *Repository) GetPendingEvents(ctx context.Context, limit int) ([]*Event, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, routing_key, payload, status, 
		       retry_count, next_retry_at, created_at, updated_at
		FROM outbox_events
		WHERE status = 'pending'
		AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		err := rows.Scan(
			&e.ID,
			&e.AggregateType,
			&e.AggregateID,
			&e.RoutingKey,
			&e.Payload,
			&e.Status,
			&e.RetryCount,
			&e.NextRetryAt,
			&e.CreatedAt,
			&e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &e)
	}

	return events, rows.Err()
}

// MarkAsSent 标记事件为已发送
func (r *Repository) MarkAsSent(ctx context.Context, eventID int64) error {
	query := `
		UPDATE outbox_events
		SET status = 'sent', updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as sent: %w", err)
	}

	return nil
}

// MarkAsFailed 标记事件为失败，并设置重试时间
func (r *Repository) MarkAsFailed(ctx context.Context, eventID int64, maxRetries int) error {
	// 先获取当前重试次数
	var retryCount int
	err := r.db.QueryRow(ctx, `
		SELECT retry_count FROM outbox_events WHERE id = $1
	`, eventID).Scan(&retryCount)

	if err != nil {
		return fmt.Errorf("failed to get retry count: %w", err)
	}

	retryCount++

	var status string
	var nextRetryAt *time.Time
	if retryCount >= maxRetries {
		status = "failed"
		nextRetryAt = nil // 失败后不再重试
	} else {
		status = "pending"
		nextRetry := time.Now().Add(time.Duration(retryCount) * 5 * time.Second) // 指数退避：5s, 10s, 15s...
		nextRetryAt = &nextRetry
	}

	query := `
		UPDATE outbox_events
		SET status = $1, retry_count = $2, next_retry_at = $3, updated_at = NOW()
		WHERE id = $4
	`

	_, err = r.db.Exec(ctx, query, status, retryCount, nextRetryAt, eventID)
	if err != nil {
		return fmt.Errorf("failed to mark event as failed: %w", err)
	}

	return nil
}

// GetEventByID 根据 ID 获取事件（用于 Replay）
func (r *Repository) GetEventByID(ctx context.Context, eventID int64) (*Event, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, routing_key, payload, status,
		       retry_count, next_retry_at, created_at, updated_at
		FROM outbox_events
		WHERE id = $1
	`

	var e Event
	err := r.db.QueryRow(ctx, query, eventID).Scan(
		&e.ID,
		&e.AggregateType,
		&e.AggregateID,
		&e.RoutingKey,
		&e.Payload,
		&e.Status,
		&e.RetryCount,
		&e.NextRetryAt,
		&e.CreatedAt,
		&e.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("event not found: %d", eventID)
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	return &e, nil
}

// ReplayEvent 重放事件（将状态重置为 pending）
func (r *Repository) ReplayEvent(ctx context.Context, eventID int64) error {
	query := `
		UPDATE outbox_events
		SET status = 'pending', retry_count = 0, next_retry_at = NULL, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, eventID)
	if err != nil {
		return fmt.Errorf("failed to replay event: %w", err)
	}

	return nil
}

// GetFailedEvents 获取所有失败的事件（用于管理界面）
func (r *Repository) GetFailedEvents(ctx context.Context, limit int) ([]*Event, error) {
	query := `
		SELECT id, aggregate_type, aggregate_id, routing_key, payload, status,
		       retry_count, next_retry_at, created_at, updated_at
		FROM outbox_events
		WHERE status = 'failed'
		ORDER BY created_at DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query failed events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var e Event
		err := rows.Scan(
			&e.ID,
			&e.AggregateType,
			&e.AggregateID,
			&e.RoutingKey,
			&e.Payload,
			&e.Status,
			&e.RetryCount,
			&e.NextRetryAt,
			&e.CreatedAt,
			&e.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &e)
	}

	return events, rows.Err()
}
