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

func (r *NotificationRepository) Insert(ctx context.Context, n *model.Notification) error {
	query := `
        INSERT INTO notifications (user_id, type, content, is_read, created_at)
        VALUES ($1, $2, $3, false, NOW())
    `
	_, err := r.db.Exec(ctx, query, n.UserID, n.Type, n.Content)
	return err
}

