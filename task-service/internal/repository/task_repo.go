package repository

import (
	"context"
	"task-service/internal/model"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type TaskRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewTaskRepository(db *pgxpool.Pool, logger *zap.Logger) *TaskRepository {
	return &TaskRepository{db: db, logger: logger}
}

func (r *TaskRepository) Insert(ctx context.Context, t *model.Task) (int, error) {
	r.logger.Debug("Inserting task",
		zap.Int("user_id", t.UserID),
		zap.Int("email_id", t.EmailID),
		zap.String("title", t.Title),
		zap.String("status", t.Status),
	)
	query := `
        INSERT INTO tasks (user_id, email_id, title, due_date, status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query,
		t.UserID,
		t.EmailID,
		t.Title,
		t.DueDate,
		t.Status,
	).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to insert task",
			zap.Error(err),
			zap.Int("user_id", t.UserID),
			zap.Int("email_id", t.EmailID),
		)
		return 0, err
	}
	r.logger.Info("Task inserted successfully",
		zap.Int("task_id", id),
		zap.Int("user_id", t.UserID),
	)
	return id, nil
}

func (r *TaskRepository) ListByUser(ctx context.Context, userID int) ([]model.Task, error) {
	r.logger.Debug("Listing tasks for user", zap.Int("user_id", userID))
	query := `
        SELECT id, user_id, email_id, title, due_date, status, created_at
        FROM tasks
        WHERE user_id = $1
        ORDER BY created_at DESC
    `
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to query tasks",
			zap.Error(err),
			zap.Int("user_id", userID),
		)
		return nil, err
	}
	defer rows.Close()

	tasks := []model.Task{}
	for rows.Next() {
		var t model.Task
		if err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.EmailID,
			&t.Title,
			&t.DueDate,
			&t.Status,
			&t.CreatedAt,
		); err != nil {
			r.logger.Error("Failed to scan task row",
				zap.Error(err),
				zap.Int("user_id", userID),
			)
			return nil, err
		}
		tasks = append(tasks, t)
	}
	r.logger.Info("Tasks listed successfully",
		zap.Int("user_id", userID),
		zap.Int("count", len(tasks)),
	)
	return tasks, nil
}

func (r *TaskRepository) MarkCompleted(ctx context.Context, taskID int) error {
	r.logger.Debug("Marking task as completed", zap.Int("task_id", taskID))
	query := `
        UPDATE tasks
        SET status = 'done', completed_at = NOW()
        WHERE id = $1
    `
	result, err := r.db.Exec(ctx, query, taskID)
	if err != nil {
		r.logger.Error("Failed to mark task as completed",
			zap.Error(err),
			zap.Int("task_id", taskID),
		)
		return err
	}
	rowsAffected := result.RowsAffected()
	r.logger.Info("Task marked as completed",
		zap.Int("task_id", taskID),
		zap.Int64("rows_affected", rowsAffected),
	)
	return nil
}

func (r *TaskRepository) MarkExpired(ctx context.Context) error {
	r.logger.Debug("Marking expired tasks as overdue")
	query := `
        UPDATE tasks
        SET status = 'overdue'
        WHERE status = 'pending'
        AND due_date < NOW()
    `
	result, err := r.db.Exec(ctx, query)
	if err != nil {
		r.logger.Error("Failed to mark expired tasks", zap.Error(err))
		return err
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		r.logger.Info("Expired tasks marked as overdue",
			zap.Int64("tasks_updated", rowsAffected),
		)
	} else {
		r.logger.Debug("No expired tasks found")
	}
	return nil
}
