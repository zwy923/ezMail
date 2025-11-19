package service

import (
	"context"
	"fmt"
	"time"

	"notification-service/internal/repository"
	"mygoproject/pkg/mq"

	"go.uber.org/zap"
)

type NotificationSender struct {
	repo     *repository.NotificationRepository
	publisher *mq.Publisher
	logger   *zap.Logger
}

func NewNotificationSender(
	repo *repository.NotificationRepository,
	publisher *mq.Publisher,
	logger *zap.Logger,
) *NotificationSender {
	return &NotificationSender{
		repo:      repo,
		publisher: publisher,
		logger:    logger,
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

	if err != nil {
		s.logger.Error("Failed to send notification",
			zap.Int("notification_id", notificationID),
			zap.String("channel", channel),
			zap.Error(err),
		)
		
		// Publish notification.failed event
		payload := map[string]interface{}{
			"notification_id": notificationID,
			"user_id":         userID,
			"channel":         channel,
			"error":           err.Error(),
			"retry_count":     0,
		}
		if pubErr := s.publisher.Publish("notification.failed", payload); pubErr != nil {
			s.logger.Error("Failed to publish notification.failed event", zap.Error(pubErr))
		}
		return err
	}

	// Publish notification.sent event
	payload := map[string]interface{}{
		"notification_id": notificationID,
		"user_id":         userID,
		"channel":         channel,
		"sent_at":         time.Now(),
	}
	if err := s.publisher.Publish("notification.sent", payload); err != nil {
		s.logger.Error("Failed to publish notification.sent event", zap.Error(err))
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

