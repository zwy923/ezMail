package repository

import (
    "context"
    "time"
    "worker-service/internal/model"

    "github.com/jackc/pgx/v5/pgxpool"
)

type TaskRepository struct {
    db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
    return &TaskRepository{db: db}
}

func (r *TaskRepository) Insert(ctx context.Context,
    emailID int,
    userID int,
    task *model.TaskDecision,
) error {

    dueDate := time.Now().AddDate(0, 0, task.DueInDays)

    sql := `
        INSERT INTO tasks (email_id, user_id, title, due_date, status)
        VALUES ($1, $2, $3, $4, 'pending')
        ON CONFLICT (email_id) DO NOTHING;
    `

    _, err := r.db.Exec(ctx, sql, emailID, userID, task.Title, dueDate)
    return err
}
