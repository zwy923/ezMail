package mqhandler

import (
	"context"
	"encoding/json"
	"time"

	mqcontracts "mygoproject/contracts/mq"
	"task-service/internal/model"
	"task-service/internal/repository"

	"go.uber.org/zap"
)

type TaskBulkCreatedHandler struct {
	taskRepo *repository.TaskRepository
	logger   *zap.Logger
}

func NewTaskBulkCreatedHandler(taskRepo *repository.TaskRepository, logger *zap.Logger) *TaskBulkCreatedHandler {
	return &TaskBulkCreatedHandler{
		taskRepo: taskRepo,
		logger:   logger,
	}
}

func (h *TaskBulkCreatedHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var p mqcontracts.TaskBulkCreatedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal TaskBulkCreatedPayload", zap.Error(err))
		return err
	}

	h.logger.Info("Handling task.bulk_created event",
		zap.Int("user_id", p.UserID),
		zap.Int("task_count", len(p.Tasks)),
	)

	if len(p.Tasks) == 0 {
		h.logger.Warn("Empty task list in bulk_created event",
			zap.Int("user_id", p.UserID),
		)
		return nil
	}

	// 转换为 model.Task 列表
	tasks := make([]model.Task, len(p.Tasks))
	now := time.Now()
	for i, taskItem := range p.Tasks {
		dueDate := now.AddDate(0, 0, taskItem.DueInDays)
		tasks[i] = model.Task{
			UserID:  p.UserID,
			EmailID: 0, // 文本转任务没有关联的 email
			Title:   taskItem.Title,
			DueDate: dueDate,
			Status:  "pending",
		}
	}

	// 批量插入任务
	ids, err := h.taskRepo.BulkInsert(ctx, p.UserID, tasks)
	if err != nil {
		h.logger.Error("Failed to bulk insert tasks",
			zap.Error(err),
			zap.Int("user_id", p.UserID),
			zap.Int("task_count", len(tasks)),
		)
		return err
	}

	h.logger.Info("Tasks bulk created successfully",
		zap.Int("user_id", p.UserID),
		zap.Int("created_count", len(ids)),
	)

	return nil
}

