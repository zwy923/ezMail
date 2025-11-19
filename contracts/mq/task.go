package mq

type TaskCreatedPayload struct {
    EmailID   int    `json:"email_id"`
    UserID    int    `json:"user_id"`
    Title     string `json:"title"`
    DueInDays int    `json:"due_in_days"`
}
