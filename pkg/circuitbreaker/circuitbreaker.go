package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State 表示熔断器状态
type State int

const (
	StateClosed   State = iota // 关闭：正常状态，允许请求通过
	StateOpen                  // 打开：熔断状态，直接拒绝请求
	StateHalfOpen              // 半开：尝试恢复，允许少量请求通过
)

// Config 熔断器配置
type Config struct {
	// 失败阈值：连续失败多少次后打开熔断器
	FailureThreshold int
	// 成功阈值：半开状态下成功多少次后关闭熔断器
	SuccessThreshold int
	// 超时时间：打开状态持续多久后进入半开状态
	Timeout time.Duration
	// 半开状态下的最大请求数
	HalfOpenMaxRequests int
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		FailureThreshold:    5,                // 连续失败5次后打开
		SuccessThreshold:    2,                // 半开状态下成功2次后关闭
		Timeout:             30 * time.Second, // 打开状态持续30秒
		HalfOpenMaxRequests: 3,                // 半开状态下最多允许3个请求
	}
}

// CircuitBreaker 熔断器
type CircuitBreaker struct {
	config Config

	state         State
	failureCount  int
	successCount  int
	halfOpenCount int
	lastFailTime  time.Time
	lastStateTime time.Time

	mu sync.RWMutex
}

// NewCircuitBreaker 创建新的熔断器
func NewCircuitBreaker(config Config) *CircuitBreaker {
	return &CircuitBreaker{
		config:        config,
		state:         StateClosed,
		lastStateTime: time.Now(),
	}
}

// Execute 执行函数，带熔断保护
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()

	// 检查是否需要状态转换
	cb.checkStateTransition()

	// 根据当前状态决定是否允许执行
	state := cb.state
	switch state {
	case StateOpen:
		cb.mu.Unlock()
		return ErrCircuitBreakerOpen
	case StateHalfOpen:
		if cb.halfOpenCount >= cb.config.HalfOpenMaxRequests {
			cb.mu.Unlock()
			return ErrCircuitBreakerOpen
		}
		cb.halfOpenCount++
	}

	cb.mu.Unlock()

	// 执行函数
	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	// 根据执行结果更新状态
	if err != nil {
		cb.onFailure()
	} else {
		cb.onSuccess()
	}

	return err
}

// checkStateTransition 检查并执行状态转换
func (cb *CircuitBreaker) checkStateTransition() {
	now := time.Now()

	switch cb.state {
	case StateOpen:
		// 打开状态：超时后进入半开状态
		if now.Sub(cb.lastStateTime) >= cb.config.Timeout {
			cb.state = StateHalfOpen
			cb.halfOpenCount = 0
			cb.successCount = 0
			cb.lastStateTime = now
		}
	case StateHalfOpen:
		// 半开状态：成功次数达到阈值后关闭
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			cb.lastStateTime = now
		}
	case StateClosed:
		// 关闭状态：失败次数达到阈值后打开
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = StateOpen
			cb.lastFailTime = now
			cb.lastStateTime = now
		}
	}
}

// onFailure 处理失败
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailTime = time.Now()

	if cb.state == StateHalfOpen {
		// 半开状态下失败，立即打开
		cb.state = StateOpen
		cb.halfOpenCount = 0
		cb.lastStateTime = time.Now()
	}
}

// onSuccess 处理成功
func (cb *CircuitBreaker) onSuccess() {
	cb.failureCount = 0

	if cb.state == StateHalfOpen {
		cb.successCount++
		cb.halfOpenCount--
	} else if cb.state == StateClosed {
		// 关闭状态下成功，重置计数
		cb.failureCount = 0
	}
}

// GetState 获取当前状态（线程安全）
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Reset 重置熔断器
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.halfOpenCount = 0
	cb.lastStateTime = time.Now()
}

// 错误定义
var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
)
