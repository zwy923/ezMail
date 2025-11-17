package mqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	mqcontracts "mygoproject/contracts/mq"
	"mygoproject/pkg/util"
	"strings"
	"worker-service/internal/repository"
	"worker-service/internal/service"

	"go.uber.org/zap"
)

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

const (
	maxRetries = 5 // 最大重试次数
)

type EmailReceivedClassifyHandler struct {
	emailRepo    *repository.EmailRepository
	metadataRepo *repository.MetadataRepository
	agentClient  *service.AgentClient
	retryCounter *util.RetryCounter
	deduper      *util.Deduper
	logger       *zap.Logger
}

func NewEmailReceivedClassifyHandler(
	emailRepo *repository.EmailRepository,
	metadataRepo *repository.MetadataRepository,
	agentClient *service.AgentClient,
	retryCounter *util.RetryCounter,
	deduper *util.Deduper,
	logger *zap.Logger,
) *EmailReceivedClassifyHandler {
	return &EmailReceivedClassifyHandler{
		emailRepo:    emailRepo,
		metadataRepo: metadataRepo,
		agentClient:  agentClient,
		retryCounter: retryCounter,
		deduper:      deduper,
		logger:       logger,
	}
}

// HandleEmailReceived processes an EmailReceivedEvent and stores classification.
// This method is idempotent and handles retries with max retry limit.
// Returns error only for retryable errors that haven't exceeded max retries.
func (h *EmailReceivedClassifyHandler) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
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
		// 返回特殊错误，让 consumer 发送到 DLQ
		return fmt.Errorf("json_unmarshal_error: %w", err)
	}

	h.logger.Info("Processing email classification",
		zap.Int("email_id", p.EmailID),
		zap.Int("user_id", p.UserID),
		zap.String("subject", p.Subject),
	)

	// 一次性查询：获取 email 和 metadata 是否存在（单次 round trip）
	email, metadataExists, err := h.emailRepo.FindRawWithMetadataByID(ctx, p.EmailID)
	if err != nil {
		// 检查错误类型
		isRetryable, errType := util.IsRetryableError(err)
		h.logger.Error("Failed to find email",
			zap.Int("email_id", p.EmailID),
			zap.String("error_type", errType),
			zap.Bool("retryable", isRetryable),
			zap.Error(err),
		)

		if !isRetryable {
			// 不可重试错误（如 email_not_found）- 直接返回 nil，ack 掉
			return nil
		}
		// 可重试错误（如 DB 连接问题）- 返回 error，让 consumer nack
		return err
	}

	// 幂等性检查：如果已经分类，直接返回
	if email.Status == "classified" {
		h.logger.Debug("Email already classified, skipping",
			zap.Int("email_id", p.EmailID),
		)
		return nil
	}

	// Redis 去重：避免重复处理（减少 DB UNIQUE 错误日志）
	if !h.deduper.AcquireOnce(ctx, "classify", p.EmailID) {
		h.logger.Info("Skipped duplicated classify event",
			zap.Int("email_id", p.EmailID),
			zap.Int("user_id", p.UserID),
		)
		// 如果 metadata 已存在，只需更新状态
		if metadataExists && email.Status != "classified" {
			if err := h.emailRepo.UpdateStatus(ctx, p.EmailID, "classified"); err != nil {
				isRetryable, errType := util.IsRetryableError(err)
				h.logger.Error("Failed to update email status",
					zap.Int("email_id", p.EmailID),
					zap.String("error_type", errType),
					zap.Bool("retryable", isRetryable),
					zap.Error(err),
				)
				if !isRetryable {
					return nil
				}
				return err
			}
		}
		return nil
	}

	// 如果 metadata 已存在，只需更新状态
	if metadataExists {
		if email.Status != "classified" {
			if err := h.emailRepo.UpdateStatus(ctx, p.EmailID, "classified"); err != nil {
				isRetryable, errType := util.IsRetryableError(err)
				h.logger.Error("Failed to update email status",
					zap.Int("email_id", p.EmailID),
					zap.String("error_type", errType),
					zap.Bool("retryable", isRetryable),
					zap.Error(err),
				)
				if !isRetryable {
					return nil
				}
				return err
			}
		}
		h.logger.Debug("Metadata already exists, status updated",
			zap.Int("email_id", p.EmailID),
		)
		return nil
	}

	// 检查重试次数
	retryKey := util.FormatRetryKey("classify", p.EmailID)
	retryCount, err := h.retryCounter.IncrementAndGet(ctx, retryKey)
	if err != nil {
		// Redis 错误不影响处理，继续执行
		h.logger.Warn("Failed to get retry count, continuing anyway",
			zap.Int("email_id", p.EmailID),
			zap.Error(err),
		)
		retryCount = 1 // 假设是第一次
	}

	h.logger.Info("Retry count check",
		zap.Int("email_id", p.EmailID),
		zap.Int64("retry_count", retryCount),
		zap.Int64("max_retries", maxRetries),
	)

	// 调用 agent-service 进行分类（带超时控制）
	h.logger.Info("Calling agent-service for classification",
		zap.Int("email_id", p.EmailID),
		zap.Int64("retry_count", retryCount),
	)

	result, err := h.agentClient.ClassifyEmail(ctx, p.Subject, p.Body)
	if err != nil {
		// 检查错误类型
		isRetryable, errType := util.IsRetryableError(err)

		// 检查是否是超时或 500 错误
		isTimeout := contains(err.Error(), "timeout") || contains(err.Error(), "context deadline exceeded")
		is5xx := contains(err.Error(), "5xx")

		// 如果是超时或 500 错误，且是第一次尝试，可以重试一次
		if (isTimeout || is5xx) && retryCount == 1 {
			h.logger.Warn("Agent service timeout/5xx on first attempt, will retry",
				zap.Int("email_id", p.EmailID),
				zap.String("error_type", errType),
				zap.Bool("is_timeout", isTimeout),
				zap.Bool("is_5xx", is5xx),
			)
			// 返回错误，让 consumer 重试
			return err
		}

		// 如果超时或 500 错误且已重试过，写入 unknown 分类
		if (isTimeout || is5xx) && retryCount > 1 {
			h.logger.Warn("Agent service timeout/5xx after retry, writing unknown classification",
				zap.Int("email_id", p.EmailID),
				zap.String("error_type", errType),
				zap.Int64("retry_count", retryCount),
			)

			// 写入 unknown 分类
			if err := h.metadataRepo.Insert(ctx, p.EmailID, "unknown", 0.0); err != nil {
				h.logger.Error("Failed to insert unknown classification",
					zap.Int("email_id", p.EmailID),
					zap.Error(err),
				)
				// 如果插入失败，返回错误让 consumer 重试
				return err
			}

			// 记录错误到失败状态
			if err := h.metadataRepo.InsertFailed(ctx, p.EmailID, "ai_timeout"); err != nil {
				h.logger.Error("Failed to insert failed status",
					zap.Int("email_id", p.EmailID),
					zap.Error(err),
				)
			}

			// 重置重试计数
			h.retryCounter.Reset(ctx, retryKey)

			// 返回 nil，让 consumer ack
			return nil
		}

		h.logger.Error("Failed to classify email via agent-service",
			zap.Int("email_id", p.EmailID),
			zap.String("error_type", errType),
			zap.Bool("retryable", isRetryable),
			zap.Int64("retry_count", retryCount),
			zap.Error(err),
		)

		// 如果超过最大重试次数，写入失败状态并返回 nil（ack 掉）
		if retryCount > maxRetries {
			h.logger.Warn("Max retries exceeded, marking as failed",
				zap.Int("email_id", p.EmailID),
				zap.Int64("retry_count", retryCount),
			)

			// 写入失败状态
			if err := h.metadataRepo.InsertFailed(ctx, p.EmailID, "ai_failed"); err != nil {
				h.logger.Error("Failed to insert failed status",
					zap.Int("email_id", p.EmailID),
					zap.Error(err),
				)
			}

			// 重置重试计数
			h.retryCounter.Reset(ctx, retryKey)

			// 返回 nil，让 consumer ack 掉，不再重试
			return nil
		}

		// 如果不可重试，写入失败状态并返回 nil（ack 掉）
		if !isRetryable {
			h.logger.Warn("Non-retryable error, marking as failed",
				zap.Int("email_id", p.EmailID),
				zap.String("error_type", errType),
			)

			if err := h.metadataRepo.InsertFailed(ctx, p.EmailID, "ai_failed"); err != nil {
				h.logger.Error("Failed to insert failed status",
					zap.Int("email_id", p.EmailID),
					zap.Error(err),
				)
			}

			h.retryCounter.Reset(ctx, retryKey)
			return nil
		}

		// 可重试错误且未超过最大次数 - 返回 error，让 consumer nack 并重试
		return err
	}

	// 分类成功，重置重试计数
	h.retryCounter.Reset(ctx, retryKey)

	h.logger.Info("Classification result from agent-service",
		zap.Int("email_id", p.EmailID),
		zap.String("category", result.Category),
		zap.Float64("confidence", result.Confidence),
	)

	// 插入metadata（使用 ON CONFLICT 保证幂等性）
	if err := h.metadataRepo.Insert(ctx, p.EmailID, result.Category, result.Confidence); err != nil {
		isRetryable, errType := util.IsRetryableError(err)
		h.logger.Error("Failed to insert metadata",
			zap.Int("email_id", p.EmailID),
			zap.String("error_type", errType),
			zap.Bool("retryable", isRetryable),
			zap.Error(err),
		)
		if !isRetryable {
			return nil
		}
		return err
	}

	// 更新状态
	if err := h.emailRepo.UpdateStatus(ctx, p.EmailID, "classified"); err != nil {
		isRetryable, errType := util.IsRetryableError(err)
		h.logger.Error("Failed to update email status",
			zap.Int("email_id", p.EmailID),
			zap.String("error_type", errType),
			zap.Bool("retryable", isRetryable),
			zap.Error(err),
		)
		if !isRetryable {
			return nil
		}
		return err
	}

	h.logger.Info("Email classified successfully",
		zap.Int("email_id", p.EmailID),
		zap.String("category", result.Category),
		zap.Float64("confidence", result.Confidence),
	)

	return nil
}
