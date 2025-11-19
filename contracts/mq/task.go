package mq

type TaskCreatedPayload struct {
	EmailID   int    `json:"email_id"`
	UserID    int    `json:"user_id"`
	Title     string `json:"title"`
	DueInDays int    `json:"due_in_days"`
}

type TaskItem struct {
	Title     string `json:"title"`
	DueInDays int    `json:"due_in_days"`
}

type TaskBulkCreatedPayload struct {
	UserID int        `json:"user_id"`
	Tasks  []TaskItem `json:"tasks"`
}

type HabitCreatedPayload struct {
	UserID            int    `json:"user_id"`
	Title             string `json:"title"`
	RecurrencePattern string `json:"recurrence_pattern"` // "weekly Wednesday", "daily", "monthly 1"
}

type ProjectTask struct {
	Title     string   `json:"title"`
	DueInDays int      `json:"due_in_days"`
	Priority  string   `json:"priority"`   // LOW / MEDIUM / HIGH
	DependsOn []string `json:"depends_on"` // List of task titles this task depends on
}

type Milestone struct {
	Title     string        `json:"title"`
	Order     int           `json:"order"`
	DueInDays int           `json:"due_in_days"`
	Tasks     []ProjectTask `json:"tasks"`
}

type ProjectCreatedPayload struct {
	UserID      int         `json:"user_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	TargetDays  int         `json:"target_days"` // Days until project completion
	Milestones  []Milestone `json:"milestones"`
}

// Task Orchestrator Events
type TaskOverduePayload struct {
	TaskID int `json:"task_id"`
}

type TaskUnlockedPayload struct {
	TaskID int    `json:"task_id"`
	UserID int    `json:"user_id"`
	Title  string `json:"title"`
}

type HabitTaskGeneratedPayload struct {
	HabitID int    `json:"habit_id"`
	UserID  int    `json:"user_id"`
	Title   string `json:"title"`
	DueDate string `json:"due_date"` // YYYY-MM-DD format
}
