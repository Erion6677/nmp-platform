package redis

import (
	"context"
	"nmp-platform/internal/config"
	"testing"
	"time"
)

func TestRedisConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	client, err := Connect(cfg)
	if err != nil {
		t.Skipf("Cannot connect to Redis: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试基本操作
	key := "test:key"
	value := "test-value"

	// 设置值
	err = client.Set(ctx, key, value, time.Minute)
	if err != nil {
		t.Errorf("Failed to set value: %v", err)
	}

	// 获取值
	retrieved, err := client.Get(ctx, key)
	if err != nil {
		t.Errorf("Failed to get value: %v", err)
	}

	if retrieved != value {
		t.Errorf("Expected %s, got %s", value, retrieved)
	}

	// 删除键
	err = client.Delete(ctx, key)
	if err != nil {
		t.Errorf("Failed to delete key: %v", err)
	}
}

func TestRedisJSONOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	client, err := Connect(cfg)
	if err != nil {
		t.Skipf("Cannot connect to Redis: %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	// 测试JSON操作
	type TestData struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	key := "test:json"
	data := TestData{Name: "test", Value: 42}

	// 设置JSON值
	err = client.SetJSON(ctx, key, data, time.Minute)
	if err != nil {
		t.Errorf("Failed to set JSON value: %v", err)
	}

	// 获取JSON值
	var retrieved TestData
	err = client.GetJSON(ctx, key, &retrieved)
	if err != nil {
		t.Errorf("Failed to get JSON value: %v", err)
	}

	if retrieved.Name != data.Name || retrieved.Value != data.Value {
		t.Errorf("JSON data mismatch: expected %+v, got %+v", data, retrieved)
	}

	// 清理
	client.Delete(ctx, key)
}

func TestRedisHealth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	cfg := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}

	client, err := Connect(cfg)
	if err != nil {
		t.Skipf("Cannot connect to Redis: %v", err)
	}
	defer client.Close()

	if err := client.Health(); err != nil {
		t.Errorf("Redis should be healthy: %v", err)
	}
}