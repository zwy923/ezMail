package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type AgentClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAgentClient(baseURL string) *AgentClient {
	return &AgentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // 5秒超时，避免 worker 卡死
		},
	}
}

type ClassifyRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type ClassifyResponse struct {
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
}

// ClassifyEmail calls the agent-service to classify an email
func (c *AgentClient) ClassifyEmail(ctx context.Context, subject, body string) (*ClassifyResponse, error) {
	reqBody := ClassifyRequest{
		Subject: subject,
		Body:    body,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := c.baseURL + "/classify"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// 检查是否是超时错误（HTTP client timeout 或 context deadline）
		errStr := err.Error()
		if ctx.Err() == context.DeadlineExceeded || 
		   strings.Contains(errStr, "timeout") || 
		   strings.Contains(errStr, "deadline exceeded") ||
		   strings.Contains(errStr, "context deadline exceeded") {
			return nil, fmt.Errorf("agent service timeout: %w", err)
		}
		return nil, fmt.Errorf("failed to call agent service: %w", err)
	}
	defer resp.Body.Close()

	// 处理非 200 状态码
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		// 500 错误视为可重试
		if resp.StatusCode >= 500 {
			return nil, fmt.Errorf("agent service returned 5xx error: %d - %s", resp.StatusCode, string(bodyBytes))
		}
		// 4xx 错误视为不可重试
		return nil, fmt.Errorf("agent service returned error: %d - %s", resp.StatusCode, string(bodyBytes))
	}

	var result ClassifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

