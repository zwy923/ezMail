package util

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strings"

	"github.com/jackc/pgx/v5"
)

// IsRetryableError determines if an error is retryable
// Returns: (isRetryable, errorType)
func IsRetryableError(err error) (bool, string) {
	if err == nil {
		return false, ""
	}

	errStr := err.Error()

	// JSON decode errors - 不可重试（数据格式错误）
	if _, ok := err.(*json.SyntaxError); ok {
		return false, "json_decode_error"
	}
	if _, ok := err.(*json.UnmarshalTypeError); ok {
		return false, "json_decode_error"
	}
	if strings.Contains(errStr, "json:") {
		return false, "json_decode_error"
	}

	// Database errors
	if errors.Is(err, pgx.ErrNoRows) {
		// email_id 不存在 - 不可重试
		return false, "email_not_found"
	}
	if strings.Contains(errStr, "duplicate key") || strings.Contains(errStr, "UNIQUE constraint") {
		// 唯一约束冲突 - 不可重试（幂等性）
		return false, "duplicate_key"
	}
	if strings.Contains(errStr, "connection") || strings.Contains(errStr, "timeout") {
		// DB 连接问题 - 可重试
		return true, "db_connection_error"
	}

	// Network errors - 可重试
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true, "network_timeout"
		}
		return true, "network_error"
	}

	// URL errors - 可重试（配置问题）
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return true, "network_timeout"
		}
		return true, "network_error"
	}

	// Context timeout - 可重试
	if errors.Is(err, context.DeadlineExceeded) {
		return true, "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return false, "context_canceled"
	}

	// HTTP errors - 根据状态码判断
	if strings.Contains(errStr, "agent service returned error") {
		// Agent service 错误 - 可重试（少量）
		return true, "agent_service_error"
	}
	if strings.Contains(errStr, "failed to call agent service") {
		// Agent service 连接失败 - 可重试
		return true, "agent_service_unavailable"
	}

	// 默认：未知错误，保守处理 - 不重试
	return false, "unknown_error"
}

// ShouldRetry checks if an error should be retried based on retry count
func ShouldRetry(retryCount int64, maxRetries int64, isRetryable bool) bool {
	if !isRetryable {
		return false
	}
	return retryCount <= maxRetries
}
