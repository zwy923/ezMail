package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"task-runner-service/internal/repository"
	"mygoproject/pkg/mq"

	"go.uber.org/zap"
)

type Orchestrator struct {
	taskRepo     *repository.TaskRepository
	habitRepo    *repository.HabitRepository
	publisher    *mq.Publisher
	logger       *zap.Logger
}

func NewOrchestrator(
	taskRepo *repository.TaskRepository,
	habitRepo *repository.HabitRepository,
	publisher *mq.Publisher,
	logger *zap.Logger,
) *Orchestrator {
	return &Orchestrator{
		taskRepo:  taskRepo,
		habitRepo: habitRepo,
		publisher: publisher,
		logger:    logger,
	}
}

// CheckAndMarkOverdue checks for expired tasks and publishes task.overdue events
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

	// Mark as overdue in database
	if err := o.taskRepo.MarkExpired(ctx); err != nil {
		o.logger.Error("Failed to mark tasks as overdue", zap.Error(err))
		return err
	}

	// Publish task.overdue events
	for _, taskID := range taskIDs {
		payload := map[string]interface{}{
			"task_id": taskID,
		}
		if err := o.publisher.Publish("task.overdue", payload); err != nil {
			o.logger.Error("Failed to publish task.overdue event",
				zap.Int("task_id", taskID),
				zap.Error(err),
			)
			continue
		}
		o.logger.Info("Published task.overdue event",
			zap.Int("task_id", taskID),
		)
	}

	o.logger.Info("Overdue check completed",
		zap.Int("overdue_count", len(taskIDs)),
	)
	return nil
}

// CheckAndUnlockTasks checks for tasks with completed dependencies and publishes task.unlocked events
func (o *Orchestrator) CheckAndUnlockTasks(ctx context.Context) error {
	o.logger.Info("Checking for unlockable tasks...")

	tasks, err := o.taskRepo.ListTasksWithDependencies(ctx)
	if err != nil {
		o.logger.Error("Failed to list tasks with dependencies", zap.Error(err))
		return err
	}

	unlockedCount := 0
	for _, task := range tasks {
		// If all dependencies are completed, task is unlocked
		if task.DepCount > 0 && task.CompletedDepCount == task.DepCount {
			payload := map[string]interface{}{
				"task_id": task.ID,
				"user_id": task.UserID,
				"title":   task.Title,
			}
			if err := o.publisher.Publish("task.unlocked", payload); err != nil {
				o.logger.Error("Failed to publish task.unlocked event",
					zap.Int("task_id", task.ID),
					zap.Error(err),
				)
				continue
			}
			unlockedCount++
			o.logger.Info("Published task.unlocked event",
				zap.Int("task_id", task.ID),
				zap.String("title", task.Title),
			)
		}
	}

	o.logger.Info("Unlock check completed",
		zap.Int("unlocked_count", unlockedCount),
		zap.Int("total_checked", len(tasks)),
	)
	return nil
}

// GenerateHabitTasks generates tasks for habits that should occur today and publishes habit.task.generated events
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

	generatedCount := 0
	for _, habit := range habits {
		if o.shouldGenerateToday(habit.RecurrencePattern, today) {
			dueDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
			
			payload := map[string]interface{}{
				"habit_id":  habit.ID,
				"user_id":   habit.UserID,
				"title":     habit.Title,
				"due_date":  dueDate.Format("2006-01-02"),
			}
			
			if err := o.publisher.Publish("habit.task.generated", payload); err != nil {
				o.logger.Error("Failed to publish habit.task.generated event",
					zap.Int("habit_id", habit.ID),
					zap.Error(err),
				)
				continue
			}
			
			generatedCount++
			o.logger.Info("Published habit.task.generated event",
				zap.Int("habit_id", habit.ID),
				zap.String("title", habit.Title),
			)
		}
	}

	o.logger.Info("Habit task generation completed",
		zap.Int("total_habits", len(habits)),
		zap.Int("generated_count", generatedCount),
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

