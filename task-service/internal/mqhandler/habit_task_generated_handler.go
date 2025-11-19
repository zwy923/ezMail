package mqhandler

import (
	"context"
	"encoding/json"
	"time"

	"task-service/internal/repository"

	"go.uber.org/zap"
)

type HabitTaskGeneratedHandler struct {
	taskRepo *repository.TaskRepository
	logger   *zap.Logger
}

func NewHabitTaskGeneratedHandler(taskRepo *repository.TaskRepository, logger *zap.Logger) *HabitTaskGeneratedHandler {
	return &HabitTaskGeneratedHandler{
		taskRepo: taskRepo,
		logger:   logger,
	}
}

func (h *HabitTaskGeneratedHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var p struct {
		HabitID int    `json:"habit_id"`
		UserID  int    `json:"user_id"`
		Title   string `json:"title"`
		DueDate string `json:"due_date"` // YYYY-MM-DD format
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal HabitTaskGeneratedPayload", zap.Error(err))
		return err
	}

	h.logger.Info("Handling habit.task.generated event",
		zap.Int("habit_id", p.HabitID),
		zap.Int("user_id", p.UserID),
		zap.String("title", p.Title),
		zap.String("due_date", p.DueDate),
	)

	// Parse due date
	dueDate, err := time.Parse("2006-01-02", p.DueDate)
	if err != nil {
		h.logger.Error("Failed to parse due_date", zap.Error(err))
		return err
	}

	// Insert task from habit (幂等性由数据库唯一索引保证)
	_, err = h.taskRepo.InsertFromHabit(ctx, p.HabitID, p.UserID, p.Title, dueDate)
	if err != nil {
		h.logger.Error("Failed to insert task from habit", zap.Error(err))
		return err
	}

	h.logger.Info("Task from habit created successfully",
		zap.Int("habit_id", p.HabitID),
		zap.Int("user_id", p.UserID),
		zap.String("title", p.Title),
	)

	return nil
}

