package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FailedEventRepository struct {
	db *pgxpool.Pool
}

func NewFailedEventRepository(db *pgxpool.Pool) *FailedEventRepository {
	return &FailedEventRepository{db: db}
}

// InsertFailedEvent 插入失败的事件记录
func (r *FailedEventRepository) InsertFailedEvent(
	ctx context.Context,
	emailID, userID int,
	eventType, routingKey string,
	payload interface{},
	errorMsg string,
) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO failed_events (email_id, user_id, event_type, routing_key, payload, error_message, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
		ON CONFLICT DO NOTHING
	`
	_, err = r.db.Exec(ctx, query, emailID, userID, eventType, routingKey, payloadJSON, errorMsg)
	return err
}

// GetPendingEvents 获取待重试的事件
func (r *FailedEventRepository) GetPendingEvents(ctx context.Context, limit int) ([]FailedEvent, error) {
	query := `
		SELECT id, email_id, user_id, event_type, routing_key, payload, error_message, retry_count, status
		FROM failed_events
		WHERE status = 'pending' AND retry_count < 3
		ORDER BY created_at ASC
		LIMIT $1
	`
	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []FailedEvent
	for rows.Next() {
		var e FailedEvent
		if err := rows.Scan(&e.ID, &e.EmailID, &e.UserID, &e.EventType, &e.RoutingKey, &e.Payload, &e.ErrorMessage, &e.RetryCount, &e.Status); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// MarkAsRetried 标记事件为已重试
func (r *FailedEventRepository) MarkAsRetried(ctx context.Context, id int) error {
	query := `
		UPDATE failed_events
		SET status = 'retried', retry_count = retry_count + 1, updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// MarkAsFailed 标记事件为最终失败（超过最大重试次数）
func (r *FailedEventRepository) MarkAsFailed(ctx context.Context, id int) error {
	query := `
		UPDATE failed_events
		SET status = 'failed', updated_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

type FailedEvent struct {
	ID          int
	EmailID     int
	UserID      int
	EventType   string
	RoutingKey  string
	Payload     json.RawMessage
	ErrorMessage string
	RetryCount  int
	Status      string
}

