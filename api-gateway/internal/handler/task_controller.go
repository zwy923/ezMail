package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	mqcontracts "mygoproject/contracts/mq"
	"mygoproject/pkg/mq"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type TaskController struct {
	agentServiceURL string
	taskServiceURL  string
	taskPublisher   *mq.Publisher

	httpClient *http.Client
	logger     *zap.Logger
}

func NewTaskController(agentURL, taskURL string, pub *mq.Publisher, logger *zap.Logger) *TaskController {
	return &TaskController{
		agentServiceURL: agentURL,
		taskServiceURL:  taskURL,
		taskPublisher:   pub,
		logger:          logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // LLM 可能需要更长时间
		},
	}
}

// getUserID 统一的 userID 读取工具（避免重复代码）
func (tc *TaskController) getUserID(c *gin.Context) (int, bool) {
	userID, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return 0, false
	}
	return userID.(int), true
}

// CreateTasksFromText handles POST /tasks/from-text
// 功能：调用 agent-service 解析文本，然后发布 task.bulk_created 事件到 MQ
func (tc *TaskController) CreateTasksFromText(c *gin.Context) {
	userID, ok := tc.getUserID(c)
	if !ok {
		return
	}

	var req struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tc.logger.Info("Text-to-tasks request received",
		zap.Int("user_id", userID),
		zap.Int("text_length", len(req.Text)),
	)

	// Step 1: Call agent-service to parse text
	agentReq := map[string]interface{}{
		"user_id": userID,
		"text":    req.Text,
	}

	reqBody, err := json.Marshal(agentReq)
	if err != nil {
		tc.logger.Error("Failed to marshal agent request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	agentURL := tc.agentServiceURL + "/text-to-tasks"
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", agentURL, bytes.NewReader(reqBody))
	if err != nil {
		tc.logger.Error("Failed to create HTTP request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := tc.httpClient.Do(httpReq)
	if err != nil {
		tc.logger.Error("Agent service unreachable", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "agent-service unreachable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tc.logger.Error("Agent service error",
			zap.Int("status_code", resp.StatusCode),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": "agent-service error"})
		return
	}

	var agentResp struct {
		Tasks  []mqcontracts.TaskItem `json:"tasks"`
		Habits []struct {
			Title             string `json:"title"`
			RecurrencePattern string `json:"recurrence_pattern"`
		} `json:"habits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		tc.logger.Error("Failed to decode agent response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	// Step 2: Publish habit.created events
	for _, habit := range agentResp.Habits {
		habitPayload := mqcontracts.HabitCreatedPayload{
			UserID:            userID,
			Title:             habit.Title,
			RecurrencePattern: habit.RecurrencePattern,
		}
		if err := tc.taskPublisher.Publish("habit.created", habitPayload); err != nil {
			tc.logger.Error("Failed to publish habit.created event", zap.Error(err))
			// Continue processing other habits and tasks
		} else {
			tc.logger.Info("Habit created event published",
				zap.Int("user_id", userID),
				zap.String("title", habit.Title),
				zap.String("recurrence", habit.RecurrencePattern),
			)
		}
	}

	// Step 3: Publish task.bulk_created event (if any tasks)
	if len(agentResp.Tasks) > 0 {
		bulkPayload := mqcontracts.TaskBulkCreatedPayload{
			UserID: userID,
			Tasks:  agentResp.Tasks,
		}

		if err := tc.taskPublisher.Publish("task.bulk_created", bulkPayload); err != nil {
			tc.logger.Error("Failed to publish task.bulk_created event", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tasks"})
			return
		}

		tc.logger.Info("Task bulk created event published",
			zap.Int("user_id", userID),
			zap.Int("task_count", len(agentResp.Tasks)),
		)
	}

	// Prepare response
	response := gin.H{
		"message": fmt.Sprintf("Created %d tasks and %d habits", len(agentResp.Tasks), len(agentResp.Habits)),
		"tasks":   agentResp.Tasks,
		"habits":  agentResp.Habits,
	}

	if len(agentResp.Tasks) == 0 && len(agentResp.Habits) == 0 {
		response["message"] = "No tasks or habits found in the text"
	}

	c.JSON(http.StatusOK, response)
}

// GetTasks handles GET /tasks
// 功能：代理请求到 task-service
func (tc *TaskController) GetTasks(c *gin.Context) {
	userID, ok := tc.getUserID(c)
	if !ok {
		return
	}

	// Forward to task-service
	url := fmt.Sprintf("%s/tasks?user_id=%d", tc.taskServiceURL, userID)
	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, nil)
	if err != nil {
		tc.logger.Error("Failed to create request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	// Forward request
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		tc.logger.Error("Task service unreachable", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "task-service unreachable"})
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tc.logger.Error("Failed to read response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	// Forward response
	c.Data(resp.StatusCode, "application/json", body)
}

// CompleteTask handles POST /tasks/:id/complete
// 功能：代理请求到 task-service
func (tc *TaskController) CompleteTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	// Forward to task-service
	url := fmt.Sprintf("%s/tasks/%s/complete", tc.taskServiceURL, taskID)
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, nil)
	if err != nil {
		tc.logger.Error("Failed to create request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	// Forward request
	resp, err := tc.httpClient.Do(req)
	if err != nil {
		tc.logger.Error("Task service unreachable", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "task-service unreachable"})
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		tc.logger.Error("Failed to read response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	// Forward response
	c.Data(resp.StatusCode, "application/json", body)
}

// PlanProject handles POST /tasks/plan-project
// 功能：调用 agent-service 规划项目，然后发布 project.created 事件到 MQ
func (tc *TaskController) PlanProject(c *gin.Context) {
	userID, ok := tc.getUserID(c)
	if !ok {
		return
	}

	var req struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	tc.logger.Info("Project planning request received",
		zap.Int("user_id", userID),
		zap.Int("text_length", len(req.Text)),
	)

	// Step 1: Call agent-service to plan project
	agentReq := map[string]interface{}{
		"user_id": userID,
		"text":    req.Text,
	}

	reqBody, err := json.Marshal(agentReq)
	if err != nil {
		tc.logger.Error("Failed to marshal agent request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	agentURL := tc.agentServiceURL + "/plan-project"
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), "POST", agentURL, bytes.NewReader(reqBody))
	if err != nil {
		tc.logger.Error("Failed to create HTTP request", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := tc.httpClient.Do(httpReq)
	if err != nil {
		tc.logger.Error("Agent service unreachable", zap.Error(err))
		c.JSON(http.StatusBadGateway, gin.H{"error": "agent-service unreachable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		tc.logger.Error("Agent service error",
			zap.Int("status_code", resp.StatusCode),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": "agent-service error"})
		return
	}

	var agentResp struct {
		Project struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			TargetDays  int    `json:"target_days"`
			Milestones  []struct {
				Title     string `json:"title"`
				Order     int    `json:"order"`
				DueInDays int    `json:"due_in_days"`
				Tasks     []struct {
					Title     string   `json:"title"`
					DueInDays int      `json:"due_in_days"`
					Priority  string   `json:"priority"`
					DependsOn []string `json:"depends_on"`
				} `json:"tasks"`
			} `json:"milestones"`
		} `json:"project"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&agentResp); err != nil {
		tc.logger.Error("Failed to decode agent response", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse response"})
		return
	}

	// Step 2: Convert to MQ contract format
	milestones := make([]mqcontracts.Milestone, len(agentResp.Project.Milestones))
	for i, m := range agentResp.Project.Milestones {
		tasks := make([]mqcontracts.ProjectTask, len(m.Tasks))
		for j, t := range m.Tasks {
			tasks[j] = mqcontracts.ProjectTask{
				Title:     t.Title,
				DueInDays: t.DueInDays,
				Priority:  t.Priority,
				DependsOn: t.DependsOn,
			}
		}
		milestones[i] = mqcontracts.Milestone{
			Title:     m.Title,
			Order:     m.Order,
			DueInDays: m.DueInDays,
			Tasks:     tasks,
		}
	}

	// Step 3: Publish project.created event
	projectPayload := mqcontracts.ProjectCreatedPayload{
		UserID:      userID,
		Title:       agentResp.Project.Title,
		Description: agentResp.Project.Description,
		TargetDays:  agentResp.Project.TargetDays,
		Milestones:  milestones,
	}

	if err := tc.taskPublisher.Publish("project.created", projectPayload); err != nil {
		tc.logger.Error("Failed to publish project.created event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create project"})
		return
	}

	tc.logger.Info("Project created event published",
		zap.Int("user_id", userID),
		zap.String("title", agentResp.Project.Title),
		zap.Int("milestone_count", len(milestones)),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Project created successfully",
		"project": agentResp.Project,
	})
}
