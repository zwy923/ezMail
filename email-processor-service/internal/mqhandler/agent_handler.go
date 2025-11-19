package mqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"email-processor-service/internal/repository"
	"email-processor-service/internal/service"
	mqcontracts "mygoproject/contracts/mq"
	"mygoproject/pkg/util"

	"mygoproject/pkg/logger"
	"mygoproject/pkg/metrics"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/outbox"
	"mygoproject/pkg/trace"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

const (
	maxRetries = 5
)

type AgentDecisionHandler struct {
	db           *pgxpool.Pool
	emailRepo    *repository.EmailRepository
	metadataRepo *repository.MetadataRepository
	outboxRepo   *outbox.Repository

	agentClient  *service.AgentClient
	retryCounter *util.RetryCounter
	deduper      *util.Deduper
	logger       *zap.Logger

	taskPublisher *mq.Publisher // 负责任务和通知事件的发布
}

func NewAgentDecisionHandler(
	db *pgxpool.Pool,
	emailRepo *repository.EmailRepository,
	metadataRepo *repository.MetadataRepository,
	agentClient *service.AgentClient,
	retryCounter *util.RetryCounter,
	deduper *util.Deduper,
	taskPublisher *mq.Publisher,
	logger *zap.Logger,
) *AgentDecisionHandler {
	return &AgentDecisionHandler{
		db:            db,
		emailRepo:     emailRepo,
		metadataRepo:  metadataRepo,
		outboxRepo:    outbox.NewRepository(db),
		agentClient:   agentClient,
		retryCounter:  retryCounter,
		deduper:       deduper,
		taskPublisher: taskPublisher,
		logger:        logger,
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

	// 从 payload 中提取 trace_id 并添加到 context（如果存在）
	if payload.TraceID != "" {
		ctx = trace.WithContext(ctx, payload.TraceID)
	}

	// 使用带 trace_id 的 logger
	traceLogger := logger.WithTrace(ctx, h.logger)
	traceLogger.Info("AgentDecisionHandler: received email",
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
	// Step 5-8: 使用事务写入 metadata、outbox 事件和更新状态
	// --------------------------
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Step 5: write metadata (in transaction)
	if err := h.metadataRepo.InsertDecisionTx(ctx, tx, payload.EmailID, decision); err != nil {
		return h.handleRepoError("InsertDecision", err)
	}

	// Step 6: insert task.created event to outbox (if needed)
	traceID := trace.FromContext(ctx)
	emailID64 := int64(payload.EmailID)
	if decision.ShouldCreateTask && decision.Task != nil {
		taskPayload := mqcontracts.TaskCreatedPayload{
			EmailID:   payload.EmailID,
			UserID:    payload.UserID,
			Title:     decision.Task.Title,
			DueInDays: decision.Task.DueInDays,
			TraceID:   traceID,
		}

		if err := outbox.InsertEventInTx(ctx, tx, h.outboxRepo, "task", &emailID64, "task.created", taskPayload); err != nil {
			h.logger.Error("Failed to insert task.created to outbox", zap.Error(err))
			return err
		}

		traceLogger.Info("Inserted task.created event to outbox",
			zap.Int("email_id", taskPayload.EmailID),
			zap.Int("user_id", taskPayload.UserID),
			zap.String("title", taskPayload.Title),
		)
	}

	// Step 7: insert notification.created event to outbox (if needed)
	if decision.ShouldNotify {
		notiPayload := mqcontracts.NotificationCreatedPayload{
			UserID:    payload.UserID,
			EmailID:   payload.EmailID,
			Channel:   decision.NotificationChannel,
			Message:   decision.NotificationMessage,
			CreatedAt: time.Now(),
			TraceID:   traceID,
		}

		if err := outbox.InsertEventInTx(ctx, tx, h.outboxRepo, "email", &emailID64, "notification.created", notiPayload); err != nil {
			h.logger.Error("Failed to insert notification.created to outbox", zap.Error(err))
			// 通知失败不影响主流程，继续
		} else {
			traceLogger.Info("Inserted notification.created event to outbox",
				zap.Int("user_id", payload.UserID),
				zap.Int("email_id", payload.EmailID),
			)
		}
	}

	// Step 8: update email status (in transaction)
	if err := h.emailRepo.UpdateStatusTx(ctx, tx, payload.EmailID, "classified"); err != nil {
		return h.handleRepoError("UpdateStatus", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// 记录邮件处理成功
	metrics.IncrementEmailProcessed("success")
	// 记录任务生成（如果创建了任务）
	if decision.ShouldCreateTask && decision.Task != nil {
		metrics.IncrementTaskGeneration("email")
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
