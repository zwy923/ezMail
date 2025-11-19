package model

import "time"

type Task struct {
    ID        int       `json:"id"`
    UserID    int       `json:"user_id"`
    EmailID   int       `json:"email_id"`
    Title     string    `json:"title"`
    DueDate   time.Time `json:"due_date"`
    Status    string    `json:"status"`
    CreatedAt time.Time `json:"created_at"`
}
