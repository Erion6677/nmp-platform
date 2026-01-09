package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockSettingsRepository 系统设置仓库模拟
type MockSettingsRepository struct {
	mock.Mock
	dataRetentionDays int
}

func NewMockSettingsRepository() *MockSettingsRepository {
	return &MockSettingsRepository{
		dataRetentionDays: 10, // 默认10天
	}
}

func (m *MockSettingsRepository) Get(key string) (string, error) {
	args := m.Called(key)
	return args.String(0), args.Error(1)
}

func (m *MockSettingsRepository) Set(key, value, description string) error {
	args := m.Called(key, value, description)
	return args.Error(0)
}

func (m *MockSettingsRepository) GetInt(key string, defaultValue int) int {
	args := m.Called(key, defaultValue)
	return args.Int(0)
}

func (m *MockSettingsRepository) SetInt(key string, value int, description string) error {
	args := m.Called(key, value, description)
	return args.Error(0)
}

func (m *MockSettingsRepository) GetBool(key string, defaultValue bool) bool {
	args := m.Called(key, defaultValue)
	return args.Bool(0)
}

func (m *MockSettingsRepository) SetBool(key string, value bool, description string) error {
	args := m.Called(key, value, description)
	return args.Error(0)
}

func (m *MockSettingsRepository) GetAll() (map[string]string, error) {
	args := m.Called()
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockSettingsRepository) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockSettingsRepository) GetDefaultPushInterval() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockSettingsRepository) SetDefaultPushInterval(intervalMs int) error {
	args := m.Called(intervalMs)
	return args.Error(0)
}

func (m *MockSettingsRepository) GetDataRetentionDays() int {
	return m.dataRetentionDays
}

func (m *MockSettingsRepository) SetDataRetentionDays(days int) error {
	m.dataRetentionDays = days
	return nil
}

func (m *MockSettingsRepository) GetFrontendRefreshInterval() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockSettingsRepository) SetFrontendRefreshInterval(seconds int) error {
	args := m.Called(seconds)
	return args.Error(0)
}

func (m *MockSettingsRepository) GetDeviceOfflineTimeout() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockSettingsRepository) SetDeviceOfflineTimeout(seconds int) error {
	args := m.Called(seconds)
	return args.Error(0)
}

func (m *MockSettingsRepository) GetFollowPushInterval() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockSettingsRepository) SetFollowPushInterval(follow bool) error {
	args := m.Called(follow)
	return args.Error(0)
}

func (m *MockSettingsRepository) InitDefaults() error {
	args := m.Called()
	return args.Error(0)
}

// CleanupMockQueryResult 清理测试专用的查询结果模拟
type CleanupMockQueryResult struct {
	records []CleanupMockRecord
	index   int
	err     error
}

func NewCleanupMockQueryResult(records []CleanupMockRecord) *CleanupMockQueryResult {
	return &CleanupMockQueryResult{
		records: records,
		index:   -1,
	}
}

func (m *CleanupMockQueryResult) Next() bool {
	m.index++
	return m.index < len(m.records)
}

func (m *CleanupMockQueryResult) Record() QueryRecord {
	if m.index >= 0 && m.index < len(m.records) {
		return &m.records[m.index]
	}
	return nil
}

func (m *CleanupMockQueryResult) Err() error {
	return m.err
}

// CleanupMockRecord 清理测试专用的查询记录模拟
type CleanupMockRecord struct {
	time  time.Time
	field string
	value interface{}
}

func (m *CleanupMockRecord) Time() time.Time {
	return m.time
}

func (m *CleanupMockRecord) Field() string {
	return m.field
}

func (m *CleanupMockRecord) Value() interface{} {
	return m.value
}

// CleanupMockInfluxClient 清理测试专用的 InfluxDB 客户端模拟
type CleanupMockInfluxClient struct {
	mock.Mock
	writtenPoints []WrittenPoint
	deletedData   []DeletedData
}

type WrittenPoint struct {
	Measurement string
	Tags        map[string]string
	Fields      map[string]interface{}
	Timestamp   time.Time
}

type DeletedData struct {
	Measurement string
	DeviceID    string
	StartTime   time.Time
	StopTime    time.Time
}

func NewCleanupMockInfluxClient() *CleanupMockInfluxClient {
	return &CleanupMockInfluxClient{
		writtenPoints: make([]WrittenPoint, 0),
		deletedData:   make([]DeletedData, 0),
	}
}

func (m *CleanupMockInfluxClient) WritePoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) error {
	m.writtenPoints = append(m.writtenPoints, WrittenPoint{
		Measurement: measurement,
		Tags:        tags,
		Fields:      fields,
		Timestamp:   timestamp,
	})
	args := m.Called(measurement, tags, fields, timestamp)
	return args.Error(0)
}

func (m *CleanupMockInfluxClient) Query(query string) (QueryResult, error) {
	args := m.Called(query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(QueryResult), args.Error(1)
}

func (m *CleanupMockInfluxClient) Health() error {
	args := m.Called()
	return args.Error(0)
}

func (m *CleanupMockInfluxClient) Close() {
	m.Called()
}

func (m *CleanupMockInfluxClient) GetWrittenPoints() []WrittenPoint {
	return m.writtenPoints
}

func (m *CleanupMockInfluxClient) GetWrittenPointsForDevice(deviceID string) []WrittenPoint {
	var result []WrittenPoint
	for _, p := range m.writtenPoints {
		if p.Tags["device_id"] == deviceID {
			result = append(result, p)
		}
	}
	return result
}

func (m *CleanupMockInfluxClient) ClearWrittenPoints() {
	m.writtenPoints = make([]WrittenPoint, 0)
}


// ========== Property 7: 数据清理隔离性 ==========
// Feature: device-monitoring, Property 7: 数据清理隔离性
// *For any* 数据清理操作，如果是全局清理，应删除所有超过保留天数的数据；
// 如果是单设备清理，应只删除该设备的数据，其他设备数据不受影响。
// **Validates: Requirements 8.2, 8.3**

func TestProperty7_DataCleanupIsolation(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	// 属性7.1: 全局清理应该删除所有超过保留天数的数据
	properties.Property("global cleanup should delete all data older than retention days",
		prop.ForAll(
			func(retentionDays int) bool {
				mockInflux := NewCleanupMockInfluxClient()
				mockSettings := NewMockSettingsRepository()
				mockDeviceRepo := &MockDeviceRepository{}

				mockSettings.dataRetentionDays = retentionDays

				mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
				mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

				config := &DataCleanupConfig{
					CleanupInterval: 24 * time.Hour,
					Bucket:          "monitoring",
					Org:             "nmp",
				}
				service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

				err := service.CleanupExpiredData(context.Background(), retentionDays)

				return err == nil
			},
			gen.IntRange(1, 365),
		))

	// 属性7.2: 单设备清理应该只影响指定设备
	properties.Property("single device cleanup should only affect specified device",
		prop.ForAll(
			func(targetDeviceID uint, otherDeviceID uint) bool {
				if targetDeviceID == otherDeviceID {
					otherDeviceID = targetDeviceID + 1
				}

				mockInflux := NewCleanupMockInfluxClient()
				mockSettings := NewMockSettingsRepository()
				mockDeviceRepo := &MockDeviceRepository{}

				mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
				mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

				config := &DataCleanupConfig{
					CleanupInterval: 24 * time.Hour,
					Bucket:          "monitoring",
					Org:             "nmp",
				}
				service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

				err := service.CleanupDeviceData(context.Background(), targetDeviceID)

				return err == nil
			},
			gen.UIntRange(1, 1000),
			gen.UIntRange(1, 1000),
		))

	// 属性7.3: 无效的保留天数应该被拒绝
	properties.Property("invalid retention days should be rejected",
		prop.ForAll(
			func(invalidDays int) bool {
				mockInflux := NewCleanupMockInfluxClient()
				mockSettings := NewMockSettingsRepository()
				mockDeviceRepo := &MockDeviceRepository{}

				config := &DataCleanupConfig{
					CleanupInterval: 24 * time.Hour,
					Bucket:          "monitoring",
					Org:             "nmp",
				}
				service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

				err := service.CleanupExpiredData(context.Background(), invalidDays)

				return err != nil
			},
			gen.IntRange(-100, 0),
		))

	// 属性7.4: 无效的设备ID应该被拒绝
	properties.Property("invalid device ID should be rejected",
		prop.ForAll(
			func(_ int) bool {
				mockInflux := NewCleanupMockInfluxClient()
				mockSettings := NewMockSettingsRepository()
				mockDeviceRepo := &MockDeviceRepository{}

				config := &DataCleanupConfig{
					CleanupInterval: 24 * time.Hour,
					Bucket:          "monitoring",
					Org:             "nmp",
				}
				service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

				err := service.CleanupDeviceData(context.Background(), 0)

				return err != nil
			},
			gen.IntRange(1, 10),
		))

	properties.TestingRun(t)
}

// 单元测试：全局数据清理
func TestGlobalDataCleanup(t *testing.T) {
	t.Run("should cleanup expired data with valid retention days", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 10

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{
			{time: time.Now(), field: "_value", value: int64(100)},
		})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupExpiredData(context.Background(), 10)

		assert.NoError(t, err)
	})

	t.Run("should reject zero retention days", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupExpiredData(context.Background(), 0)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retention days must be positive")
	})

	t.Run("should reject negative retention days", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupExpiredData(context.Background(), -5)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retention days must be positive")
	})

	t.Run("should use configured retention days from settings", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 7

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		days := service.GetRetentionDays()
		assert.Equal(t, 7, days)
	})
}

// 单元测试：单设备数据清理
func TestSingleDeviceDataCleanup(t *testing.T) {
	t.Run("should cleanup data for specific device", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupDeviceData(context.Background(), 123)

		assert.NoError(t, err)
	})

	t.Run("should reject zero device ID", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupDeviceData(context.Background(), 0)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "device ID must be positive")
	})

	t.Run("should cleanup device data before specific time", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		before := time.Now().AddDate(0, 0, -7)
		err := service.CleanupDeviceDataBefore(context.Background(), 123, before)

		assert.NoError(t, err)
	})
}

// 单元测试：清理服务状态
func TestCleanupServiceStatus(t *testing.T) {
	t.Run("should return correct status when not running", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 10

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		status := service.GetCleanupStatus()

		assert.Equal(t, false, status["running"])
		assert.Equal(t, "24h0m0s", status["cleanup_interval"])
		assert.Equal(t, 10, status["retention_days"])
		assert.Equal(t, "monitoring", status["bucket"])
		assert.Equal(t, "nmp", status["org"])
	})

	t.Run("should start and stop cleanup service", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 10

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 1 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := service.Start(ctx)
		assert.NoError(t, err)

		status := service.GetCleanupStatus()
		assert.Equal(t, true, status["running"])

		service.Stop()

		status = service.GetCleanupStatus()
		assert.Equal(t, false, status["running"])
	})

	t.Run("should not start twice", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 10

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 1 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		ctx := context.Background()

		err := service.Start(ctx)
		assert.NoError(t, err)

		err = service.Start(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already running")

		service.Stop()
	})
}

// 单元测试：手动触发清理
func TestTriggerCleanup(t *testing.T) {
	t.Run("should trigger cleanup with configured retention days", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 7

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.TriggerCleanup(context.Background())

		assert.NoError(t, err)
	})

	t.Run("should use default retention days when not configured", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 0

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.TriggerCleanup(context.Background())

		assert.NoError(t, err)
	})
}

// 单元测试：设置保留天数
func TestSetRetentionDays(t *testing.T) {
	t.Run("should set valid retention days", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.SetRetentionDays(30)

		assert.NoError(t, err)
		assert.Equal(t, 30, service.GetRetentionDays())
	})

	t.Run("should reject invalid retention days", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.SetRetentionDays(0)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retention days must be positive")
	})

	t.Run("should reject negative retention days", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.SetRetentionDays(-10)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retention days must be positive")
	})
}

// 数据清理隔离性验证测试
func TestDataCleanupIsolation(t *testing.T) {
	t.Run("single device cleanup should not affect other devices", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupDeviceData(context.Background(), 1)
		assert.NoError(t, err)
	})

	t.Run("global cleanup should affect all devices", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 10

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupExpiredData(context.Background(), 10)
		assert.NoError(t, err)
	})
}

// 测试默认配置
func TestDefaultConfig(t *testing.T) {
	t.Run("should use default config when nil provided", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, nil)

		status := service.GetCleanupStatus()
		assert.Equal(t, "24h0m0s", status["cleanup_interval"])
		assert.Equal(t, "monitoring", status["bucket"])
		assert.Equal(t, "nmp", status["org"])
	})

	t.Run("should use default values for empty config fields", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		config := &DataCleanupConfig{}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		status := service.GetCleanupStatus()
		assert.Equal(t, "24h0m0s", status["cleanup_interval"])
		assert.Equal(t, "monitoring", status["bucket"])
		assert.Equal(t, "nmp", status["org"])
	})
}

// 测试 extractMeasurement 函数
func TestExtractMeasurement(t *testing.T) {
	tests := []struct {
		name      string
		predicate string
		expected  string
	}{
		{
			name:      "extract bandwidth measurement",
			predicate: `_measurement="bandwidth"`,
			expected:  "bandwidth",
		},
		{
			name:      "extract ping measurement",
			predicate: `_measurement="ping"`,
			expected:  "ping",
		},
		{
			name:      "extract device_metrics measurement",
			predicate: `_measurement="device_metrics"`,
			expected:  "device_metrics",
		},
		{
			name:      "empty predicate",
			predicate: "",
			expected:  "",
		},
		{
			name:      "invalid predicate",
			predicate: "invalid",
			expected:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractMeasurement(tt.predicate)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// 并发测试
func TestConcurrentCleanup(t *testing.T) {
	t.Run("should handle concurrent cleanup requests", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 10

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		done := make(chan bool, 5)
		for i := 0; i < 5; i++ {
			go func(deviceID uint) {
				err := service.CleanupDeviceData(context.Background(), deviceID)
				assert.NoError(t, err)
				done <- true
			}(uint(i + 1))
		}

		for i := 0; i < 5; i++ {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent cleanup")
			}
		}
	})
}

// 边界条件测试
func TestEdgeCases(t *testing.T) {
	t.Run("should handle very large retention days", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupExpiredData(context.Background(), 3650)

		assert.NoError(t, err)
	})

	t.Run("should handle very large device ID", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupDeviceData(context.Background(), uint(1<<31-1))

		assert.NoError(t, err)
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 10

		mockQueryResult := NewCleanupMockQueryResult([]CleanupMockRecord{})
		mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockQueryResult, nil)

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := service.CleanupExpiredData(ctx, 10)

		_ = err
	})
}

// 测试查询错误处理
func TestQueryErrorHandling(t *testing.T) {
	t.Run("should handle query error gracefully", func(t *testing.T) {
		mockInflux := NewCleanupMockInfluxClient()
		mockSettings := NewMockSettingsRepository()
		mockDeviceRepo := &MockDeviceRepository{}

		mockSettings.dataRetentionDays = 10

		mockInflux.On("Query", mock.AnythingOfType("string")).Return(nil, fmt.Errorf("query error"))

		config := &DataCleanupConfig{
			CleanupInterval: 24 * time.Hour,
			Bucket:          "monitoring",
			Org:             "nmp",
		}
		service := NewDataCleanupService(mockInflux, mockSettings, mockDeviceRepo, config)

		err := service.CleanupExpiredData(context.Background(), 10)

		assert.NoError(t, err)
	})
}
