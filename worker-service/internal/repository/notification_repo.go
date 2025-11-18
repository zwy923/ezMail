package repository

import (
	"context"
	"worker-service/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// InsertFromDecision inserts a notification from AgentDecision
func (r *NotificationRepository) Insert(
	ctx context.Context,
	emailID int,
	userID int,
	decision *model.AgentDecision,
) error {
	sql := `
		INSERT INTO notifications (user_id, email_id, channel, message, is_read, created_at)
		VALUES ($1, $2, $3, $4, false, NOW());
	`

	_, err := r.db.Exec(ctx, sql,
		userID,
		emailID,
		decision.NotificationChannel,
		decision.NotificationMessage,
	)

	return err
}

// InsertSimple inserts a simple notification (for notification_handler)
func (r *NotificationRepository) InsertSimple(
	ctx context.Context,
	userID int,
	emailID int,
	channel string,
	message string,
) error {
	sql := `
		INSERT INTO notifications (user_id, email_id, channel, message, is_read, created_at)
		VALUES ($1, $2, $3, $4, false, NOW());
	`

	_, err := r.db.Exec(ctx, sql, userID, emailID, channel, message)
	return err
}
