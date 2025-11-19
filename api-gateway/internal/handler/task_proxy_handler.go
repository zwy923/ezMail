package handler

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskProxyHandler struct {
	taskServiceURL string
	httpClient     *http.Client
}

func NewTaskProxyHandler(url string) *TaskProxyHandler {
	return &TaskProxyHandler{
		taskServiceURL: url,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetTasks proxies GET /tasks to task-service
func (h *TaskProxyHandler) GetTasks(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	// Forward to task-service
	url := h.taskServiceURL + "/tasks?user_id=" + fmt.Sprintf("%d", userID.(int))
	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	// Forward request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "task-service unreachable"})
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	// Forward response
	c.Data(resp.StatusCode, "application/json", body)
}

// CompleteTask proxies POST /tasks/:id/complete to task-service
func (h *TaskProxyHandler) CompleteTask(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "task id required"})
		return
	}

	// Forward to task-service
	url := h.taskServiceURL + "/tasks/" + taskID + "/complete"
	req, err := http.NewRequestWithContext(c.Request.Context(), "POST", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request"})
		return
	}

	// Forward request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "task-service unreachable"})
		return
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	// Forward response
	c.Data(resp.StatusCode, "application/json", body)
}
