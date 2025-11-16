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

type EmailReceivedNotificationHandler struct {
	repo   *repository.NotificationRepository
	logger *zap.Logger
}

func NewEmailReceivedNotificationHandler(repo *repository.NotificationRepository, logger *zap.Logger) *EmailReceivedNotificationHandler {
	return &EmailReceivedNotificationHandler{
		repo:   repo,
		logger: logger,
	}
}

// HandleEmailReceived -- 写入 notifications 站内通知
func (h *EmailReceivedNotificationHandler) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
	var p mq.EmailReceivedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal email received payload", zap.Error(err))
		return err
	}

	h.logger.Info("Creating notification",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
		zap.String("subject", p.Subject),
	)

	notif := &model.Notification{
		UserID:  p.UserID,
		Type:    "new_email",
		Content: fmt.Sprintf("你收到了新邮件：%s", p.Subject),
	}

	if err := h.repo.Insert(ctx, notif); err != nil {
		h.logger.Error("Failed to insert notification",
			zap.Int("email_id", p.EmailID),
			zap.Int("user_id", p.UserID),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("Notification created successfully",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
	)

	return nil
}
