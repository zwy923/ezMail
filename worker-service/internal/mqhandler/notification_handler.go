package mqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"worker-service/internal/model"
	"worker-service/internal/repository"
	mqcontracts "mygoproject/contracts/mq"
	util "mygoproject/pkg/util"

	"go.uber.org/zap"
)

type EmailReceivedNotificationHandler struct {
	repo    *repository.NotificationRepository
	logger  *zap.Logger
	deduper *util.Deduper
}

func NewEmailReceivedNotificationHandler(
	repo *repository.NotificationRepository,
	logger *zap.Logger,
	deduper *util.Deduper,
) *EmailReceivedNotificationHandler {
	return &EmailReceivedNotificationHandler{
		repo:    repo,
		logger:  logger,
		deduper: deduper,
	}
}

// HandleEmailReceived -- 写入 notifications 站内通知
func (h *EmailReceivedNotificationHandler) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
	// Panic 恢复：确保 handler 是稳态的
	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("Panic in HandleEmailReceived",
				zap.Any("panic", r),
			)
		}
	}()

	var p mqcontracts.EmailReceivedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		// JSON decode 错误 - 不可重试
		h.logger.Error("Failed to unmarshal email received payload (non-retryable)",
			zap.Error(err),
		)
		return nil // 返回 nil，让 consumer ack 掉
	}

	// Redis 去重
	if !h.deduper.AcquireOnce(ctx, "notification", p.EmailID) {
		h.logger.Info("Duplicate notification event skipped",
			zap.Int("email_id", p.EmailID),
			zap.Int("user_id", p.UserID))
		return nil
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
		isRetryable, errType := util.IsRetryableError(err)
		h.logger.Error("Failed to insert notification",
			zap.Int("email_id", p.EmailID),
			zap.Int("user_id", p.UserID),
			zap.String("error_type", errType),
			zap.Bool("retryable", isRetryable),
			zap.Error(err),
		)
		if !isRetryable {
			return nil // 不可重试错误，ack 掉
		}
		return err // 可重试错误，nack 并重试
	}

	h.logger.Info("Notification created successfully",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
	)

	return nil
}

