package db

import "time"

// EmailMetadata 表示 emails_metadata 表的完整结构（Phase 2）
type EmailMetadata struct {
	EmailID   int       `json:"email_id"`
	Categories []string `json:"categories"`
	Priority   string   `json:"priority"`
	Summary    string   `json:"summary"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

