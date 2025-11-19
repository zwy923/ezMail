package mqhandler

import (
	"context"
	"encoding/json"

	"notification-service/internal/repository"
	"notification-service/internal/service"
	mqcontracts "mygoproject/contracts/mq"

	"go.uber.org/zap"
)

type NotificationCreatedHandler struct {
	repo            *repository.NotificationRepository
	notificationSender *service.NotificationSender
	logger          *zap.Logger
}

func NewNotificationCreatedHandler(
	repo *repository.NotificationRepository,
	sender *service.NotificationSender,
	logger *zap.Logger,
) *NotificationCreatedHandler {
	return &NotificationCreatedHandler{
		repo:            repo,
		notificationSender: sender,
		logger:          logger,
	}
}

func (h *NotificationCreatedHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var p mqcontracts.NotificationCreatedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal NotificationCreatedPayload", zap.Error(err))
		return err
	}

	h.logger.Info("Handling notification.created event",
		zap.Int("user_id", p.UserID),
		zap.String("channel", p.Channel),
	)

	// Insert notification to database
	notificationID, err := h.repo.Insert(ctx, p.UserID, p.EmailID, p.Channel, p.Message)
	if err != nil {
		h.logger.Error("Failed to insert notification", zap.Error(err))
		return err
	}

	// Send notification
	if err := h.notificationSender.SendNotification(ctx, notificationID, p.UserID, p.Channel, p.Message); err != nil {
		h.logger.Error("Failed to send notification", zap.Error(err))
		// Don't return error - notification is already saved, can retry later
	}

	return nil
}

