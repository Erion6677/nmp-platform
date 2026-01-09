package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// MockHealthChecker 模拟健康检查器
type MockHealthChecker struct {
	mock.Mock
}

func (m *MockHealthChecker) CheckHealth(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestRecoveryService(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockHealthChecker := &MockHealthChecker{}

	t.Run("创建恢复服务", func(t *testing.T) {
		rs := NewRecoveryService(logger, mockHealthChecker)
		
		assert.NotNil(t, rs)
		assert.Equal(t, logger, rs.logger)
		assert.Equal(t, mockHealthChecker, rs.healthChecker)
		assert.Equal(t, 3, rs.retryAttempts)
		assert.Equal(t, time.Second*5, rs.retryInterval)
		assert.NotNil(t, rs.circuitBreaker)
		assert.False(t, rs.isRecovering)
	})

	t.Run("成功恢复", func(t *testing.T) {
		rs := NewRecoveryService(logger, mockHealthChecker)
		
		recoveryFunc := func() error {
			return nil
		}
		
		ctx := context.Background()
		err := rs.AttemptRecovery(ctx, "test-service", recoveryFunc)
		
		assert.NoError(t, err)
		assert.False(t, rs.isRecovering)
	})

	t.Run("恢复失败后重试", func(t *testing.T) {
		rs := NewRecoveryService(logger, mockHealthChecker)
		rs.retryInterval = time.Millisecond * 10 // 加快测试速度
		
		callCount := 0
		recoveryFunc := func() error {
			callCount++
			if callCount < 2 {
				return errors.New("恢复失败")
			}
			return nil
		}
		
		ctx := context.Background()
		err := rs.AttemptRecovery(ctx, "test-service", recoveryFunc)
		
		assert.NoError(t, err)
		assert.Equal(t, 2, callCount)
	})

	t.Run("所有重试都失败", func(t *testing.T) {
		rs := NewRecoveryService(logger, mockHealthChecker)
		rs.retryInterval = time.Millisecond * 10
		
		recoveryFunc := func() error {
			return errors.New("恢复失败")
		}
		
		ctx := context.Background()
		err := rs.AttemptRecovery(ctx, "test-service", recoveryFunc)
		
		assert.Error(t, err)
		var appErr *AppError
		if assert.True(t, errors.As(err, &appErr)) {
			assert.Equal(t, ErrCodeServiceUnavailable, appErr.Code)
		}
	})

	t.Run("并发恢复保护", func(t *testing.T) {
		rs := NewRecoveryService(logger, mockHealthChecker)
		
		recoveryFunc := func() error {
			time.Sleep(time.Millisecond * 50)
			return nil
		}
		
		ctx := context.Background()
		
		// 启动第一个恢复
		go func() {
			rs.AttemptRecovery(ctx, "test-service", recoveryFunc)
		}()
		
		// 稍等一下确保第一个恢复已经开始
		time.Sleep(time.Millisecond * 10)
		
		// 尝试第二个恢复
		err := rs.AttemptRecovery(ctx, "test-service", recoveryFunc)
		
		assert.Error(t, err)
		var appErr *AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, ErrCodeServiceUnavailable, appErr.Code)
		assert.Contains(t, appErr.Message, "Recovery already in progress")
	})

	t.Run("上下文取消", func(t *testing.T) {
		rs := NewRecoveryService(logger, mockHealthChecker)
		rs.retryInterval = time.Second * 1
		
		callCount := 0
		recoveryFunc := func() error {
			callCount++
			return errors.New("恢复失败")
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
		defer cancel()
		
		err := rs.AttemptRecovery(ctx, "test-service", recoveryFunc)
		
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		assert.Equal(t, 1, callCount) // 只调用一次就被取消了
	})

	t.Run("健康检查", func(t *testing.T) {
		rs := NewRecoveryService(logger, mockHealthChecker)
		ctx := context.Background()
		
		// 健康的情况
		mockHealthChecker.On("CheckHealth", ctx).Return(nil).Once()
		healthy := rs.IsServiceHealthy(ctx)
		assert.True(t, healthy)
		
		// 不健康的情况
		mockHealthChecker.On("CheckHealth", ctx).Return(errors.New("服务不健康")).Once()
		healthy = rs.IsServiceHealthy(ctx)
		assert.False(t, healthy)
		
		mockHealthChecker.AssertExpectations(t)
	})
}

func TestCircuitBreaker(t *testing.T) {
	t.Run("正常执行", func(t *testing.T) {
		cb := &CircuitBreaker{
			maxFailures:  3,
			resetTimeout: time.Minute,
			state:        CircuitClosed,
		}
		
		err := cb.Call(func() error {
			return nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, CircuitClosed, cb.GetState())
		assert.Equal(t, 0, cb.failures)
	})

	t.Run("失败计数", func(t *testing.T) {
		cb := &CircuitBreaker{
			maxFailures:  3,
			resetTimeout: time.Minute,
			state:        CircuitClosed,
		}
		
		// 第一次失败
		err := cb.Call(func() error {
			return errors.New("失败")
		})
		
		assert.Error(t, err)
		assert.Equal(t, CircuitClosed, cb.GetState())
		assert.Equal(t, 1, cb.failures)
		
		// 第二次失败
		err = cb.Call(func() error {
			return errors.New("失败")
		})
		
		assert.Error(t, err)
		assert.Equal(t, CircuitClosed, cb.GetState())
		assert.Equal(t, 2, cb.failures)
		
		// 第三次失败，熔断器打开
		err = cb.Call(func() error {
			return errors.New("失败")
		})
		
		assert.Error(t, err)
		assert.Equal(t, CircuitOpen, cb.GetState())
		assert.Equal(t, 3, cb.failures)
	})

	t.Run("熔断器打开状态", func(t *testing.T) {
		cb := &CircuitBreaker{
			maxFailures:  2,
			resetTimeout: time.Millisecond * 100,
			state:        CircuitOpen,
			failures:     2,
			lastFailTime: time.Now(),
		}
		
		// 在重置时间内，应该直接返回错误
		err := cb.Call(func() error {
			return nil
		})
		
		assert.Error(t, err)
		var appErr *AppError
		assert.True(t, errors.As(err, &appErr))
		assert.Equal(t, ErrCodeServiceUnavailable, appErr.Code)
	})

	t.Run("熔断器半开状态", func(t *testing.T) {
		cb := &CircuitBreaker{
			maxFailures:  2,
			resetTimeout: time.Millisecond * 10,
			state:        CircuitOpen,
			failures:     2,
			lastFailTime: time.Now().Add(-time.Millisecond * 20), // 超过重置时间
		}
		
		// 第一次调用应该进入半开状态并成功
		err := cb.Call(func() error {
			return nil
		})
		
		assert.NoError(t, err)
		assert.Equal(t, CircuitClosed, cb.GetState())
		assert.Equal(t, 0, cb.failures)
	})

	t.Run("半开状态失败", func(t *testing.T) {
		cb := &CircuitBreaker{
			maxFailures:  2,
			resetTimeout: time.Millisecond * 10,
			state:        CircuitOpen,
			failures:     2,
			lastFailTime: time.Now().Add(-time.Millisecond * 20),
		}
		
		// 半开状态下失败，应该重新打开
		err := cb.Call(func() error {
			return errors.New("失败")
		})
		
		assert.Error(t, err)
		assert.Equal(t, CircuitOpen, cb.GetState())
		assert.Equal(t, 1, cb.failures) // 半开状态重置了failures，然后失败+1
	})

	t.Run("重置熔断器", func(t *testing.T) {
		cb := &CircuitBreaker{
			maxFailures:  2,
			resetTimeout: time.Minute,
			state:        CircuitOpen,
			failures:     3,
		}
		
		cb.Reset()
		
		assert.Equal(t, CircuitClosed, cb.GetState())
		assert.Equal(t, 0, cb.failures)
	})
}

func TestRecoveryFunctions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockHealthChecker := &MockHealthChecker{}
	rs := NewRecoveryService(logger, mockHealthChecker)

	t.Run("数据库恢复", func(t *testing.T) {
		err := rs.DatabaseRecovery()
		assert.NoError(t, err)
	})

	t.Run("Redis恢复", func(t *testing.T) {
		err := rs.RedisRecovery()
		assert.NoError(t, err)
	})

	t.Run("InfluxDB恢复", func(t *testing.T) {
		err := rs.InfluxDBRecovery()
		assert.NoError(t, err)
	})

	t.Run("恢复所有服务", func(t *testing.T) {
		ctx := context.Background()
		err := rs.RecoverAllServices(ctx)
		assert.NoError(t, err)
	})
}

// 基准测试
func BenchmarkCircuitBreaker(b *testing.B) {
	cb := &CircuitBreaker{
		maxFailures:  5,
		resetTimeout: time.Minute,
		state:        CircuitClosed,
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cb.Call(func() error {
			return nil
		})
	}
}

func BenchmarkRecoveryService(b *testing.B) {
	logger := zaptest.NewLogger(b)
	mockHealthChecker := &MockHealthChecker{}
	rs := NewRecoveryService(logger, mockHealthChecker)
	
	recoveryFunc := func() error {
		return nil
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rs.AttemptRecovery(ctx, "test-service", recoveryFunc)
	}
}