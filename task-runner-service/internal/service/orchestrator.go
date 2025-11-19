package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"task-runner-service/internal/repository"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/outbox"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type Orchestrator struct {
	db           *pgxpool.Pool
	taskRepo     *repository.TaskRepository
	habitRepo    *repository.HabitRepository
	publisher    *mq.Publisher
	outboxRepo   *outbox.Repository
	logger       *zap.Logger
}

func NewOrchestrator(
	db *pgxpool.Pool,
	taskRepo *repository.TaskRepository,
	habitRepo *repository.HabitRepository,
	publisher *mq.Publisher,
	logger *zap.Logger,
) *Orchestrator {
	return &Orchestrator{
		db:         db,
		taskRepo:   taskRepo,
		habitRepo:  habitRepo,
		publisher:  publisher,
		outboxRepo: outbox.NewRepository(db),
		logger:     logger,
	}
}

// CheckAndMarkOverdue checks for expired tasks and publishes task.overdue events using Outbox
func (o *Orchestrator) CheckAndMarkOverdue(ctx context.Context) error {
	o.logger.Info("Checking for overdue tasks...")

	taskIDs, err := o.taskRepo.ListExpiredPendingTasks(ctx)
	if err != nil {
		o.logger.Error("Failed to list expired tasks", zap.Error(err))
		return err
	}

	if len(taskIDs) == 0 {
		o.logger.Debug("No overdue tasks found")
		return nil
	}

	// 使用事务：标记为 overdue + 写入 outbox
	tx, err := o.db.Begin(ctx)
	if err != nil {
		o.logger.Error("Failed to begin transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback(ctx)

	// Mark as overdue in database (in transaction)
	if err := o.taskRepo.MarkExpiredTx(ctx, tx); err != nil {
		o.logger.Error("Failed to mark tasks as overdue", zap.Error(err))
		return err
	}

	// Insert task.overdue events to outbox (in transaction)
	for _, taskID := range taskIDs {
		payload := map[string]interface{}{
			"task_id": taskID,
		}
		taskID64 := int64(taskID)
		if err := outbox.InsertEventInTx(ctx, tx, o.outboxRepo, "task", &taskID64, "task.overdue", payload); err != nil {
			o.logger.Error("Failed to insert task.overdue to outbox",
				zap.Int("task_id", taskID),
				zap.Error(err),
			)
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		o.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	o.logger.Info("Overdue check completed",
		zap.Int("overdue_count", len(taskIDs)),
	)
	return nil
}

// CheckAndUnlockTasks checks for tasks with completed dependencies and publishes task.unlocked events using Outbox
func (o *Orchestrator) CheckAndUnlockTasks(ctx context.Context) error {
	o.logger.Info("Checking for unlockable tasks...")

	tasks, err := o.taskRepo.ListTasksWithDependencies(ctx)
	if err != nil {
		o.logger.Error("Failed to list tasks with dependencies", zap.Error(err))
		return err
	}

	// 找出需要解锁的任务
	var unlockedTasks []struct {
		ID     int
		UserID int
		Title  string
	}
	for _, task := range tasks {
		if task.DepCount > 0 && task.CompletedDepCount == task.DepCount {
			unlockedTasks = append(unlockedTasks, struct {
				ID     int
				UserID int
				Title  string
			}{ID: task.ID, UserID: task.UserID, Title: task.Title})
		}
	}

	if len(unlockedTasks) == 0 {
		o.logger.Debug("No unlockable tasks found")
		return nil
	}

	// 使用事务写入 outbox
	tx, err := o.db.Begin(ctx)
	if err != nil {
		o.logger.Error("Failed to begin transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback(ctx)

	// Insert task.unlocked events to outbox (in transaction)
	for _, task := range unlockedTasks {
		payload := map[string]interface{}{
			"task_id": task.ID,
			"user_id": task.UserID,
			"title":   task.Title,
		}
		taskID64 := int64(task.ID)
		if err := outbox.InsertEventInTx(ctx, tx, o.outboxRepo, "task", &taskID64, "task.unlocked", payload); err != nil {
			o.logger.Error("Failed to insert task.unlocked to outbox",
				zap.Int("task_id", task.ID),
				zap.Error(err),
			)
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		o.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	o.logger.Info("Unlock check completed",
		zap.Int("unlocked_count", len(unlockedTasks)),
		zap.Int("total_checked", len(tasks)),
	)
	return nil
}

// GenerateHabitTasks generates tasks for habits that should occur today and publishes habit.task.generated events using Outbox
func (o *Orchestrator) GenerateHabitTasks(ctx context.Context) error {
	today := time.Now()
	o.logger.Info("Generating habit tasks for today",
		zap.String("date", today.Format("2006-01-02")),
	)

	habits, err := o.habitRepo.ListAllActive(ctx)
	if err != nil {
		o.logger.Error("Failed to list habits", zap.Error(err))
		return err
	}

	// 找出今天需要生成的习惯
	var habitsToGenerate []struct {
		ID     int
		UserID int
		Title  string
	}
	dueDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	
	for _, habit := range habits {
		if o.shouldGenerateToday(habit.RecurrencePattern, today) {
			habitsToGenerate = append(habitsToGenerate, struct {
				ID     int
				UserID int
				Title  string
			}{ID: habit.ID, UserID: habit.UserID, Title: habit.Title})
		}
	}

	if len(habitsToGenerate) == 0 {
		o.logger.Debug("No habits to generate today")
		return nil
	}

	// 使用事务写入 outbox
	tx, err := o.db.Begin(ctx)
	if err != nil {
		o.logger.Error("Failed to begin transaction", zap.Error(err))
		return err
	}
	defer tx.Rollback(ctx)

	// Insert habit.task.generated events to outbox (in transaction)
	for _, habit := range habitsToGenerate {
		payload := map[string]interface{}{
			"habit_id": habit.ID,
			"user_id":  habit.UserID,
			"title":    habit.Title,
			"due_date": dueDate.Format("2006-01-02"),
		}
		habitID64 := int64(habit.ID)
		if err := outbox.InsertEventInTx(ctx, tx, o.outboxRepo, "habit", &habitID64, "habit.task.generated", payload); err != nil {
			o.logger.Error("Failed to insert habit.task.generated to outbox",
				zap.Int("habit_id", habit.ID),
				zap.Error(err),
			)
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		o.logger.Error("Failed to commit transaction", zap.Error(err))
		return err
	}

	o.logger.Info("Habit task generation completed",
		zap.Int("total_habits", len(habits)),
		zap.Int("generated_count", len(habitsToGenerate)),
	)

	return nil
}

func (o *Orchestrator) shouldGenerateToday(pattern string, today time.Time) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	weekday := today.Weekday()

	switch pattern {
	case "daily":
		return true
	case "weekly monday":
		return weekday == time.Monday
	case "weekly tuesday":
		return weekday == time.Tuesday
	case "weekly wednesday":
		return weekday == time.Wednesday
	case "weekly thursday":
		return weekday == time.Thursday
	case "weekly friday":
		return weekday == time.Friday
	case "weekly saturday":
		return weekday == time.Saturday
	case "weekly sunday":
		return weekday == time.Sunday
	default:
		if strings.HasPrefix(pattern, "monthly ") {
			dayStr := strings.TrimPrefix(pattern, "monthly ")
			var day int
			if _, err := fmt.Sscanf(dayStr, "%d", &day); err == nil {
				return today.Day() == day
			}
		}
		return false
	}
}

