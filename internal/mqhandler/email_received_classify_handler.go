package mqhandler

import (
	"context"
	"encoding/json"
	"mygoproject/internal/mq"
	"mygoproject/internal/repository"
	"strings"

	"go.uber.org/zap"
)

type EmailReceivedClassifyHandler struct {
	emailRepo    *repository.EmailRepository
	metadataRepo *repository.MetadataRepository
	logger       *zap.Logger
}

func NewEmailReceivedClassifyHandler(emailRepo *repository.EmailRepository, metadataRepo *repository.MetadataRepository, logger *zap.Logger) *EmailReceivedClassifyHandler {
	return &EmailReceivedClassifyHandler{
		emailRepo:    emailRepo,
		metadataRepo: metadataRepo,
		logger:       logger,
	}
}

// HandleEmailReceived processes an EmailReceivedEvent and stores classification.
// This method is idempotent: calling it multiple times with the same event
// will have the same effect as calling it once.
// Optimized to use a single database query to check both email status and metadata existence.
func (h *EmailReceivedClassifyHandler) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
	var p mq.EmailReceivedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal email received payload", zap.Error(err))
		return err
	}

	h.logger.Info("Processing email classification",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
		zap.String("subject", p.Subject),
	)

	// 一次性查询：获取 email 和 metadata 是否存在（单次 round trip）
	email, metadataExists, err := h.emailRepo.FindRawWithMetadataByID(ctx, p.EmailID)
	if err != nil {
		h.logger.Error("Failed to find email", zap.Int("email_id", p.EmailID), zap.Error(err))
		return err
	}

	// 幂等性检查：如果已经分类，直接返回
	if email.Status == "classified" {
		h.logger.Debug("Email already classified, skipping",
			zap.Int("email_id", p.EmailID),
		)
		return nil
	}

	// 如果 metadata 已存在，只需更新状态
	if metadataExists {
		if email.Status != "classified" {
			if err := h.emailRepo.UpdateStatus(ctx, p.EmailID, "classified"); err != nil {
				h.logger.Error("Failed to update email status", zap.Int("email_id", p.EmailID), zap.Error(err))
				return err
			}
		}
		h.logger.Debug("Metadata already exists, status updated",
			zap.Int("email_id", p.EmailID),
		)
		return nil
	}

	// 执行分类逻辑
	var category string
	subj := strings.ToLower(p.Subject)
	switch {
	case strings.Contains(subj, "payment"):
		category = "finance"
	case strings.Contains(subj, "meeting"):
		category = "schedule"
	default:
		category = "other"
	}

	h.logger.Info("Classifying email",
		zap.Int("email_id", p.EmailID),
		zap.String("category", category),
		zap.String("subject", p.Subject),
	)

	// 插入metadata（使用 ON CONFLICT 保证幂等性）
	if err := h.metadataRepo.Insert(ctx, p.EmailID, category, 1.0); err != nil {
		h.logger.Error("Failed to insert metadata", zap.Int("email_id", p.EmailID), zap.Error(err))
		return err
	}

	// 更新状态
	if err := h.emailRepo.UpdateStatus(ctx, p.EmailID, "classified"); err != nil {
		h.logger.Error("Failed to update email status", zap.Int("email_id", p.EmailID), zap.Error(err))
		return err
	}

	h.logger.Info("Email classified successfully",
		zap.Int("email_id", p.EmailID),
		zap.String("category", category),
	)

	return nil
}
