package model


type TaskDecision struct {
    Title     string `json:"title"`
    DueInDays int    `json:"due_in_days"`
}



type AgentDecision struct {
    Categories []string `json:"categories"`
    Priority   string   `json:"priority"`
    Summary    string   `json:"summary"`

    ShouldCreateTask bool           `json:"should_create_task"`
    Task             *TaskDecision  `json:"task"`

    ShouldNotify        bool   `json:"should_notify"`
    NotificationChannel string `json:"notification_channel"`
    NotificationMessage string `json:"notification_message"`
}