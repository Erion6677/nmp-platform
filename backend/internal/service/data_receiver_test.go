package service

import (
	"context"
	"errors"
	"fmt"
	"nmp-platform/internal/models"
	"strconv"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockInfluxClient InfluxDB客户端模拟
type MockInfluxClient struct {
	mock.Mock
}

func (m *MockInfluxClient) WritePoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) error {
	args := m.Called(measurement, tags, fields, timestamp)
	return args.Error(0)
}

func (m *MockInfluxClient) Health() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockInfluxClient) Close() {
	m.Called()
}

// MockRedisClient Redis客户端模拟
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	args := m.Called(ctx, key, value, expiration)
	return args.Error(0)
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockRedisClient) Delete(ctx context.Context, keys ...string) error {
	args := m.Called(ctx, keys)
	return args.Error(0)
}

func (m *MockRedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	args := m.Called(ctx, pattern)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRedisClient) Health() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockDeviceRepository 设备仓库模拟
type MockDeviceRepository struct {
	mock.Mock
}

func (m *MockDeviceRepository) GetByID(id uint) (*models.Device, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) Create(device *models.Device) error {
	args := m.Called(device)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetByHost(host string) (*models.Device, error) {
	args := m.Called(host)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) Update(device *models.Device) error {
	args := m.Called(device)
	return args.Error(0)
}

func (m *MockDeviceRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeviceRepository) List(offset, limit int, filters map[string]interface{}) ([]*models.Device, int64, error) {
	args := m.Called(offset, limit, filters)
	return args.Get(0).([]*models.Device), args.Get(1).(int64), args.Error(2)
}

func (m *MockDeviceRepository) UpdateStatus(id uint, status models.DeviceStatus) error {
	args := m.Called(id, status)
	return args.Error(0)
}

func (m *MockDeviceRepository) UpdateLastSeen(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetByGroupID(groupID uint) ([]*models.Device, error) {
	args := m.Called(groupID)
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByTagID(tagID uint) ([]*models.Device, error) {
	args := m.Called(tagID)
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) AddToGroup(deviceID, groupID uint) error {
	args := m.Called(deviceID, groupID)
	return args.Error(0)
}

func (m *MockDeviceRepository) RemoveFromGroup(deviceID, groupID uint) error {
	args := m.Called(deviceID, groupID)
	return args.Error(0)
}

func (m *MockDeviceRepository) AddTag(deviceID, tagID uint) error {
	args := m.Called(deviceID, tagID)
	return args.Error(0)
}

func (m *MockDeviceRepository) RemoveTag(deviceID, tagID uint) error {
	args := m.Called(deviceID, tagID)
	return args.Error(0)
}

func (m *MockDeviceRepository) GetAllOnline() ([]*models.Device, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) GetByOSType(osType models.DeviceOSType) ([]*models.Device, error) {
	args := m.Called(osType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Device), args.Error(1)
}

func (m *MockDeviceRepository) UpdateConnectionInfo(id uint, apiPort, sshPort int) error {
	args := m.Called(id, apiPort, sshPort)
	return args.Error(0)
}

// TestDataPushProcessingConsistency 测试数据推送处理一致性属性
// Feature: network-monitoring-platform, Property 4: 数据推送处理一致性
// **验证需求: 3.1, 3.2, 3.3, 3.4, 3.5**
func TestDataPushProcessingConsistency(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	// 属性1: 有效数据格式和有效设备ID应该成功处理
	properties.Property("valid data with valid device ID should be processed successfully", 
		prop.ForAll(
			func(deviceID uint, metricName string, metricValue float64) bool {
				// 创建模拟对象
				mockInflux := &MockInfluxClient{}
				mockRedis := &MockRedisClient{}
				mockDeviceRepo := &MockDeviceRepository{}

				// 设置模拟行为 - 设备存在
				device := &models.Device{
					ID:   deviceID,
					Name: fmt.Sprintf("Device-%d", deviceID),
					Host: fmt.Sprintf("192.168.1.%d", deviceID%255),
				}
				mockDeviceRepo.On("GetByID", deviceID).Return(device, nil)

				// 设置Redis缓存行为
				mockRedis.On("Exists", mock.Anything, mock.AnythingOfType("string")).Return(false, nil)
				mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
				mockRedis.On("SetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)

				// 设置InfluxDB写入行为
				mockInflux.On("WritePoint", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

				// 创建服务
				service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

				// 创建有效的监控数据
				data := &models.MetricData{
					DeviceID:  strconv.FormatUint(uint64(deviceID), 10),
					Timestamp: time.Now(),
					Metrics: map[string]interface{}{
						metricName: metricValue,
					},
					Tags: map[string]string{
						"source": "test",
					},
				}

				// 处理数据
				err := service.ReceiveData(context.Background(), data)

				// 验证结果
				return err == nil
			},
			gen.UIntRange(1, 1000),                                    // 设备ID范围
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }), // 指标名称
			gen.Float64Range(-1000, 1000),                             // 指标值范围
		))

	// 属性2: 无效数据格式应该返回验证错误
	properties.Property("invalid data format should return validation error",
		prop.ForAll(
			func(deviceID string) bool {
				// 创建模拟对象
				mockInflux := &MockInfluxClient{}
				mockRedis := &MockRedisClient{}
				mockDeviceRepo := &MockDeviceRepository{}

				// 创建服务
				service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

				// 创建无效的监控数据（空的metrics）
				data := &models.MetricData{
					DeviceID:  deviceID,
					Timestamp: time.Now(),
					Metrics:   map[string]interface{}{}, // 空的metrics应该导致验证失败
					Tags:      map[string]string{},
				}

				// 处理数据
				err := service.ReceiveData(context.Background(), data)

				// 验证返回验证错误（应该因为空metrics而失败，不管设备ID是什么）
				return err != nil // 只要有错误就算通过，因为空metrics应该导致验证失败
			},
			gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		))

	properties.TestingRun(t)
}

// 单元测试：验证具体的边界情况
func TestDataReceiverEdgeCases(t *testing.T) {
	t.Run("empty device ID should fail validation", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}
		
		service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)
		
		data := &models.MetricData{
			DeviceID:  "", // 空设备ID
			Timestamp: time.Now(),
			Metrics: map[string]interface{}{
				"cpu_usage": 50.0,
			},
		}
		
		err := service.ReceiveData(context.Background(), data)
		assert.Error(t, err)
		
		// 检查是否是验证错误（可能是ValidationError或其他类型的错误）
		assert.Contains(t, err.Error(), "device_id", "Error should mention device_id")
	})
	
	t.Run("nil metrics should fail validation", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}
		
		service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)
		
		data := &models.MetricData{
			DeviceID:  "123",
			Timestamp: time.Now(),
			Metrics:   nil, // nil metrics
		}
		
		err := service.ReceiveData(context.Background(), data)
		assert.Error(t, err)
		
		// 检查是否是验证错误（可能是ValidationError或其他类型的错误）
		assert.Contains(t, err.Error(), "metrics", "Error should mention metrics")
	})
}


// MockCollectorRepository 采集器仓库模拟
type MockCollectorRepository struct {
	mock.Mock
}

func (m *MockCollectorRepository) Create(collector *models.CollectorScript) error {
	args := m.Called(collector)
	return args.Error(0)
}

func (m *MockCollectorRepository) GetByID(id uint) (*models.CollectorScript, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CollectorScript), args.Error(1)
}

func (m *MockCollectorRepository) GetByDeviceID(deviceID uint) (*models.CollectorScript, error) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CollectorScript), args.Error(1)
}

func (m *MockCollectorRepository) Update(collector *models.CollectorScript) error {
	args := m.Called(collector)
	return args.Error(0)
}

func (m *MockCollectorRepository) Delete(id uint) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockCollectorRepository) DeleteByDeviceID(deviceID uint) error {
	args := m.Called(deviceID)
	return args.Error(0)
}

func (m *MockCollectorRepository) UpdateStatus(deviceID uint, status models.CollectorStatus, errorMsg string) error {
	args := m.Called(deviceID, status, errorMsg)
	return args.Error(0)
}

func (m *MockCollectorRepository) UpdateDeployedAt(deviceID uint) error {
	args := m.Called(deviceID)
	return args.Error(0)
}

func (m *MockCollectorRepository) UpdateLastPushAt(deviceID uint) error {
	args := m.Called(deviceID)
	return args.Error(0)
}

func (m *MockCollectorRepository) IncrementPushCount(deviceID uint) error {
	args := m.Called(deviceID)
	return args.Error(0)
}

func (m *MockCollectorRepository) UpdateEnabled(deviceID uint, enabled bool) error {
	args := m.Called(deviceID, enabled)
	return args.Error(0)
}

func (m *MockCollectorRepository) UpdateInterval(deviceID uint, intervalMs int) error {
	args := m.Called(deviceID, intervalMs)
	return args.Error(0)
}

func (m *MockCollectorRepository) UpdateConfig(deviceID uint, intervalMs, pushBatchSize int) error {
	args := m.Called(deviceID, intervalMs, pushBatchSize)
	return args.Error(0)
}

func (m *MockCollectorRepository) GetAllEnabled() ([]*models.CollectorScript, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CollectorScript), args.Error(1)
}

func (m *MockCollectorRepository) GetByStatus(status models.CollectorStatus) ([]*models.CollectorScript, error) {
	args := m.Called(status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.CollectorScript), args.Error(1)
}

func (m *MockCollectorRepository) GetOrCreate(deviceID uint) (*models.CollectorScript, error) {
	args := m.Called(deviceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.CollectorScript), args.Error(1)
}

// ========== Property 3: 推送数据处理正确性 ==========
// Feature: device-monitoring, Property 3: 推送数据处理正确性
// *For any* 推送数据请求，如果设备标识无效，系统应拒绝请求；
// 如果设备标识有效，数据应被写入 InfluxDB 且 Redis 中的设备状态应被更新为在线。
// **Validates: Requirements 5.2, 5.3, 5.4**

func TestProperty3_PushDataProcessingCorrectness(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	// 属性3.1: 无效设备标识应该被拒绝
	properties.Property("invalid device ID should be rejected",
		prop.ForAll(
			func(invalidIP string) bool {
				mockInflux := &MockInfluxClient{}
				mockRedis := &MockRedisClient{}
				mockDeviceRepo := &MockDeviceRepository{}

				// 设置模拟行为 - 设备不存在
				mockRedis.On("GetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(errors.New("not found"))
				mockDeviceRepo.On("GetByHost", invalidIP).Return(nil, errors.New("device not found"))

				service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

				// 验证设备身份
				_, err := service.ValidateDeviceByIP(context.Background(), invalidIP)

				// 应该返回错误
				return err != nil
			},
			gen.Identifier(), // 使用 Identifier 生成器，生成有效的标识符字符串
		))

	// 属性3.2: 有效设备标识应该成功验证并更新状态
	properties.Property("valid device ID should be validated and status updated",
		prop.ForAll(
			func(deviceID uint) bool {
				mockInflux := &MockInfluxClient{}
				mockRedis := &MockRedisClient{}
				mockDeviceRepo := &MockDeviceRepository{}
				mockCollectorRepo := &MockCollectorRepository{}

				// 构造有效的 IP
				validIP := fmt.Sprintf("192.168.1.%d", deviceID%255)

				// 设置模拟行为 - 设备存在
				device := &models.Device{
					ID:   deviceID,
					Name: fmt.Sprintf("Device-%d", deviceID),
					Host: validIP,
				}
				mockRedis.On("GetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything).Return(errors.New("not found"))
				mockRedis.On("SetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
				mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
				mockDeviceRepo.On("GetByHost", validIP).Return(device, nil)
				mockDeviceRepo.On("UpdateStatus", deviceID, models.DeviceStatusOnline).Return(nil)
				mockCollectorRepo.On("UpdateLastPushAt", deviceID).Return(nil)
				mockCollectorRepo.On("IncrementPushCount", deviceID).Return(nil)

				service := NewDataReceiverServiceWithCollector(mockInflux, mockRedis, mockDeviceRepo, mockCollectorRepo)

				// 验证设备身份
				validatedDevice, err := service.ValidateDeviceByIP(context.Background(), validIP)
				if err != nil {
					return false
				}

				// 更新设备状态
				err = service.UpdateDeviceOnlineStatus(context.Background(), validatedDevice.ID)

				// 应该成功
				return err == nil
			},
			gen.UIntRange(1, 255),
		))

	// 属性3.3: 带宽数据应该被正确写入 InfluxDB
	properties.Property("bandwidth data should be written to InfluxDB",
		prop.ForAll(
			func(deviceID uint, rxRate, txRate int64) bool {
				mockInflux := &MockInfluxClient{}
				mockRedis := &MockRedisClient{}
				mockDeviceRepo := &MockDeviceRepository{}

				// 设置模拟行为
				mockInflux.On("WritePoint", "bandwidth", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockRedis.On("SetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)

				service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

				// 处理带宽数据
				interfaces := map[string]models.InterfaceMetrics{
					"ether1": {RxRate: rxRate, TxRate: txRate},
				}
				err := service.ProcessBandwidthData(context.Background(), deviceID, time.Now().UnixMilli(), interfaces)

				// 验证 InfluxDB 写入被调用
				mockInflux.AssertCalled(t, "WritePoint", "bandwidth", mock.Anything, mock.Anything, mock.Anything)

				return err == nil
			},
			gen.UIntRange(1, 1000),
			gen.Int64Range(0, 1000000000),
			gen.Int64Range(0, 1000000000),
		))

	// 属性3.4: Ping 数据应该被正确写入 InfluxDB
	properties.Property("ping data should be written to InfluxDB",
		prop.ForAll(
			func(deviceID uint, latency int64) bool {
				mockInflux := &MockInfluxClient{}
				mockRedis := &MockRedisClient{}
				mockDeviceRepo := &MockDeviceRepository{}

				// 设置模拟行为
				mockInflux.On("WritePoint", "ping", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				mockRedis.On("SetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)

				service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

				// 处理 Ping 数据
				pings := []models.PingMetric{
					{Target: "8.8.8.8", Latency: latency, Status: "up"},
				}
				err := service.ProcessPingData(context.Background(), deviceID, time.Now().UnixMilli(), pings)

				// 验证 InfluxDB 写入被调用
				mockInflux.AssertCalled(t, "WritePoint", "ping", mock.Anything, mock.Anything, mock.Anything)

				return err == nil
			},
			gen.UIntRange(1, 1000),
			gen.Int64Range(0, 1000),
		))

	properties.TestingRun(t)
}

// ========== Property 4: 设备离线检测 ==========
// Feature: device-monitoring, Property 4: 设备离线检测
// *For any* 设备，如果超过配置的超时时间未收到推送数据，系统应将设备状态标记为离线。
// **Validates: Requirements 5.5**

func TestProperty4_DeviceOfflineDetection(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	// 属性4.1: 超过超时时间的设备应该被标记为离线
	properties.Property("device without recent data should be marked offline",
		prop.ForAll(
			func(deviceID uint, minutesSinceLastSeen int) bool {
				mockRedis := &MockRedisClient{}
				mockDeviceRepo := &MockDeviceRepository{}
				mockCollectorRepo := &MockCollectorRepository{}

				// 计算最后在线时间（超过10分钟）
				lastSeen := time.Now().Add(-time.Duration(minutesSinceLastSeen+11) * time.Minute)
				lastSeenStr := lastSeen.Format(time.RFC3339)

				// 设置模拟行为
				mockRedis.On("Get", mock.Anything, fmt.Sprintf("device:last_seen:%d", deviceID)).Return(lastSeenStr, nil)
				mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
				
				device := &models.Device{
					ID:       deviceID,
					Name:     fmt.Sprintf("Device-%d", deviceID),
					Status:   models.DeviceStatusOnline,
					LastSeen: &lastSeen,
				}
				mockDeviceRepo.On("GetByID", deviceID).Return(device, nil)
				mockDeviceRepo.On("List", 0, 10000, mock.Anything).Return([]*models.Device{device}, int64(1), nil)
				mockDeviceRepo.On("UpdateStatus", deviceID, models.DeviceStatusOffline).Return(nil)
				mockCollectorRepo.On("GetByDeviceID", deviceID).Return(nil, nil)

				checker := NewDeviceStatusChecker(mockDeviceRepo, mockCollectorRepo, mockRedis)
				checker.SetOfflineTimeout(10 * time.Minute)

				// 检查设备状态
				status, err := checker.CheckSingleDevice(context.Background(), deviceID)

				// 应该被标记为离线
				return err == nil && status == models.DeviceStatusOffline
			},
			gen.UIntRange(1, 1000),
			gen.IntRange(0, 60), // 额外的分钟数
		))

	// 属性4.2: 最近有数据的设备应该保持在线
	properties.Property("device with recent data should remain online",
		prop.ForAll(
			func(deviceID uint, minutesSinceLastSeen int) bool {
				mockRedis := &MockRedisClient{}
				mockDeviceRepo := &MockDeviceRepository{}
				mockCollectorRepo := &MockCollectorRepository{}

				// 计算最后在线时间（在10分钟内）
				actualMinutes := minutesSinceLastSeen % 9 // 确保在0-8分钟内
				lastSeen := time.Now().Add(-time.Duration(actualMinutes) * time.Minute)
				lastSeenStr := lastSeen.Format(time.RFC3339)

				// 设置模拟行为
				mockRedis.On("Get", mock.Anything, fmt.Sprintf("device:last_seen:%d", deviceID)).Return(lastSeenStr, nil)
				mockRedis.On("Set", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)
				
				device := &models.Device{
					ID:       deviceID,
					Name:     fmt.Sprintf("Device-%d", deviceID),
					Status:   models.DeviceStatusOnline,
					LastSeen: &lastSeen,
				}
				mockDeviceRepo.On("GetByID", deviceID).Return(device, nil)
				mockDeviceRepo.On("UpdateStatus", deviceID, models.DeviceStatusOnline).Return(nil)
				mockCollectorRepo.On("GetByDeviceID", deviceID).Return(nil, nil)

				checker := NewDeviceStatusChecker(mockDeviceRepo, mockCollectorRepo, mockRedis)
				checker.SetOfflineTimeout(10 * time.Minute)

				// 检查设备状态
				status, err := checker.CheckSingleDevice(context.Background(), deviceID)

				// 应该保持在线
				return err == nil && status == models.DeviceStatusOnline
			},
			gen.UIntRange(1, 1000),
			gen.IntRange(0, 100),
		))

	properties.TestingRun(t)
}

// 单元测试：带宽数据处理
func TestBandwidthDataProcessing(t *testing.T) {
	t.Run("should write bandwidth data to InfluxDB with correct tags", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}

		// 设置模拟行为
		mockInflux.On("WritePoint", "bandwidth", mock.MatchedBy(func(tags map[string]string) bool {
			return tags["device_id"] == "123" && tags["interface"] == "ether1"
		}), mock.MatchedBy(func(fields map[string]interface{}) bool {
			return fields["rx_rate"] == int64(1000000) && fields["tx_rate"] == int64(500000)
		}), mock.Anything).Return(nil)
		mockRedis.On("SetJSON", mock.Anything, "device:bandwidth:123", mock.Anything, mock.Anything).Return(nil)

		service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

		interfaces := map[string]models.InterfaceMetrics{
			"ether1": {RxRate: 1000000, TxRate: 500000},
		}
		err := service.ProcessBandwidthData(context.Background(), 123, time.Now().UnixMilli(), interfaces)

		assert.NoError(t, err)
		mockInflux.AssertExpectations(t)
	})

	t.Run("should handle multiple interfaces", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}

		// 设置模拟行为 - 允许任何调用
		mockInflux.On("WritePoint", "bandwidth", mock.Anything, mock.Anything, mock.Anything).Return(nil)
		mockRedis.On("SetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)

		service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

		interfaces := map[string]models.InterfaceMetrics{
			"ether1": {RxRate: 1000000, TxRate: 500000},
			"ether2": {RxRate: 2000000, TxRate: 1000000},
			"ether3": {RxRate: 3000000, TxRate: 1500000},
		}
		err := service.ProcessBandwidthData(context.Background(), 123, time.Now().UnixMilli(), interfaces)

		assert.NoError(t, err)
		// 验证 WritePoint 被调用了3次（每个接口一次）
		mockInflux.AssertNumberOfCalls(t, "WritePoint", 3)
	})
}

// 单元测试：Ping 数据处理
func TestPingDataProcessing(t *testing.T) {
	t.Run("should write ping data to InfluxDB with correct tags", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}

		// 设置模拟行为
		mockInflux.On("WritePoint", "ping", mock.MatchedBy(func(tags map[string]string) bool {
			return tags["device_id"] == "123" && tags["target_address"] == "8.8.8.8"
		}), mock.MatchedBy(func(fields map[string]interface{}) bool {
			return fields["latency"] == int64(10) && fields["status"] == "up"
		}), mock.Anything).Return(nil)
		mockRedis.On("SetJSON", mock.Anything, "device:ping:123", mock.Anything, mock.Anything).Return(nil)

		service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

		pings := []models.PingMetric{
			{Target: "8.8.8.8", Latency: 10, Status: "up"},
		}
		err := service.ProcessPingData(context.Background(), 123, time.Now().UnixMilli(), pings)

		assert.NoError(t, err)
		mockInflux.AssertExpectations(t)
	})

	t.Run("should mark packet loss when latency is 0", func(t *testing.T) {
		mockInflux := &MockInfluxClient{}
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}

		// 设置模拟行为
		mockInflux.On("WritePoint", "ping", mock.Anything, mock.MatchedBy(func(fields map[string]interface{}) bool {
			return fields["latency"] == int64(0) && fields["status"] == "down"
		}), mock.Anything).Return(nil)
		mockRedis.On("SetJSON", mock.Anything, mock.AnythingOfType("string"), mock.Anything, mock.Anything).Return(nil)

		service := NewDataReceiverService(mockInflux, mockRedis, mockDeviceRepo)

		pings := []models.PingMetric{
			{Target: "8.8.8.8", Latency: 0, Status: ""}, // 空状态，应该被设置为 down
		}
		err := service.ProcessPingData(context.Background(), 123, time.Now().UnixMilli(), pings)

		assert.NoError(t, err)
		mockInflux.AssertExpectations(t)
	})
}

// 单元测试：设备状态检查器
func TestDeviceStatusChecker(t *testing.T) {
	t.Run("should mark device offline when no recent data", func(t *testing.T) {
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}
		mockCollectorRepo := &MockCollectorRepository{}

		// 设置最后在线时间为20分钟前
		lastSeen := time.Now().Add(-20 * time.Minute)
		lastSeenStr := lastSeen.Format(time.RFC3339)

		device := &models.Device{
			ID:       1,
			Name:     "Test Device",
			Status:   models.DeviceStatusOnline,
			LastSeen: &lastSeen,
		}

		mockRedis.On("Get", mock.Anything, "device:last_seen:1").Return(lastSeenStr, nil)
		mockRedis.On("Set", mock.Anything, "device:status:1", "offline", mock.Anything).Return(nil)
		mockDeviceRepo.On("GetByID", uint(1)).Return(device, nil)
		mockDeviceRepo.On("UpdateStatus", uint(1), models.DeviceStatusOffline).Return(nil)
		mockCollectorRepo.On("GetByDeviceID", uint(1)).Return(nil, nil)

		checker := NewDeviceStatusChecker(mockDeviceRepo, mockCollectorRepo, mockRedis)
		checker.SetOfflineTimeout(10 * time.Minute)

		status, err := checker.CheckSingleDevice(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, models.DeviceStatusOffline, status)
	})

	t.Run("should keep device online when recent data exists", func(t *testing.T) {
		mockRedis := &MockRedisClient{}
		mockDeviceRepo := &MockDeviceRepository{}
		mockCollectorRepo := &MockCollectorRepository{}

		// 设置最后在线时间为5分钟前
		lastSeen := time.Now().Add(-5 * time.Minute)
		lastSeenStr := lastSeen.Format(time.RFC3339)

		device := &models.Device{
			ID:       1,
			Name:     "Test Device",
			Status:   models.DeviceStatusOnline,
			LastSeen: &lastSeen,
		}

		mockRedis.On("Get", mock.Anything, "device:last_seen:1").Return(lastSeenStr, nil)
		mockDeviceRepo.On("GetByID", uint(1)).Return(device, nil)
		mockCollectorRepo.On("GetByDeviceID", uint(1)).Return(nil, nil)

		checker := NewDeviceStatusChecker(mockDeviceRepo, mockCollectorRepo, mockRedis)
		checker.SetOfflineTimeout(10 * time.Minute)

		status, err := checker.CheckSingleDevice(context.Background(), 1)

		assert.NoError(t, err)
		assert.Equal(t, models.DeviceStatusOnline, status)
	})
}
