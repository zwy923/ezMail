package mqhandler

import (
	"context"
	"encoding/json"
	"fmt"

	mqcontracts "mygoproject/contracts/mq"
	"mygoproject/pkg/util"
	"worker-service/internal/repository"
	"worker-service/internal/service"

	"mygoproject/pkg/mq"

	"go.uber.org/zap"
)

const (
	maxRetries = 5
)

type AgentDecisionHandler struct {
	emailRepo        *repository.EmailRepository
	metadataRepo     *repository.MetadataRepository
	taskRepo         *repository.TaskRepository
	notificationRepo *repository.NotificationRepository

	agentClient  *service.AgentClient
	retryCounter *util.RetryCounter
	deduper      *util.Deduper
	logger       *zap.Logger

	taskPublisher *mq.Publisher  // 新增：负责任务事件的发布
}

func NewAgentDecisionHandler(
    emailRepo *repository.EmailRepository,
    metadataRepo *repository.MetadataRepository,
    notificationRepo *repository.NotificationRepository,
    agentClient *service.AgentClient,
    retryCounter *util.RetryCounter,
    deduper *util.Deduper,
    taskPublisher *mq.Publisher,   // 新增
    logger *zap.Logger,
) *AgentDecisionHandler {
    return &AgentDecisionHandler{
        emailRepo:        emailRepo,
        metadataRepo:     metadataRepo,
        notificationRepo: notificationRepo,
        agentClient:      agentClient,
        retryCounter:     retryCounter,
        deduper:          deduper,
        taskPublisher:    taskPublisher,
        logger:           logger,
    }
}

func (h *AgentDecisionHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	defer h.recoverPanic()

	// --------------------------
	// Step 1: decode payload
	// --------------------------
	var payload mqcontracts.EmailReceivedPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		h.logger.Error("Invalid EmailReceivedPayload, sending to DLQ",
			zap.String("raw", string(raw)),
			zap.Error(err),
		)
		return fmt.Errorf("bad_payload: %w", err)
	}

	h.logger.Info("AgentDecisionHandler: received email",
		zap.Int("email_id", payload.EmailID),
		zap.Int("user_id", payload.UserID),
	)

	// --------------------------
	// Step 2: load email
	// --------------------------
	email, _, err := h.emailRepo.FindRawWithMetadataByID(ctx, payload.EmailID)
	if err != nil {
		return h.handleRepoError("FindRawWithMetadataByID", err)
	}

	// 幂等：已经标记 classified → 跳过
	if email.Status == "classified" {
		h.logger.Info("Email already classified, skip",
			zap.Int("email_id", payload.EmailID),
		)
		return nil
	}

	// Redis 去重（避免并发重复消费）
	if !h.deduper.AcquireOnce(ctx, "agent", payload.EmailID) {
		h.logger.Info("Duplicated event, skip",
			zap.Int("email_id", payload.EmailID),
		)
		return nil
	}

	// --------------------------
	// Step 3: retry count
	// --------------------------
	retryKey := util.FormatRetryKey("agent", payload.EmailID)
	retryCount, _ := h.retryCounter.IncrementAndGet(ctx, retryKey)
	h.logger.Info("Retry count", zap.Int64("retry", retryCount))

	// --------------------------
	// Step 4: call agent-service
	// --------------------------
	decision, err := h.agentClient.Decide(ctx, service.EmailInput{
		EmailID: payload.EmailID,
		UserID:  payload.UserID,
		Subject: payload.Subject,
		Body:    payload.Body,
	})

	if err != nil {
		return h.handleAgentError(ctx, err, retryKey, retryCount, payload.EmailID)
	}

	// --------------------------
	// Step 5: write metadata
	// --------------------------
	if err := h.metadataRepo.InsertDecision(ctx, payload.EmailID, decision); err != nil {
		return h.handleRepoError("InsertDecision", err)
	}

	// --------------------------
	// Step 6: create task (optional)
	// --------------------------
	if decision.ShouldCreateTask && decision.Task != nil {
		taskPayload := mqcontracts.TaskCreatedPayload{
			EmailID:   payload.EmailID,
			UserID:    payload.UserID,
			Title:     decision.Task.Title,
			DueInDays: decision.Task.DueInDays,
		}
	
		h.logger.Info("Publishing task.created event",
			zap.Int("email_id", taskPayload.EmailID),
			zap.Int("user_id", taskPayload.UserID),
			zap.String("title", taskPayload.Title),
			zap.Int("due_in_days", taskPayload.DueInDays),
		)
	
		if err := h.taskPublisher.Publish("task.created", taskPayload); err != nil {
			// 这里按你的风格走统一错误处理
			h.logger.Error("Failed to publish task.created event", zap.Error(err))
			// 一般来说：MQ 发布失败是可重试错误
			return err
		}
	}

	// --------------------------
	// Step 7: create notification (optional)
	// --------------------------
	if decision.ShouldNotify {
		if err := h.notificationRepo.Insert(ctx, payload.EmailID, payload.UserID, decision); err != nil {
			return h.handleRepoError("InsertNotification", err)
		}
	}

	// --------------------------
	// Step 8: update email status
	// --------------------------
	if err := h.emailRepo.UpdateStatus(ctx, payload.EmailID, "classified"); err != nil {
		return h.handleRepoError("UpdateStatus", err)
	}

	// --------------------------
	// Step 9: cleanup & finish
	// --------------------------
	h.retryCounter.Reset(ctx, retryKey)

	h.logger.Info("Email processed successfully",
		zap.Int("email_id", payload.EmailID),
		zap.Strings("categories", decision.Categories),
		zap.String("priority", decision.Priority),
	)

	return nil
}

func (h *AgentDecisionHandler) handleRepoError(op string, err error) error {
	isRetryable, errType := util.IsRetryableError(err)
	h.logger.Error("Repo error",
		zap.String("op", op),
		zap.String("error_type", errType),
		zap.Bool("retryable", isRetryable),
		zap.Error(err),
	)

	if isRetryable {
		return err // nack → 重试
	}
	return nil // ack → 吃掉
}

func (h *AgentDecisionHandler) handleAgentError(ctx context.Context, err error, retryKey string, retryCount int64, emailID int) error {
	isRetryable, errType := util.IsRetryableError(err)

	h.logger.Warn("Agent service error",
		zap.String("error", err.Error()),
		zap.String("type", errType),
		zap.Bool("retryable", isRetryable),
		zap.Int("email_id", emailID),
		zap.Int64("retry", retryCount),
	)

	// 多次失败 → 写 unknown + classified
	if retryCount > maxRetries {
		h.logger.Warn("Max retries exceeded → write unknown", zap.Int("email_id", emailID))

		_ = h.metadataRepo.InsertUnknown(ctx, emailID)
		_ = h.emailRepo.UpdateStatus(ctx, emailID, "classified")
		h.retryCounter.Reset(ctx, retryKey)

		return nil // ack
	}

	if !isRetryable {
		h.logger.Warn("Non-retryable agent error → write unknown")

		_ = h.metadataRepo.InsertUnknown(ctx, emailID)
		_ = h.emailRepo.UpdateStatus(ctx, emailID, "classified")
		h.retryCounter.Reset(ctx, retryKey)

		return nil // ack
	}

	return err // nack → 重试
}

func (h *AgentDecisionHandler) recoverPanic() {
	if r := recover(); r != nil {
		h.logger.Error("panic recovered in handler", zap.Any("panic", r))
	}
}
