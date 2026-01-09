package health

import (
	"context"
	"testing"
	"time"
)

func TestHealthChecker(t *testing.T) {
	// 创建健康检查器（不连接实际服务）
	checker := NewHealthChecker(nil, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 测试检查所有服务
	results := checker.CheckAll(ctx)

	// 应该有三个服务的检查结果
	expectedServices := []string{"postgresql", "redis", "influxdb"}
	for _, service := range expectedServices {
		if result, exists := results[service]; !exists {
			t.Errorf("Missing health check result for service: %s", service)
		} else {
			// 由于没有实际连接，所有服务都应该是不健康的
			if result.Status != StatusUnhealthy {
				t.Errorf("Expected service %s to be unhealthy, got %s", service, result.Status)
			}
		}
	}

	// 测试整体状态
	overallStatus, _ := checker.GetOverallStatus(ctx)
	if overallStatus != StatusUnhealthy {
		t.Errorf("Expected overall status to be unhealthy, got %s", overallStatus)
	}

	// 测试IsHealthy
	if checker.IsHealthy(ctx) {
		t.Error("Expected IsHealthy to return false")
	}
}

func TestCheckResult(t *testing.T) {
	checker := NewHealthChecker(nil, nil)

	// 测试PostgreSQL检查
	result := checker.CheckPostgreSQL()
	if result.Service != "postgresql" {
		t.Errorf("Expected service name 'postgresql', got '%s'", result.Service)
	}
	if result.Status != StatusUnhealthy {
		t.Errorf("Expected status unhealthy, got %s", result.Status)
	}
	if result.Duration <= 0 {
		t.Error("Expected positive duration")
	}

	// 测试Redis检查
	result = checker.CheckRedis()
	if result.Service != "redis" {
		t.Errorf("Expected service name 'redis', got '%s'", result.Service)
	}

	// 测试InfluxDB检查
	result = checker.CheckInfluxDB()
	if result.Service != "influxdb" {
		t.Errorf("Expected service name 'influxdb', got '%s'", result.Service)
	}
}