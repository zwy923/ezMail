package model

import "time"

type NotificationLog struct {
    ID        int
    UserID    int
    EmailID   int
    Message   string
    CreatedAt time.Time
}
