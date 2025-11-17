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
		// JSON decode 错误 - 不可重试，发送到 DLQ
		h.logger.Error("Failed to unmarshal email received payload (non-retryable, sending to DLQ)",
			zap.Error(err),
			zap.String("raw_payload", string(raw)),
		)
		return fmt.Errorf("json_unmarshal_error: %w", err)
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
		isRetryable, errType := util.IsRetryableError(err)
		h.logger.Error("Failed to insert notification log",
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

	h.logger.Info("Notification log created successfully",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
	)

	return nil
}

