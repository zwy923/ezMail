package mqhandler

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    mqcontracts "mygoproject/contracts/mq"
    "task-service/internal/model"
    "task-service/internal/repository"

    "go.uber.org/zap"
)

type TaskCreatedHandler struct {
    taskRepo *repository.TaskRepository
    logger   *zap.Logger
}

func NewTaskCreatedHandler(taskRepo *repository.TaskRepository, logger *zap.Logger) *TaskCreatedHandler {
    return &TaskCreatedHandler{taskRepo: taskRepo, logger: logger}
}

func (h *TaskCreatedHandler) Handle(ctx context.Context, raw json.RawMessage) error {
    var p mqcontracts.TaskCreatedPayload
    if err := json.Unmarshal(raw, &p); err != nil {
        h.logger.Error("Failed to unmarshal TaskCreatedPayload", zap.Error(err))
        return err // 交给 Consumer 的重试/DLQ 机制处理
    }

    h.logger.Info("Handling task.created event",
        zap.Int("email_id", p.EmailID),
        zap.Int("user_id", p.UserID),
        zap.String("title", p.Title),
        zap.Int("due_in_days", p.DueInDays),
    )

    // 验证 email_id（task.created 事件必须来自邮件）
    if p.EmailID <= 0 {
        h.logger.Error("Invalid email_id in task.created event",
            zap.Int("email_id", p.EmailID),
            zap.Int("user_id", p.UserID),
        )
        return fmt.Errorf("invalid email_id: %d (must be > 0)", p.EmailID)
    }

    dueDate := time.Now().AddDate(0, 0, p.DueInDays)

    task := &model.Task{
        UserID:  p.UserID,
        EmailID: p.EmailID,
        Title:   p.Title,
        DueDate: dueDate,
        Status:  "pending",
        // CreatedAt 让 DB 默认填
    }

    _, err := h.taskRepo.Insert(ctx, task)
    if err != nil {
        h.logger.Error("Failed to insert task", zap.Error(err))
        return err
    }

    h.logger.Info("Task created successfully",
        zap.Int("user_id", p.UserID),
        zap.Int("email_id", p.EmailID),
    )
    return nil
}
