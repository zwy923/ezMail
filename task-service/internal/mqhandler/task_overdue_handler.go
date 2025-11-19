package mqhandler

import (
	"context"
	"encoding/json"

	"task-service/internal/repository"

	"go.uber.org/zap"
)

type TaskOverdueHandler struct {
	taskRepo *repository.TaskRepository
	logger   *zap.Logger
}

func NewTaskOverdueHandler(taskRepo *repository.TaskRepository, logger *zap.Logger) *TaskOverdueHandler {
	return &TaskOverdueHandler{
		taskRepo: taskRepo,
		logger:   logger,
	}
}

func (h *TaskOverdueHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var p struct {
		TaskID int `json:"task_id"`
	}
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal TaskOverduePayload", zap.Error(err))
		return err
	}

	h.logger.Info("Handling task.overdue event",
		zap.Int("task_id", p.TaskID),
	)

	// Task is already marked as overdue by task-runner-service
	// This handler can be used for additional processing (e.g., notifications, analytics)
	
	return nil
}

