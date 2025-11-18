package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"worker-service/internal/model"
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

type EmailInput struct {
    EmailID int    `json:"email_id"`
    UserID  int    `json:"user_id"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}



func (c *AgentClient) Decide(ctx context.Context, email EmailInput) (*model.AgentDecision, error) {
    b, err := json.Marshal(email)
    if err != nil {
        return nil, err
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/decide", bytes.NewReader(b))
    if err != nil {
        return nil, err
    }
    req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    if resp.StatusCode >= 500 {
        // 可重试错误
        return nil, fmt.Errorf("agent service 5xx: %d", resp.StatusCode)
    }
    if resp.StatusCode != 200 {
        // 不可重试或者单独处理
        return nil, fmt.Errorf("agent service error: %d", resp.StatusCode)
    }

    var decision model.AgentDecision
    if err := json.NewDecoder(resp.Body).Decode(&decision); err != nil {
        return nil, err
    }
    return &decision, nil
}