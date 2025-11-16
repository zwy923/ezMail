package service

import (
    "context"
    "encoding/json"
    "mygoproject/internal/mq"
    "mygoproject/internal/repository"
    "mygoproject/internal/model"
    "fmt"
)

type NotificationService struct {
    repo *repository.NotificationRepository
}

func NewNotificationService(repo *repository.NotificationRepository) *NotificationService {
    return &NotificationService{repo: repo}
}

// HandleEmailReceived -- 写入 notifications 站内通知
func (s *NotificationService) HandleEmailReceived(ctx context.Context, raw json.RawMessage) error {
    var p mq.EmailReceivedPayload
    if err := json.Unmarshal(raw, &p); err != nil {
        return err
    }

    notif := &model.Notification{
        UserID:  p.UserID,
        Type:    "new_email",
        Content: fmt.Sprintf("你收到了新邮件：%s", p.Subject),
    }

    return s.repo.Insert(ctx, notif)
}
