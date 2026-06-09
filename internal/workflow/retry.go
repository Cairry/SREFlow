package workflow

import (
	"math"
	"time"
)

// RetryPolicy 重试策略
type RetryPolicy struct {
	maxRetries  int
	baseDelay   time.Duration
	maxDelay    time.Duration
	backoffType BackoffType
	multiplier  float64
	jitter      bool
}

// BackoffType 退避策略类型
type BackoffType string

const (
	BackoffTypeFixed       BackoffType = "fixed"       // 固定延迟
	BackoffTypeLinear      BackoffType = "linear"      // 线性退避
	BackoffTypeExponential BackoffType = "exponential" // 指数退避
)

// NewRetryPolicy 创建新的重试策略
func NewRetryPolicy(config *EngineConfig) *RetryPolicy {
	return &RetryPolicy{
		maxRetries:  config.MaxRetries,
		baseDelay:   config.RetryDelay,
		maxDelay:    config.RetryDelay * 10, // 最大延迟为基础延迟的10倍
		backoffType: BackoffTypeExponential,
		multiplier:  2.0,
		jitter:      true,
	}
}

// GetDelay 获取重试延迟
func (rp *RetryPolicy) GetDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return 0
	}

	var delay time.Duration

	switch rp.backoffType {
	case BackoffTypeFixed:
		delay = rp.baseDelay
	case BackoffTypeLinear:
		delay = time.Duration(attempt) * rp.baseDelay
	case BackoffTypeExponential:
		delay = time.Duration(float64(rp.baseDelay) * math.Pow(rp.multiplier, float64(attempt-1)))
	default:
		delay = rp.baseDelay
	}

	// 限制最大延迟
	if delay > rp.maxDelay {
		delay = rp.maxDelay
	}

	// 添加抖动
	if rp.jitter {
		delay = rp.addJitter(delay)
	}

	return delay
}

// addJitter 添加抖动
func (rp *RetryPolicy) addJitter(delay time.Duration) time.Duration {
	// 添加 ±20% 的随机抖动
	jitterRange := float64(delay) * 0.2
	jitter := time.Duration(jitterRange * (2*randomFloat() - 1))
	return delay + jitter
}

// randomFloat 生成0-1之间的随机浮点数
func randomFloat() float64 {
	// 简单的伪随机数生成
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// ShouldRetry 判断是否应该重试
func (rp *RetryPolicy) ShouldRetry(attempt int, err error) bool {
	if attempt >= rp.maxRetries {
		return false
	}

	// 可以根据错误类型判断是否应该重试
	return !isNonRetryableError(err)
}

// isNonRetryableError 判断是否为不可重试的错误
func isNonRetryableError(err error) bool {
	if err == nil {
		return true
	}

	// 这里可以根据具体的错误类型判断
	// 例如：配置错误、认证错误等不应该重试
	// 暂时返回false，表示所有错误都可以重试
	return false
}
