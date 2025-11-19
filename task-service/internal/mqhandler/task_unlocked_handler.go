package mqhandler

import (
	"context"
	"encoding/json"

	"task-service/internal/repository"

	"go.uber.org/zap"
)

type TaskUnlockedHandler struct {
	taskRepo *repository.TaskRepository
	logger   *zap.Logger
}

func NewTaskUnlockedHandler(taskRepo *repository.TaskRepository, logger *zap.Logger) *TaskUnlockedHandler {
	return &TaskUnlockedHandler{
		taskRepo: taskRepo,
		logger:   logger,
	}
}

func (h *TaskUnlockedHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var p struct {
		TaskID int    `json:"task_id"`
		UserID int    `json:"user_id"`
		Title  string `json:"title"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal TaskUnlockedPayload", zap.Error(err))
		return err
	}

	h.logger.Info("Handling task.unlocked event",
		zap.Int("task_id", p.TaskID),
		zap.Int("user_id", p.UserID),
		zap.String("title", p.Title),
	)

	// Task dependencies are completed, task is now unlockable
	// This handler can be used for additional processing (e.g., notifications, analytics)
	
	return nil
}

