package mqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"mygoproject/internal/model"
	"mygoproject/internal/mq"
	"mygoproject/internal/repository"

	"go.uber.org/zap"
)

type EmailReceivedNotificationLogHandler struct {
	repo   *repository.NotificationLogRepository
	logger *zap.Logger
}

func NewEmailReceivedNotificationLogHandler(repo *repository.NotificationLogRepository, logger *zap.Logger) *EmailReceivedNotificationLogHandler {
	return &EmailReceivedNotificationLogHandler{
		repo:   repo,
		logger: logger,
	}
}

// HandleEmailReceived -- 写入 notifications_log
func (h *EmailReceivedNotificationLogHandler) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
	var p mq.EmailReceivedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal email received payload", zap.Error(err))
		return err
	}

	h.logger.Info("Creating notification log",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
	)

	log := &model.NotificationLog{
		UserID:  p.UserID,
		EmailID: p.EmailID,
		Message: fmt.Sprintf("User %d received a new email %d", p.UserID, p.EmailID),
	}

	if err := h.repo.Insert(ctx, log); err != nil {
		h.logger.Error("Failed to insert notification log",
			zap.Int("email_id", p.EmailID),
			zap.Int("user_id", p.UserID),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("Notification log created successfully",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
	)

	return nil
}
