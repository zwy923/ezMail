package handler

import (
	"net/http"
	"strconv"
	"task-service/internal/repository"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TaskHandler struct {
	repo   *repository.TaskRepository
	logger *zap.Logger
}

func NewTaskHandler(repo *repository.TaskRepository, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{repo: repo, logger: logger}
}

func (h *TaskHandler) ListTasks(c *gin.Context) {
	userIDRaw := c.Query("user_id")
	h.logger.Info("ListTasks request received",
		zap.String("user_id", userIDRaw),
		zap.String("client_ip", c.ClientIP()),
	)

	if userIDRaw == "" {
		h.logger.Warn("ListTasks: user_id is required")
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}

	userID, err := strconv.Atoi(userIDRaw)
	if err != nil {
		h.logger.Warn("ListTasks: invalid user_id format",
			zap.String("user_id", userIDRaw),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	tasks, err := h.repo.ListByUser(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("ListTasks: failed to fetch tasks",
			zap.Int("user_id", userID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch tasks"})
		return
	}

	h.logger.Info("ListTasks: success",
		zap.Int("user_id", userID),
		zap.Int("task_count", len(tasks)),
	)
	c.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
	})
}

func (h *TaskHandler) CompleteTask(c *gin.Context) {
	idStr := c.Param("id")
	h.logger.Info("CompleteTask request received",
		zap.String("task_id", idStr),
		zap.String("client_ip", c.ClientIP()),
	)

	taskID, err := strconv.Atoi(idStr)
	if err != nil {
		h.logger.Warn("CompleteTask: invalid task id format",
			zap.String("task_id", idStr),
			zap.Error(err),
		)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.repo.MarkCompleted(c.Request.Context(), taskID); err != nil {
		h.logger.Error("CompleteTask: failed to mark task as completed",
			zap.Int("task_id", taskID),
			zap.Error(err),
		)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to complete task"})
		return
	}

	h.logger.Info("CompleteTask: success", zap.Int("task_id", taskID))
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
