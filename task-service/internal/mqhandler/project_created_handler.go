package mqhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mqcontracts "mygoproject/contracts/mq"
	"task-service/internal/model"
	"task-service/internal/repository"

	"go.uber.org/zap"
)

type ProjectCreatedHandler struct {
	projectRepo   *repository.ProjectRepository
	milestoneRepo *repository.MilestoneRepository
	taskRepo      *repository.TaskRepository
	logger        *zap.Logger
}

func NewProjectCreatedHandler(
	projectRepo *repository.ProjectRepository,
	milestoneRepo *repository.MilestoneRepository,
	taskRepo *repository.TaskRepository,
	logger *zap.Logger,
) *ProjectCreatedHandler {
	return &ProjectCreatedHandler{
		projectRepo:   projectRepo,
		milestoneRepo: milestoneRepo,
		taskRepo:      taskRepo,
		logger:        logger,
	}
}

func (h *ProjectCreatedHandler) Handle(ctx context.Context, raw json.RawMessage) error {
	var p mqcontracts.ProjectCreatedPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		h.logger.Error("Failed to unmarshal ProjectCreatedPayload", zap.Error(err))
		return err
	}

	h.logger.Info("Handling project.created event",
		zap.Int("user_id", p.UserID),
		zap.String("title", p.Title),
		zap.Int("milestone_count", len(p.Milestones)),
		zap.String("trace_id", p.TraceID),
	)

	// RBAC 验证：验证 user_id 有效性（MQ 事件来自内部服务，但记录用于审计）
	if p.UserID <= 0 {
		h.logger.Error("Invalid user_id in project.created event",
			zap.Int("user_id", p.UserID),
		)
		return fmt.Errorf("invalid user_id: %d", p.UserID)
	}

	// Step 1: Create project
	now := time.Now()
	targetDate := now.AddDate(0, 0, p.TargetDays)
	project := &model.Project{
		UserID:      p.UserID,
		Title:       p.Title,
		Description: p.Description,
		TargetDate:  targetDate,
		Status:      "active",
	}

	projectID, err := h.projectRepo.Insert(ctx, project)
	if err != nil {
		h.logger.Error("Failed to insert project", zap.Error(err))
		return err
	}

	h.logger.Info("Project created",
		zap.Int("project_id", projectID),
		zap.Int("user_id", p.UserID),
	)

	// Step 2: Create milestones and tasks
	taskTitleToID := make(map[string]int) // Map task title to task ID for dependency resolution

	for _, milestoneData := range p.Milestones {
		milestoneTargetDate := now.AddDate(0, 0, milestoneData.DueInDays)
		milestone := &model.Milestone{
			ProjectID:  projectID,
			Title:      milestoneData.Title,
			PhaseOrder: milestoneData.Order,
			TargetDate: milestoneTargetDate,
			Status:     "pending",
		}

		milestoneID, err := h.milestoneRepo.Insert(ctx, milestone)
		if err != nil {
			h.logger.Error("Failed to insert milestone", zap.Error(err))
			return err
		}

		h.logger.Info("Milestone created",
			zap.Int("milestone_id", milestoneID),
			zap.Int("project_id", projectID),
			zap.String("title", milestoneData.Title),
		)

		// Step 3: Create tasks for this milestone
		for _, taskData := range milestoneData.Tasks {
			taskDueDate := now.AddDate(0, 0, taskData.DueInDays)
			taskID, err := h.taskRepo.InsertFromProject(
				ctx,
				projectID,
				milestoneID,
				p.UserID,
				taskData.Title,
				taskDueDate,
				taskData.Priority,
			)
			if err != nil {
				h.logger.Error("Failed to insert task from project",
					zap.Error(err),
					zap.String("title", taskData.Title),
				)
				return err
			}

			// Store task ID for dependency resolution
			taskTitleToID[taskData.Title] = taskID

			h.logger.Debug("Task created",
				zap.Int("task_id", taskID),
				zap.String("title", taskData.Title),
				zap.Int("milestone_id", milestoneID),
			)
		}
	}

	// Step 4: Create task dependencies
	for _, milestoneData := range p.Milestones {
		for _, taskData := range milestoneData.Tasks {
			// Find current task ID
			currentTaskID, exists := taskTitleToID[taskData.Title]
			if !exists {
				h.logger.Warn("Task not found for dependency",
					zap.String("task_title", taskData.Title),
				)
				continue
			}

			// Create dependencies
			for _, dependsOnTitle := range taskData.DependsOn {
				dependsOnTaskID, exists := taskTitleToID[dependsOnTitle]
				if !exists {
					h.logger.Warn("Dependency task not found",
						zap.String("depends_on_title", dependsOnTitle),
						zap.String("task_title", taskData.Title),
					)
					continue
				}

				if err := h.taskRepo.InsertDependency(ctx, currentTaskID, dependsOnTaskID); err != nil {
					h.logger.Error("Failed to insert task dependency",
						zap.Error(err),
						zap.Int("task_id", currentTaskID),
						zap.Int("depends_on_task_id", dependsOnTaskID),
					)
					// Continue with other dependencies
					continue
				}

				h.logger.Debug("Task dependency created",
					zap.Int("task_id", currentTaskID),
					zap.Int("depends_on_task_id", dependsOnTaskID),
				)
			}
		}
	}

	h.logger.Info("Project created successfully with all milestones and tasks",
		zap.Int("project_id", projectID),
		zap.Int("user_id", p.UserID),
		zap.Int("milestone_count", len(p.Milestones)),
	)

	return nil
}

