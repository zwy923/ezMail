package model

import "time"

type Habit struct {
	ID               int       `json:"id"`
	UserID           int       `json:"user_id"`
	Title            string    `json:"title"`
	RecurrencePattern string    `json:"recurrence_pattern"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

