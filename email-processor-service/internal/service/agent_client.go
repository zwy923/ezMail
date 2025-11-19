package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"email-processor-service/internal/model"
	"mygoproject/pkg/circuitbreaker"
	"mygoproject/pkg/metrics"
	"mygoproject/pkg/trace"
)

type AgentClient struct {
	baseURL    string
	httpClient *http.Client
	cb         *circuitbreaker.CircuitBreaker // 熔断器
}

func NewAgentClient(baseURL string) *AgentClient {
	// 创建熔断器，配置更严格的阈值以确保快速失败
	cbConfig := circuitbreaker.Config{
		FailureThreshold:    3,              // 连续失败3次后打开
		SuccessThreshold:    2,              // 半开状态下成功2次后关闭
		Timeout:             30 * time.Second, // 打开状态持续30秒
		HalfOpenMaxRequests: 2,              // 半开状态下最多允许2个请求
	}
	
	return &AgentClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second, // 5秒超时，避免 worker 卡死
		},
		cb: circuitbreaker.NewCircuitBreaker(cbConfig),
	}
}

type EmailInput struct {
    EmailID int    `json:"email_id"`
    UserID  int    `json:"user_id"`
    Subject string `json:"subject"`
    Body    string `json:"body"`
}



// Decide 调用 agent-service 进行决策，带熔断器和 fallback
func (c *AgentClient) Decide(ctx context.Context, email EmailInput) (*model.AgentDecision, error) {
	var decision *model.AgentDecision
	var err error

	// 使用熔断器执行请求
	err = c.cb.Execute(func() error {
		start := time.Now()
		b, marshalErr := json.Marshal(email)
		if marshalErr != nil {
			return marshalErr
		}

		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/decide", bytes.NewReader(b))
		if reqErr != nil {
			return reqErr
		}
		req.Header.Set("Content-Type", "application/json")
		// 传播 trace_id
		if traceID := trace.FromContext(ctx); traceID != "" {
			req.Header.Set(trace.HeaderName(), traceID)
		}

		resp, doErr := c.httpClient.Do(req)
		latency := time.Since(start)
		status := "success"
		
		if doErr != nil {
			status = "error"
			metrics.RecordAgentCallLatency("/decide", status, latency)
			return doErr
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			status = "5xx"
			metrics.RecordAgentCallLatency("/decide", status, latency)
			// 可重试错误
			return fmt.Errorf("agent service 5xx: %d", resp.StatusCode)
		}
		if resp.StatusCode != 200 {
			status = fmt.Sprintf("%d", resp.StatusCode)
			metrics.RecordAgentCallLatency("/decide", status, latency)
			// 不可重试或者单独处理
			return fmt.Errorf("agent service error: %d", resp.StatusCode)
		}

		metrics.RecordAgentCallLatency("/decide", status, latency)
		var decodeErr error
		decision, decodeErr = c.decodeDecision(resp)
		return decodeErr
	})

	// 如果失败（包括熔断器打开），使用 fallback
	if err != nil {
		return c.fallbackDecision(email), nil // 返回 fallback，不返回错误，确保 ingestion-service 继续运行
	}

	return decision, nil
}

// decodeDecision 解码响应
func (c *AgentClient) decodeDecision(resp *http.Response) (*model.AgentDecision, error) {
	var decision model.AgentDecision
	if err := json.NewDecoder(resp.Body).Decode(&decision); err != nil {
		return nil, err
	}
	return &decision, nil
}

// fallbackDecision 返回默认决策（当 agent-service 不可用时）
func (c *AgentClient) fallbackDecision(email EmailInput) *model.AgentDecision {
	// 返回一个保守的默认决策：
	// - 不创建任务（避免误操作）
	// - 不发送通知（避免骚扰）
	// - 优先级设为 MEDIUM（中等优先级）
	// - 分类设为空数组（表示未分类）
	return &model.AgentDecision{
		Categories:        []string{}, // 空分类，表示未分类
		Priority:          "MEDIUM",    // 默认中等优先级
		Summary:           fmt.Sprintf("Agent service unavailable, email not processed: %s", email.Subject),
		ShouldCreateTask:  false,       // 不创建任务，避免误操作
		Task:              nil,
		ShouldNotify:      false,       // 不发送通知，避免骚扰
		NotificationChannel: "",
		NotificationMessage: "",
	}
}