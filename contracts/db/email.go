package db

import "time"

// Email 表示 emails_raw 表的完整结构
type Email struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	RawJSON   string    `json:"raw_json"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

// EmailWithMetadata 表示带元数据的邮件（用于查询结果）
type EmailWithMetadata struct {
	ID         int       `json:"id"`
	UserID     int       `json:"user_id,omitempty"`
	Subject    string    `json:"subject"`
	Body       string    `json:"body"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	Categories []string  `json:"categories,omitempty"`
	Priority   string    `json:"priority,omitempty"`
	Summary    string    `json:"summary,omitempty"`
}

