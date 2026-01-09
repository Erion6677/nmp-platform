package service

import (
	"context"
	"nmp-platform/internal/models"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestDataCompressionService 测试数据压缩服务
func TestDataCompressionService(t *testing.T) {
	t.Run("compress historical data", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		
		service := NewDataCompressionService(mockInflux, mockRedis)
		
		// 测试压缩7天前的数据
		err := service.CompressHistoricalData(context.Background(), 7*24*time.Hour)
		assert.NoError(t, err, "Should compress data without error")
	})
	
	t.Run("cleanup old data", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		
		// 设置Redis模拟行为
		mockRedis.On("Keys", mock.Anything, mock.AnythingOfType("string")).Return([]string{
			"device:latest:1",
			"device:metric:1:cpu",
			"device:last_seen:1",
		}, nil)
		mockRedis.On("Delete", mock.Anything, mock.Anything).Return(nil)
		
		service := NewDataCompressionService(mockInflux, mockRedis)
		
		// 测试清理90天前的数据
		err := service.CleanupOldData(context.Background(), 90*24*time.Hour)
		assert.NoError(t, err, "Should cleanup data without error")
		
		// 验证Redis清理被调用
		mockRedis.AssertCalled(t, "Keys", mock.Anything, "device:latest:*")
	})
	
	t.Run("get data statistics", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		
		// 设置Redis模拟行为
		mockRedis.On("Keys", mock.Anything, "device:*").Return([]string{
			"device:latest:1",
			"device:latest:2",
			"device:metric:1:cpu",
		}, nil)
		
		service := NewDataCompressionService(mockInflux, mockRedis)
		
		stats, err := service.GetDataStatistics(context.Background())
		assert.NoError(t, err, "Should get statistics without error")
		assert.NotNil(t, stats, "Statistics should not be nil")
		assert.Equal(t, 3, stats["redis_keys_count"], "Should count Redis keys correctly")
		assert.Contains(t, stats, "last_updated", "Should include last_updated timestamp")
	})
	
	t.Run("optimize storage", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		
		// 设置Redis模拟行为
		mockRedis.On("Keys", mock.Anything, mock.AnythingOfType("string")).Return([]string{
			"device:latest:1",
			"device:latest:2",
		}, nil)
		mockRedis.On("Get", mock.Anything, mock.AnythingOfType("string")).Return("valid_data", nil)
		
		service := NewDataCompressionService(mockInflux, mockRedis)
		
		err := service.OptimizeStorage(context.Background())
		assert.NoError(t, err, "Should optimize storage without error")
	})
}

// TestDataStorageIntegration 测试数据存储集成
func TestDataStorageIntegration(t *testing.T) {
	t.Run("data receiver and compression integration", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}
		
		// 设置模拟行为
		device := &models.Device{
			ID:     1,
			Name:   "TestDevice-1",
			Host:   "192.168.1.1",
			Type:   models.DeviceTypeRouter,
			Status: models.DeviceStatusOnline,
		}
		mockDeviceRepo.On("GetByID", uint(1)).Return(device, nil)
		mockRedis.On("Exists", mock.Anything, mock.AnythingOfType("string")).Return(false, nil)
		mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
		mockRedis.On("SetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
		mockInflux.On("WritePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		
		// 创建服务
		receiverService := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)
		compressionService := NewDataCompressionService(mockInflux, mockRedis)
		
		// 测试数据接收
		data := &models.MetricData{
			DeviceID:  "1",
			Timestamp: time.Now(),
			Metrics: map[string]interface{}{
				"cpu_usage":    75.5,
				"memory_usage": 60.2,
			},
			Tags: map[string]string{
				"source": "test",
			},
		}
		
		err := receiverService.ReceiveData(context.Background(), data)
		assert.NoError(t, err, "Should receive data without error")
		
		// 验证数据被写入
		mockInflux.AssertCalled(t, "WritePoint", "device_metrics", mock.Anything, mock.Anything, mock.Anything)
		
		// 测试数据压缩
		err = compressionService.CompressHistoricalData(context.Background(), 24*time.Hour)
		assert.NoError(t, err, "Should compress data without error")
	})
}