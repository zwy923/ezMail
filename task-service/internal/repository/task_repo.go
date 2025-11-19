package repository

import (
	"context"
	"database/sql"
	"time"

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

	// 如果 email_id 为 0，使用 NULL（避免外键冲突）
	var emailID interface{}
	if t.EmailID > 0 {
		emailID = t.EmailID
	} else {
		emailID = nil
	}

	query := `
        INSERT INTO tasks (user_id, email_id, title, due_date, status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query,
		t.UserID,
		emailID,
		t.Title,
		t.DueDate,
		t.Status,
	).Scan(&id)
	if err != nil {
		r.logger.Error("Failed to insert task",
			zap.Error(err),
			zap.Int("user_id", t.UserID),
			zap.Any("email_id", emailID),
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
		var emailID sql.NullInt32 // 使用 NullInt32 处理可能为 NULL 的 email_id
		if err := rows.Scan(
			&t.ID,
			&t.UserID,
			&emailID,
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
		// 如果 email_id 为 NULL，设置为 0
		if emailID.Valid {
			t.EmailID = int(emailID.Int32)
		} else {
			t.EmailID = 0
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

// BulkInsert inserts multiple tasks in a single transaction
func (r *TaskRepository) BulkInsert(ctx context.Context, userID int, tasks []model.Task) ([]int, error) {
	if len(tasks) == 0 {
		return []int{}, nil
	}

	r.logger.Debug("Bulk inserting tasks",
		zap.Int("user_id", userID),
		zap.Int("count", len(tasks)),
	)

	// 使用事务确保原子性
	tx, err := r.db.Begin(ctx)
	if err != nil {
		r.logger.Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
        INSERT INTO tasks (user_id, email_id, title, due_date, status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id
    `

	ids := make([]int, 0, len(tasks))
	for _, t := range tasks {
		// 如果 email_id 为 0，使用 NULL（避免外键冲突）
		var emailID interface{}
		if t.EmailID > 0 {
			emailID = t.EmailID
		} else {
			emailID = nil
		}

		var id int
		err := tx.QueryRow(ctx, query,
			userID,
			emailID,
			t.Title,
			t.DueDate,
			t.Status,
		).Scan(&id)
		if err != nil {
			r.logger.Error("Failed to insert task in bulk",
				zap.Error(err),
				zap.String("title", t.Title),
			)
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("Failed to commit bulk insert transaction", zap.Error(err))
		return nil, err
	}

	r.logger.Info("Bulk insert completed successfully",
		zap.Int("user_id", userID),
		zap.Int("count", len(ids)),
	)

	return ids, nil
}

// InsertFromHabit inserts a task generated from a habit (幂等性由数据库唯一索引保证)
func (r *TaskRepository) InsertFromHabit(ctx context.Context, habitID int, userID int, title string, dueDate time.Time) (int, error) {
	r.logger.Debug("Inserting task from habit",
		zap.Int("habit_id", habitID),
		zap.Int("user_id", userID),
		zap.String("title", title),
		zap.Time("due_date", dueDate),
	)

	query := `
        INSERT INTO tasks (user_id, habit_id, title, due_date, status)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (habit_id, due_date) WHERE status = 'pending' AND habit_id IS NOT NULL
        DO NOTHING
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query,
		userID,
		habitID,
		title,
		dueDate,
		"pending",
	).Scan(&id)

	if err != nil {
		// 如果是没有返回行（冲突导致 DO NOTHING），这是正常的幂等行为
		if err.Error() == "no rows in result set" {
			r.logger.Debug("Task already exists for habit and date (幂等)",
				zap.Int("habit_id", habitID),
				zap.Time("due_date", dueDate),
			)
			return 0, nil // 返回 0 表示已存在，不是错误
		}
		r.logger.Error("Failed to insert task from habit", zap.Error(err))
		return 0, err
	}

	r.logger.Info("Task from habit inserted successfully",
		zap.Int("id", id),
		zap.Int("habit_id", habitID),
		zap.Int("user_id", userID),
	)
	return id, nil
}

// InsertFromProject inserts a task from a project milestone
func (r *TaskRepository) InsertFromProject(ctx context.Context, projectID, milestoneID, userID int, title string, dueDate time.Time, priority string) (int, error) {
	r.logger.Debug("Inserting task from project",
		zap.Int("project_id", projectID),
		zap.Int("milestone_id", milestoneID),
		zap.Int("user_id", userID),
		zap.String("title", title),
		zap.String("priority", priority),
	)

	query := `
        INSERT INTO tasks (user_id, project_id, milestone_id, title, due_date, priority, status)
        VALUES ($1, $2, $3, $4, $5, $6, $7)
        RETURNING id
    `
	var id int
	err := r.db.QueryRow(ctx, query,
		userID,
		projectID,
		milestoneID,
		title,
		dueDate,
		priority,
		"pending",
	).Scan(&id)

	if err != nil {
		r.logger.Error("Failed to insert task from project", zap.Error(err))
		return 0, err
	}

	r.logger.Info("Task from project inserted successfully",
		zap.Int("id", id),
		zap.Int("project_id", projectID),
		zap.Int("milestone_id", milestoneID),
	)
	return id, nil
}

// InsertDependency inserts a task dependency
func (r *TaskRepository) InsertDependency(ctx context.Context, taskID, dependsOnTaskID int) error {
	r.logger.Debug("Inserting task dependency",
		zap.Int("task_id", taskID),
		zap.Int("depends_on_task_id", dependsOnTaskID),
	)

	query := `
        INSERT INTO task_dependencies (task_id, depends_on_task_id)
        VALUES ($1, $2)
        ON CONFLICT DO NOTHING
    `
	_, err := r.db.Exec(ctx, query, taskID, dependsOnTaskID)
	if err != nil {
		r.logger.Error("Failed to insert task dependency", zap.Error(err))
		return err
	}

	return nil
}

// FindByTitleAndProject finds a task by title within a project (for dependency resolution)
func (r *TaskRepository) FindByTitleAndProject(ctx context.Context, projectID int, title string) (int, error) {
	query := `
        SELECT id FROM tasks
        WHERE project_id = $1 AND title = $2
        LIMIT 1
    `
	var id int
	err := r.db.QueryRow(ctx, query, projectID, title).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}
