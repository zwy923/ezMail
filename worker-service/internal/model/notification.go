package model

import "time"

type Notification struct {
	ID        int
	UserID    int
	Type      string
	Content   string
	IsRead    bool
	CreatedAt time.Time
}

type NotificationLog struct {
	ID        int
	UserID    int
	EmailID   int
	Message   string
	CreatedAt time.Time
}

