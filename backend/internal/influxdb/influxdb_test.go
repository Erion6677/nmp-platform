package influxdb

import (
	"nmp-platform/internal/config"
	"testing"
	"time"
)

func TestInfluxDBConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping InfluxDB integration test in short mode")
	}

	cfg := &config.InfluxConfig{
		URL:    "http://localhost:8086",
		Token:  "",
		Org:    "nmp",
		Bucket: "test",
	}

	client, err := Connect(cfg)
	if err != nil {
		t.Skipf("Cannot connect to InfluxDB: %v", err)
	}
	defer client.Close()

	// 测试健康检查
	if err := client.Health(); err != nil {
		t.Errorf("InfluxDB health check failed: %v", err)
	}

	// 测试写入数据点
	tags := map[string]string{
		"device": "test-device",
		"type":   "cpu",
	}
	fields := map[string]interface{}{
		"usage": 75.5,
	}

	err = client.WritePoint("system_metrics", tags, fields, time.Now())
	if err != nil {
		t.Errorf("Failed to write point: %v", err)
	}

	// 强制刷新
	client.Flush()
}

func TestInfluxDBHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping InfluxDB integration test in short mode")
	}

	cfg := &config.InfluxConfig{
		URL:    "http://localhost:8086",
		Token:  "",
		Org:    "nmp",
		Bucket: "test",
	}

	client, err := Connect(cfg)
	if err != nil {
		t.Skipf("Cannot connect to InfluxDB: %v", err)
	}
	defer client.Close()

	if err := client.Health(); err != nil {
		t.Errorf("InfluxDB should be healthy: %v", err)
	}
}