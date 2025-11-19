package mq

import "time"

type NotificationCreatedPayload struct {
	UserID    int       `json:"user_id"`
	EmailID   int       `json:"email_id,omitempty"`
	TaskID    int       `json:"task_id,omitempty"`
	Channel   string    `json:"channel"` // EMAIL / PUSH / SMS / WEBHOOK
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
	TraceID   string    `json:"trace_id,omitempty"`
}

type NotificationSentPayload struct {
	NotificationID int       `json:"notification_id"`
	UserID         int       `json:"user_id"`
	Channel        string    `json:"channel"`
	SentAt         time.Time `json:"sent_at"`
	TraceID        string    `json:"trace_id,omitempty"`
}

type NotificationFailedPayload struct {
	NotificationID int    `json:"notification_id"`
	UserID         int    `json:"user_id"`
	Channel        string `json:"channel"`
	Error          string `json:"error"`
	RetryCount     int    `json:"retry_count"`
	TraceID        string `json:"trace_id,omitempty"`
}
