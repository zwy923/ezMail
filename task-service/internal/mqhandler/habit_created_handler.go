package mqhandler

import (
	"context"
	"encoding/json"

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
	)

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

