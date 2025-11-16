package service

import (
    "context"
    "encoding/json"
    "mygoproject/internal/mq"
    "mygoproject/internal/repository"
    "mygoproject/internal/model"
    "fmt"
)

type NotificationLogService struct {
    repo *repository.NotificationLogRepository
}

func NewNotificationLogService(repo *repository.NotificationLogRepository) *NotificationLogService {
    return &NotificationLogService{repo: repo}
}

// HandleEmailReceived -- 写入 notifications_log
func (s *NotificationLogService) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
    var p mq.EmailReceivedPayload
    if err := json.Unmarshal(raw, &p); err != nil {
        return err
    }

    log := &model.NotificationLog{
        UserID:  p.UserID,
        EmailID: p.EmailID,
        Message: fmt.Sprintf("User %d received a new email %d", p.UserID, p.EmailID),
    }

    return s.repo.Insert(ctx, log)
}
