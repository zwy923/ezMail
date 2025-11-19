package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type NotificationRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewNotificationRepository(db *pgxpool.Pool, logger *zap.Logger) *NotificationRepository {
	return &NotificationRepository{
		db:     db,
		logger: logger,
	}
}

func (r *NotificationRepository) Insert(ctx context.Context, userID, emailID int, channel, message string) (int, error) {
	r.logger.Debug("Inserting notification",
		zap.Int("user_id", userID),
		zap.Int("email_id", emailID),
		zap.String("channel", channel),
	)

	query := `
        INSERT INTO notifications (user_id, email_id, channel, message)
        VALUES ($1, $2, $3, $4)
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query, userID, emailID, channel, message).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to insert notification", zap.Error(err))
		return 0, err
	}

	r.logger.Info("Notification inserted successfully",
		zap.Int("id", id),
		zap.Int("user_id", userID),
	)
	return id, nil
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id int) error {
	query := `UPDATE notifications SET is_read = TRUE WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *NotificationRepository) GetByID(ctx context.Context, id int) (*Notification, error) {
	query := `
        SELECT id, user_id, email_id, channel, message, is_read, created_at
        FROM notifications
        WHERE id = $1
    `
	var n Notification
	err := r.db.QueryRow(ctx, query, id).Scan(
		&n.ID,
		&n.UserID,
		&n.EmailID,
		&n.Channel,
		&n.Message,
		&n.IsRead,
		&n.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &n, nil
}

type Notification struct {
	ID        int
	UserID    int
	EmailID   int
	Channel   string
	Message   string
	IsRead    bool
	CreatedAt interface{}
}

