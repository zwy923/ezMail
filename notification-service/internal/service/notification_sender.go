package service

import (
	"context"
	"fmt"
	"time"

	"notification-service/internal/repository"
	mqcontracts "mygoproject/contracts/mq"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type NotificationSender struct {
	db         *pgxpool.Pool
	repo       *repository.NotificationRepository
	publisher  *mq.Publisher
	outboxRepo *outbox.Repository
	logger     *zap.Logger
}

func NewNotificationSender(
	db *pgxpool.Pool,
	repo *repository.NotificationRepository,
	publisher *mq.Publisher,
	logger *zap.Logger,
) *NotificationSender {
	return &NotificationSender{
		db:         db,
		repo:       repo,
		publisher:  publisher,
		outboxRepo: outbox.NewRepository(db),
		logger:     logger,
	}
}

// SendNotification sends a notification via the specified channel
func (s *NotificationSender) SendNotification(ctx context.Context, notificationID, userID int, channel, message string) error {
	s.logger.Info("Sending notification",
		zap.Int("notification_id", notificationID),
		zap.Int("user_id", userID),
		zap.String("channel", channel),
	)

	var err error
	switch channel {
	case "EMAIL":
		err = s.sendEmail(ctx, userID, message)
	case "PUSH":
		err = s.sendPush(ctx, userID, message)
	case "SMS":
		err = s.sendSMS(ctx, userID, message)
	case "WEBHOOK":
		err = s.sendWebhook(ctx, userID, message)
	default:
		err = fmt.Errorf("unsupported channel: %s", channel)
	}

	// 使用事务写入 Outbox 事件
	tx, txErr := s.db.Begin(ctx)
	if txErr != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(txErr))
		return txErr
	}
	defer tx.Rollback(ctx)

	if err != nil {
		s.logger.Error("Failed to send notification",
			zap.Int("notification_id", notificationID),
			zap.String("channel", channel),
			zap.Error(err),
		)
		
		// Insert notification.failed event to outbox (in transaction)
		payload := mqcontracts.NotificationFailedPayload{
			NotificationID: notificationID,
			UserID:         userID,
			Channel:        channel,
			Error:          err.Error(),
			RetryCount:     0,
		}
		notiID64 := int64(notificationID)
		if pubErr := outbox.InsertEventInTx(ctx, tx, s.outboxRepo, "notification", &notiID64, "notification.failed", payload); pubErr != nil {
			s.logger.Error("Failed to insert notification.failed to outbox", zap.Error(pubErr))
			return pubErr
		}

		if commitErr := tx.Commit(ctx); commitErr != nil {
			s.logger.Error("Failed to commit transaction", zap.Error(commitErr))
			return commitErr
		}
		return err
	}

	// Insert notification.sent event to outbox (in transaction)
	payload := mqcontracts.NotificationSentPayload{
		NotificationID: notificationID,
		UserID:         userID,
		Channel:        channel,
		SentAt:         time.Now(),
	}
	notiID64 := int64(notificationID)
	if err := outbox.InsertEventInTx(ctx, tx, s.outboxRepo, "notification", &notiID64, "notification.sent", payload); err != nil {
		s.logger.Error("Failed to insert notification.sent to outbox", zap.Error(err))
		return err
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	s.logger.Info("Notification sent successfully",
		zap.Int("notification_id", notificationID),
		zap.String("channel", channel),
	)
	return nil
}

func (s *NotificationSender) sendEmail(ctx context.Context, userID int, message string) error {
	// TODO: Implement email sending (SMTP, SendGrid, etc.)
	s.logger.Info("Sending email notification",
		zap.Int("user_id", userID),
		zap.String("message", message),
	)
	// Simulate email sending
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (s *NotificationSender) sendPush(ctx context.Context, userID int, message string) error {
	// TODO: Implement push notification (FCM, APNS, etc.)
	s.logger.Info("Sending push notification",
		zap.Int("user_id", userID),
		zap.String("message", message),
	)
	// Simulate push sending
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (s *NotificationSender) sendSMS(ctx context.Context, userID int, message string) error {
	// TODO: Implement SMS sending (Twilio, etc.)
	s.logger.Info("Sending SMS notification",
		zap.Int("user_id", userID),
		zap.String("message", message),
	)
	// Simulate SMS sending
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (s *NotificationSender) sendWebhook(ctx context.Context, userID int, message string) error {
	// TODO: Implement webhook (Slack, Telegram, etc.)
	s.logger.Info("Sending webhook notification",
		zap.Int("user_id", userID),
		zap.String("message", message),
	)
	// Simulate webhook sending
	time.Sleep(100 * time.Millisecond)
	return nil
}

