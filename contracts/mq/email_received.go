package mq

import "time"

// EmailReceivedPayload 邮件收到事件的 payload
type EmailReceivedPayload struct {
	EmailID    int       `json:"email_id"`
	UserID     int       `json:"user_id"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	ReceivedAt time.Time `json:"received_at"`
}
