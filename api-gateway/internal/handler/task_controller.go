package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	mqcontracts "mygoproject/contracts/mq"
	"mygoproject/pkg/circuitbreaker"
	"mygoproject/pkg/mq"
	"mygoproject/pkg/outbox"
	"mygoproject/pkg/rbac"
	"mygoproject/pkg/trace"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type TaskController struct {
	db              *pgxpool.Pool
	agentServiceURL string
	taskServiceURL  string
	taskPublisher   *mq.Publisher
	outboxRepo      *outbox.Repository

	httpClient *http.Client
	logger     *zap.Logger
	cb         *circuitbreaker.CircuitBreaker // 熔断器（用于 text-to-tasks）
	cbProject  *circuitbreaker.CircuitBreaker // 熔断器（用于 plan-project）
}

func NewTaskController(db *pgxpool.Pool, agentURL, taskURL string, pub *mq.Publisher, logger *zap.Logger) *TaskController {
	// 为 text-to-tasks 创建熔断器
	cbConfig := circuitbreaker.Config{
		FailureThreshold:    3,
		SuccessThreshold:    2,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 2,
	}

	// 为 plan-project 创建熔断器（可以有不同的配置）
	cbProjectConfig := circuitbreaker.Config{
		FailureThreshold:    3,
		SuccessThreshold:    2,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 2,
	}

	return &TaskController{
		db:              db,
		agentServiceURL: agentURL,
		taskServiceURL:  taskURL,
		taskPublisher:   pub,
		outboxRepo:      outbox.NewRepository(db),
		logger:          logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // LLM 可能需要更长时间
		},
		cb:        circuitbreaker.NewCircuitBreaker(cbConfig),
		cbProject: circuitbreaker.NewCircuitBreaker(cbProjectConfig),
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

	// 使用熔断器调用 agent-service
	var resp *http.Response
	err = tc.cb.Execute(func() error {
		agentURL := tc.agentServiceURL + "/text-to-tasks"
		httpReq, reqErr := http.NewRequestWithContext(c.Request.Context(), "POST", agentURL, bytes.NewReader(reqBody))
		if reqErr != nil {
			return reqErr
		}
		httpReq.Header.Set("Content-Type", "application/json")
		// 传播 trace_id
		if traceID := trace.FromContext(c.Request.Context()); traceID != "" {
			httpReq.Header.Set(trace.HeaderName(), traceID)
		}

		doResp, doErr := tc.httpClient.Do(httpReq)
		if doErr != nil {
			return doErr
		}

		if doResp.StatusCode != http.StatusOK {
			doResp.Body.Close()
			return fmt.Errorf("agent service error: %d", doResp.StatusCode)
		}

		resp = doResp
		return nil
	})

	if err != nil {
		// 检查是否是熔断器打开
		if err == circuitbreaker.ErrCircuitBreakerOpen {
			tc.logger.Warn("Circuit breaker is open for text-to-tasks",
				zap.String("state", "open"),
				zap.Int("user_id", userID),
			)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "agent-service is temporarily unavailable, please try again later",
			})
			return
		}

		tc.logger.Error("Agent service error",
			zap.Error(err),
			zap.Int("user_id", userID),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": "agent-service error"})
		return
	}
	defer resp.Body.Close()

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
	// RBAC 验证：确保 user_id 匹配 token（已在中间件中验证，这里再次确认）
	traceID := trace.FromContext(c.Request.Context())
	for _, habit := range agentResp.Habits {
		habitPayload := mqcontracts.HabitCreatedPayload{
			UserID:            userID, // userID 来自 token，已通过 AuthMiddleware 验证
			Title:             habit.Title,
			RecurrencePattern: habit.RecurrencePattern,
			TraceID:           traceID,
		}

		// 双重验证：确保 payload 中的 user_id 与 token 中的 user_id 匹配
		if err := rbac.ValidateUserIDInPayload(userID, habitPayload.UserID); err != nil {
			tc.logger.Error("User ID mismatch in habit.created payload",
				zap.Error(err),
				zap.Int("token_user_id", userID),
				zap.Int("payload_user_id", habitPayload.UserID),
			)
			c.JSON(http.StatusForbidden, gin.H{"error": "user_id mismatch"})
			return
		}

		ctx := trace.WithContext(c.Request.Context(), traceID)
		if err := tc.taskPublisher.PublishWithContext(ctx, "habit.created", habitPayload); err != nil {
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
	// RBAC 验证：确保 user_id 匹配 token
	if len(agentResp.Tasks) > 0 {
		bulkPayload := mqcontracts.TaskBulkCreatedPayload{
			UserID:  userID, // userID 来自 token，已通过 AuthMiddleware 验证
			Tasks:   agentResp.Tasks,
			TraceID: traceID,
		}

		// 双重验证：确保 payload 中的 user_id 与 token 中的 user_id 匹配
		if err := rbac.ValidateUserIDInPayload(userID, bulkPayload.UserID); err != nil {
			tc.logger.Error("User ID mismatch in task.bulk_created payload",
				zap.Error(err),
				zap.Int("token_user_id", userID),
				zap.Int("payload_user_id", bulkPayload.UserID),
			)
			c.JSON(http.StatusForbidden, gin.H{"error": "user_id mismatch"})
			return
		}

		ctx := trace.WithContext(c.Request.Context(), traceID)
		if err := tc.taskPublisher.PublishWithContext(ctx, "task.bulk_created", bulkPayload); err != nil {
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
	// 传播 trace_id
	if traceID := trace.FromContext(c.Request.Context()); traceID != "" {
		req.Header.Set(trace.HeaderName(), traceID)
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
	// 传播 trace_id
	if traceID := trace.FromContext(c.Request.Context()); traceID != "" {
		req.Header.Set(trace.HeaderName(), traceID)
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

	// 使用熔断器调用 agent-service
	var resp *http.Response
	err = tc.cbProject.Execute(func() error {
		agentURL := tc.agentServiceURL + "/plan-project"
		httpReq, reqErr := http.NewRequestWithContext(c.Request.Context(), "POST", agentURL, bytes.NewReader(reqBody))
		if reqErr != nil {
			return reqErr
		}
		httpReq.Header.Set("Content-Type", "application/json")
		// 传播 trace_id
		if traceID := trace.FromContext(c.Request.Context()); traceID != "" {
			httpReq.Header.Set(trace.HeaderName(), traceID)
		}

		doResp, doErr := tc.httpClient.Do(httpReq)
		if doErr != nil {
			return doErr
		}

		if doResp.StatusCode != http.StatusOK {
			doResp.Body.Close()
			return fmt.Errorf("agent service error: %d", doResp.StatusCode)
		}

		resp = doResp
		return nil
	})

	if err != nil {
		// 检查是否是熔断器打开
		if err == circuitbreaker.ErrCircuitBreakerOpen {
			tc.logger.Warn("Circuit breaker is open for plan-project",
				zap.String("state", "open"),
				zap.Int("user_id", userID),
			)
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "agent-service is temporarily unavailable, please try again later",
			})
			return
		}

		tc.logger.Error("Agent service error",
			zap.Error(err),
			zap.Int("user_id", userID),
		)
		c.JSON(http.StatusBadGateway, gin.H{"error": "agent-service error"})
		return
	}
	defer resp.Body.Close()

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

	// Step 3: Insert project.created event to outbox (使用事务)
	traceID := trace.FromContext(c.Request.Context())
	projectPayload := mqcontracts.ProjectCreatedPayload{
		UserID:      userID,
		Title:       agentResp.Project.Title,
		Description: agentResp.Project.Description,
		TargetDays:  agentResp.Project.TargetDays,
		Milestones:  milestones,
		TraceID:     traceID,
	}

	// RBAC 验证
	if err := rbac.ValidateUserIDInPayload(userID, projectPayload.UserID); err != nil {
		tc.logger.Error("User ID mismatch in project.created payload", zap.Error(err))
		c.JSON(http.StatusForbidden, gin.H{"error": "user_id mismatch"})
		return
	}

	// 使用事务写入 Outbox
	tx, err := tc.db.Begin(c.Request.Context())
	if err != nil {
		tc.logger.Error("Failed to begin transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create transaction"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	if err := outbox.InsertEventInTx(c.Request.Context(), tx, tc.outboxRepo, "project", nil, "project.created", projectPayload); err != nil {
		tc.logger.Error("Failed to insert project.created to outbox", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create project"})
		return
	}

	if err := tx.Commit(c.Request.Context()); err != nil {
		tc.logger.Error("Failed to commit transaction", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to commit transaction"})
		return
	}

	tc.logger.Info("Project created event inserted to outbox",
		zap.Int("user_id", userID),
		zap.String("title", agentResp.Project.Title),
		zap.Int("milestone_count", len(milestones)),
	)

	c.JSON(http.StatusOK, gin.H{
		"message": "Project created successfully",
		"project": agentResp.Project,
	})
}
