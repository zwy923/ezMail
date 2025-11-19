package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type TaskRepository struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewTaskRepository(db *pgxpool.Pool, logger *zap.Logger) *TaskRepository {
	return &TaskRepository{
		db:     db,
		logger: logger,
	}
}

// MarkExpired marks tasks as overdue if due_date < today and status = 'pending'
func (r *TaskRepository) MarkExpired(ctx context.Context) error {
	query := `
        UPDATE tasks
        SET status = 'overdue'
        WHERE status = 'pending'
          AND due_date < CURRENT_DATE
          AND due_date IS NOT NULL
    `
	result, err := r.db.Exec(ctx, query)
	if err != nil {
		r.logger.Error("Failed to mark expired tasks", zap.Error(err))
		return err
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected > 0 {
		r.logger.Info("Marked tasks as overdue",
			zap.Int64("count", rowsAffected),
		)
	}
	return nil
}

// ListExpiredPendingTasks returns tasks that should be marked as overdue
func (r *TaskRepository) ListExpiredPendingTasks(ctx context.Context) ([]int, error) {
	query := `
        SELECT id FROM tasks
        WHERE status = 'pending'
          AND due_date < CURRENT_DATE
          AND due_date IS NOT NULL
    `
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var taskIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		taskIDs = append(taskIDs, id)
	}
	return taskIDs, nil
}

// ListTasksWithDependencies returns tasks that have dependencies and are still pending
func (r *TaskRepository) ListTasksWithDependencies(ctx context.Context) ([]TaskWithDeps, error) {
	query := `
        SELECT t.id, t.user_id, t.title, t.status,
               COALESCE(COUNT(td.depends_on_task_id), 0) as dep_count,
               COALESCE(COUNT(CASE WHEN td_status.status = 'done' THEN 1 END), 0) as completed_dep_count
        FROM tasks t
        LEFT JOIN task_dependencies td ON t.id = td.task_id
        LEFT JOIN tasks td_status ON td.depends_on_task_id = td_status.id
        WHERE t.status = 'pending'
          AND t.project_id IS NOT NULL
        GROUP BY t.id, t.user_id, t.title, t.status
        HAVING COUNT(td.depends_on_task_id) > 0
    `
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []TaskWithDeps
	for rows.Next() {
		var t TaskWithDeps
		if err := rows.Scan(
			&t.ID,
			&t.UserID,
			&t.Title,
			&t.Status,
			&t.DepCount,
			&t.CompletedDepCount,
		); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, nil
}

type TaskWithDeps struct {
	ID               int
	UserID           int
	Title            string
	Status           string
	DepCount         int64
	CompletedDepCount int64
}


