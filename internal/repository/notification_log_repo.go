package repository

import (
    "context"
    "mygoproject/internal/model"

    "github.com/jackc/pgx/v5/pgxpool"
)

type NotificationLogRepository struct {
    db *pgxpool.Pool
}

func NewNotificationLogRepository(db *pgxpool.Pool) *NotificationLogRepository {
    return &NotificationLogRepository{db: db}
}

func (r *NotificationLogRepository) Insert(ctx context.Context, log *model.NotificationLog) error {
    query := `
        INSERT INTO notifications_log (user_id, email_id, message, created_at)
        VALUES ($1, $2, $3, NOW())
    `
    _, err := r.db.Exec(ctx, query, log.UserID, log.EmailID, log.Message)
    return err
}
