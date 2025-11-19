package model

import "time"

type Project struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TargetDate  time.Time `json:"target_date"`
	Status      string    `json:"status"` // active / completed / cancelled
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Milestone struct {
	ID          int       `json:"id"`
	ProjectID   int       `json:"project_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	PhaseOrder  int       `json:"phase_order"`
	TargetDate  time.Time `json:"target_date"`
	Status      string    `json:"status"` // pending / in_progress / completed
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

