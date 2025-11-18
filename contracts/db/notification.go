package db

import "time"

// Notification 表示 notifications 表的完整结构（Phase 2）
type Notification struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	EmailID   int       `json:"email_id"`
	Channel   string    `json:"channel"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}

// NotificationLog 表示 notifications_log 表的结构
type NotificationLog struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	EmailID   int       `json:"email_id"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}
