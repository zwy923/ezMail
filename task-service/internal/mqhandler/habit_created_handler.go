package mqhandler

import (
	"context"
	"encoding/json"
	"fmt"

	mqcontracts "mygoproject/contracts/mq"
	"task-service/internal/model"
	"task-service/internal/repository"

	"go.uber.org/zap"
)

type HabitCreatedHandler struct {
	habitRepo *repository.HabitRepository
	logger    *zap.Logger
}

func NewHabitCreatedHandler(habitRepo *repository.HabitRepository, logger *zap.Logger) *HabitCreatedHandler {
	return &HabitCreatedHandler{
		habitRepo: habitRepo,
		logger:    logger,
	}
}

func (h *HabitCreatedHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var p mqcontracts.HabitCreatedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal HabitCreatedPayload", zap.Error(err))
		return err
	}

	h.logger.Info("Handling habit.created event",
		zap.Int("user_id", p.UserID),
		zap.String("title", p.Title),
		zap.String("recurrence_pattern", p.RecurrencePattern),
		zap.String("trace_id", p.TraceID),
	)

	// RBAC 验证：记录 user_id（MQ 事件来自内部服务，但记录用于审计）
	// 注意：MQ 事件中的 user_id 应该已经在 api-gateway 中验证过
	// 这里主要是记录和审计，防止潜在的伪造事件
	if p.UserID <= 0 {
		h.logger.Error("Invalid user_id in habit.created event",
			zap.Int("user_id", p.UserID),
		)
		return fmt.Errorf("invalid user_id: %d", p.UserID)
	}

	habit := &model.Habit{
		UserID:            p.UserID,
		Title:             p.Title,
		RecurrencePattern: p.RecurrencePattern,
		IsActive:          true,
	}

	_, err := h.habitRepo.Insert(ctx, habit)
	if err != nil {
		h.logger.Error("Failed to insert habit",
			zap.Error(err),
			zap.Int("user_id", p.UserID),
			zap.String("title", p.Title),
		)
		return err
	}

	h.logger.Info("Habit created successfully",
		zap.Int("user_id", p.UserID),
		zap.String("title", p.Title),
		zap.String("recurrence_pattern", p.RecurrencePattern),
	)

	return nil
}

