package mqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mqcontracts "mygoproject/contracts/mq"
	util "mygoproject/pkg/util"
	"mygoproject/pkg/mq"

	"go.uber.org/zap"
)

type EmailReceivedNotificationHandler struct {
	publisher *mq.Publisher
	logger    *zap.Logger
	deduper   *util.Deduper
}

func NewEmailReceivedNotificationHandler(
	publisher *mq.Publisher,
	logger *zap.Logger,
	deduper *util.Deduper,
) *EmailReceivedNotificationHandler {
	return &EmailReceivedNotificationHandler{
		publisher: publisher,
		logger:    logger,
		deduper:   deduper,
	}
}

// HandleEmailReceived -- 发布 notification.created 事件（通知由 notification-service 处理）
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
		// JSON decode 错误 - 不可重试，发送到 DLQ
		h.logger.Error("Failed to unmarshal email received payload (non-retryable, sending to DLQ)",
			zap.Error(err),
			zap.String("raw_payload", string(raw)),
		)
		return fmt.Errorf("json_unmarshal_error: %w", err)
	}

	// Redis 去重：确保不重复发布通知事件
	if !h.deduper.AcquireOnce(ctx, "notification", p.EmailID) {
		h.logger.Info("Skipped duplicated event",
			zap.String("handler", "notification"),
			zap.Int("email_id", p.EmailID),
			zap.Int("user_id", p.UserID),
		)
		return nil
	}

	h.logger.Info("Publishing notification.created event",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
		zap.String("subject", p.Subject),
	)

	// Publish notification.created event
	notiPayload := map[string]interface{}{
		"user_id":    p.UserID,
		"email_id":   p.EmailID,
		"channel":    "EMAIL",
		"message":    fmt.Sprintf("你收到了新邮件：%s", p.Subject),
		"created_at": time.Now(),
	}

	if err := h.publisher.Publish("notification.created", notiPayload); err != nil {
		h.logger.Error("Failed to publish notification.created event",
			zap.Int("email_id", p.EmailID),
			zap.Int("user_id", p.UserID),
			zap.Error(err),
		)
		return err
	}

	h.logger.Info("Notification.created event published successfully",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
	)

	return nil
}

