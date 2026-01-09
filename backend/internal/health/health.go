package health

import (
	"context"
	"fmt"
	"nmp-platform/internal/database"
	"nmp-platform/internal/influxdb"
	"nmp-platform/internal/redis"
	"time"
)

// Status 健康检查状态
type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusUnhealthy Status = "unhealthy"
	StatusDegraded  Status = "degraded"
)

// CheckResult 健康检查结果
type CheckResult struct {
	Service   string        `json:"service"`
	Status    Status        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthChecker 健康检查器
type HealthChecker struct {
	redisClient    *redis.Client
	influxdbClient *influxdb.Client
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker(redisClient *redis.Client, influxdbClient *influxdb.Client) *HealthChecker {
	return &HealthChecker{
		redisClient:    redisClient,
		influxdbClient: influxdbClient,
	}
}

// CheckAll 检查所有服务的健康状态
func (h *HealthChecker) CheckAll(ctx context.Context) map[string]CheckResult {
	results := make(map[string]CheckResult)

	// 并发检查所有服务
	checks := []func() CheckResult{
		h.CheckPostgreSQL,
		h.CheckRedis,
		h.CheckInfluxDB,
	}

	resultChan := make(chan CheckResult, len(checks))

	for _, check := range checks {
		go func(checkFunc func() CheckResult) {
			resultChan <- checkFunc()
		}(check)
	}

	// 收集结果
	for i := 0; i < len(checks); i++ {
		result := <-resultChan
		results[result.Service] = result
	}

	return results
}

// CheckPostgreSQL 检查PostgreSQL连接
func (h *HealthChecker) CheckPostgreSQL() CheckResult {
	start := time.Now()
	result := CheckResult{
		Service:   "postgresql",
		Timestamp: start,
	}

	if database.DB == nil {
		result.Status = StatusUnhealthy
		result.Message = "Database connection not initialized"
		result.Duration = time.Since(start)
		return result
	}

	sqlDB, err := database.DB.DB()
	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Failed to get underlying sql.DB: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Database ping failed: %v", err)
	} else {
		result.Status = StatusHealthy
		result.Message = "Database connection is healthy"
	}

	result.Duration = time.Since(start)
	return result
}

// CheckRedis 检查Redis连接
func (h *HealthChecker) CheckRedis() CheckResult {
	start := time.Now()
	result := CheckResult{
		Service:   "redis",
		Timestamp: start,
	}

	if h.redisClient == nil {
		result.Status = StatusUnhealthy
		result.Message = "Redis client not initialized"
		result.Duration = time.Since(start)
		return result
	}

	if err := h.redisClient.Health(); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("Redis health check failed: %v", err)
	} else {
		result.Status = StatusHealthy
		result.Message = "Redis connection is healthy"
	}

	result.Duration = time.Since(start)
	return result
}

// CheckInfluxDB 检查InfluxDB连接
func (h *HealthChecker) CheckInfluxDB() CheckResult {
	start := time.Now()
	result := CheckResult{
		Service:   "influxdb",
		Timestamp: start,
	}

	if h.influxdbClient == nil {
		result.Status = StatusUnhealthy
		result.Message = "InfluxDB client not initialized"
		result.Duration = time.Since(start)
		return result
	}

	if err := h.influxdbClient.Health(); err != nil {
		result.Status = StatusUnhealthy
		result.Message = fmt.Sprintf("InfluxDB health check failed: %v", err)
	} else {
		result.Status = StatusHealthy
		result.Message = "InfluxDB connection is healthy"
	}

	result.Duration = time.Since(start)
	return result
}

// GetOverallStatus 获取整体健康状态
func (h *HealthChecker) GetOverallStatus(ctx context.Context) (Status, map[string]CheckResult) {
	results := h.CheckAll(ctx)
	
	overallStatus := StatusHealthy
	unhealthyCount := 0
	
	for _, result := range results {
		if result.Status == StatusUnhealthy {
			unhealthyCount++
		}
	}

	// 如果有服务不健康，根据数量确定整体状态
	if unhealthyCount > 0 {
		if unhealthyCount == len(results) {
			overallStatus = StatusUnhealthy
		} else {
			overallStatus = StatusDegraded
		}
	}

	return overallStatus, results
}

// IsHealthy 检查是否所有服务都健康
func (h *HealthChecker) IsHealthy(ctx context.Context) bool {
	status, _ := h.GetOverallStatus(ctx)
	return status == StatusHealthy
}