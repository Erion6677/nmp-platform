package service

import (
	"context"
	"fmt"
	"nmp-platform/internal/models"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockQueryResult 模拟查询结果
type MockQueryResult struct {
	records []MockRecord
	index   int
	err     error
}

type MockRecord struct {
	time  time.Time
	field string
	value interface{}
}

func (r *MockRecord) Time() time.Time      { return r.time }
func (r *MockRecord) Field() string       { return r.field }
func (r *MockRecord) Value() interface{}  { return r.value }

func (m *MockQueryResult) Next() bool {
	if m.index < len(m.records) {
		m.index++
		return true
	}
	return false
}

func (m *MockQueryResult) Record() QueryRecord {
	if m.index > 0 && m.index <= len(m.records) {
		return &m.records[m.index-1]
	}
	return nil
}

func (m *MockQueryResult) Err() error {
	return m.err
}

// 扩展现有的MockInfluxClient以支持查询
func (m *MockInfluxClient) Query(query string) (QueryResult, error) {
	args := m.Called(query)
	return args.Get(0).(QueryResult), args.Error(1)
}

// 扩展现有的MockRedisClient以支持更多方法
func NewMockRedisClientWithData() *MockRedisClient {
	client := &MockRedisClient{}
	return client
}

// TestDataQueryCorrectness 测试数据查询正确性属性
// Feature: network-monitoring-platform, Property 7: 数据查询正确性
// **验证需求: 6.1, 6.2, 6.3, 6.4, 6.5**
func TestDataQueryCorrectness(t *testing.T) {
	properties := gopter.NewProperties(gopter.DefaultTestParameters())
	properties.TestingRun(t)

	// 属性1: 实时数据查询应该从Redis返回最新数据
	properties.Property("real-time data query should return latest data from Redis", 
		prop.ForAll(
			func(deviceID string, metrics map[string]interface{}) bool {
				// 创建模拟客户端
				mockRedis := &MockRedisClient{}
				mockInflux := &MockInfluxClient{}
				
				// 设置模拟数据
				latestData := models.MetricData{
					DeviceID:  deviceID,
					Timestamp: time.Now(),
					Metrics:   metrics,
				}
				
				mockRedis.On("GetJSON", mock.Anything, fmt.Sprintf("device:latest:%s", deviceID), mock.Anything).
					Run(func(args mock.Arguments) {
						dest := args.Get(2).(*models.MetricData)
						*dest = latestData
					}).Return(nil)
				
				mockRedis.On("Get", mock.Anything, fmt.Sprintf("device:last_seen:%s", deviceID)).
					Return(time.Now().Format(time.RFC3339), nil)
				
				// 创建查询服务
				service := NewDataQueryService(mockInflux, mockRedis)
				
				// 执行查询
				req := &RealTimeQueryRequest{
					DeviceIDs: []string{deviceID},
				}
				
				response, err := service.QueryRealTimeData(context.Background(), req)
				
				// 验证结果
				if err != nil {
					return false
				}
				
				if len(response.Devices) != 1 {
					return false
				}
				
				device := response.Devices[0]
				return device.DeviceID == deviceID && len(device.Metrics) == len(metrics)
			},
			genValidDeviceID(),
			genMetricsMap(),
		))

	// 属性2: 历史数据查询应该从InfluxDB返回指定时间范围的数据
	properties.Property("historical data query should return data from InfluxDB for specified time range", 
		prop.ForAll(
			func(deviceID string, startTime, endTime time.Time, metrics []string) bool {
				if startTime.After(endTime) {
					startTime, endTime = endTime, startTime
				}
				
				// 创建模拟客户端
				mockRedis := &MockRedisClient{}
				mockInflux := &MockInfluxClient{}
				
				// 设置模拟查询结果
				mockResult := &MockQueryResult{
					records: []MockRecord{
						{time: startTime.Add(time.Hour), field: "cpu_usage", value: 50.0},
						{time: startTime.Add(2 * time.Hour), field: "memory_usage", value: 60.0},
					},
				}
				
				mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockResult, nil)
				
				// 创建查询服务
				service := NewDataQueryService(mockInflux, mockRedis)
				
				// 执行查询
				req := &QueryRequest{
					DeviceID:  deviceID,
					StartTime: &startTime,
					EndTime:   &endTime,
					Metrics:   metrics,
				}
				
				response, err := service.QueryHistoricalData(context.Background(), req)
				
				// 验证结果
				if err != nil {
					return false
				}
				
				// 验证返回的数据在指定时间范围内
				for _, point := range response.DataPoints {
					if point.Timestamp.Before(startTime) || point.Timestamp.After(endTime) {
						return false
					}
				}
				
				return response.DeviceID == deviceID
			},
			genValidDeviceID(),
			genValidTime(),
			genValidTime(),
			genMetricsList(),
		))

	// 属性3: 数据查询分页应该正确限制返回的数据量
	properties.Property("data query pagination should correctly limit returned data", 
		prop.ForAll(
			func(deviceID string, limit, page int) bool {
				if limit <= 0 || limit > 1000 {
					limit = 100
				}
				if page < 0 {
					page = 0
				}
				
				// 创建模拟客户端
				mockRedis := &MockRedisClient{}
				mockInflux := &MockInfluxClient{}
				
				// 创建大量模拟数据
				var records []MockRecord
				totalRecords := limit*3 + 10 // 确保有足够的数据进行分页测试
				baseTime := time.Now().Add(-24 * time.Hour)
				
				for i := 0; i < totalRecords; i++ {
					records = append(records, MockRecord{
						time:  baseTime.Add(time.Duration(i) * time.Minute),
						field: "cpu_usage",
						value: float64(i % 100),
					})
				}
				
				mockResult := &MockQueryResult{records: records}
				mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockResult, nil)
				
				// 创建查询服务
				service := NewDataQueryService(mockInflux, mockRedis)
				
				// 执行查询
				req := &QueryRequest{
					DeviceID: deviceID,
					Limit:    limit,
					Page:     page,
				}
				
				response, err := service.QueryHistoricalData(context.Background(), req)
				
				// 验证结果
				if err != nil {
					return false
				}
				
				// 验证分页信息
				if response.Pagination == nil {
					return false
				}
				
				// 验证返回的数据量不超过限制
				if len(response.DataPoints) > limit {
					return false
				}
				
				// 验证分页信息的正确性
				expectedTotalPages := (totalRecords + limit - 1) / limit
				return response.Pagination.Limit == limit && 
					   response.Pagination.Page == page &&
					   response.Pagination.TotalPages == expectedTotalPages
			},
			genValidDeviceID(),
			gen.IntRange(1, 1000),
			gen.IntRange(0, 10),
		))

	// 属性4: 数据聚合查询应该根据指定粒度返回聚合数据
	properties.Property("aggregated data query should return data with specified granularity", 
		prop.ForAll(
			func(deviceID string, granularity string) bool {
				validGranularities := []string{"1m", "5m", "15m", "1h", "6h", "1d"}
				isValid := false
				for _, g := range validGranularities {
					if g == granularity {
						isValid = true
						break
					}
				}
				if !isValid {
					granularity = "1h" // 使用默认值
				}
				
				// 创建模拟客户端
				mockRedis := &MockRedisClient{}
				mockInflux := &MockInfluxClient{}
				
				// 设置模拟聚合查询结果
				mockResult := &MockQueryResult{
					records: []MockRecord{
						{time: time.Now().Add(-2 * time.Hour), field: "cpu_usage", value: 45.0},
						{time: time.Now().Add(-1 * time.Hour), field: "cpu_usage", value: 55.0},
					},
				}
				
				mockInflux.On("Query", mock.AnythingOfType("string")).Return(mockResult, nil)
				
				// 创建查询服务
				service := NewDataQueryService(mockInflux, mockRedis)
				
				// 执行聚合查询
				req := &QueryRequest{
					DeviceID:    deviceID,
					Granularity: granularity,
				}
				
				response, err := service.QueryAggregatedData(context.Background(), req)
				
				// 验证结果
				if err != nil {
					return false
				}
				
				// 验证查询摘要包含正确的粒度信息
				return response.Summary != nil && response.Summary.Granularity == granularity
			},
			genValidDeviceID(),
			gen.OneConstOf("1m", "5m", "15m", "1h", "6h", "1d"),
		))

	// 属性5: 查询请求验证应该拒绝无效参数
	properties.Property("query request validation should reject invalid parameters", 
		prop.ForAll(
			func(deviceID string, startTime, endTime time.Time, limit int, page int) bool {
				// 创建模拟客户端
				mockRedis := &MockRedisClient{}
				mockInflux := &MockInfluxClient{}
				
				// 创建查询服务
				service := NewDataQueryService(mockInflux, mockRedis)
				
				// 测试无效的设备ID
				if deviceID == "" {
					req := &QueryRequest{DeviceID: deviceID}
					_, err := service.QueryHistoricalData(context.Background(), req)
					if err == nil {
						return false // 应该返回错误
					}
				}
				
				// 测试无效的时间范围（开始时间晚于结束时间）
				if startTime.After(endTime) {
					req := &QueryRequest{
						DeviceID:  "valid-device",
						StartTime: &startTime,
						EndTime:   &endTime,
					}
					_, err := service.QueryHistoricalData(context.Background(), req)
					if err == nil {
						return false // 应该返回错误
					}
				}
				
				// 测试无效的限制参数
				if limit < 0 || limit > 10000 {
					req := &QueryRequest{
						DeviceID: "valid-device",
						Limit:    limit,
					}
					_, err := service.QueryHistoricalData(context.Background(), req)
					if err == nil && (limit < 0 || limit > 10000) {
						return false // 应该返回错误
					}
				}
				
				// 测试无效的页码参数
				if page < 0 {
					req := &QueryRequest{
						DeviceID: "valid-device",
						Page:     page,
					}
					_, err := service.QueryHistoricalData(context.Background(), req)
					if err == nil {
						return false // 应该返回错误
					}
				}
				
				return true
			},
			gen.OneConstOf("", genValidDeviceID()),
			genValidTime(),
			genValidTime(),
			gen.IntRange(-100, 20000),
			gen.IntRange(-10, 100),
		))
}

// 生成器函数

func genValidDeviceID() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0 && len(s) <= 50
	})
}

func genMetricsMap() gopter.Gen {
	return gen.MapOf(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 }),
		gen.Float64Range(0, 100),
	).Map(func(m map[string]float64) map[string]interface{} {
		result := make(map[string]interface{})
		for k, v := range m {
			result[k] = v
		}
		return result
	}).SuchThat(func(m map[string]interface{}) bool {
		return len(m) > 0 && len(m) <= 10
	})
}

func genMetricsList() gopter.Gen {
	return gen.SliceOf(gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0
	})).SuchThat(func(slice []string) bool {
		return len(slice) <= 5
	})
}

func genValidTime() gopter.Gen {
	return gen.Int64Range(
		time.Now().Add(-30*24*time.Hour).Unix(), // 30天前的Unix时间戳
		time.Now().Unix(),                       // 现在的Unix时间戳
	).Map(func(timestamp int64) time.Time {
		return time.Unix(timestamp, 0)
	})
}

// 单元测试辅助函数

func TestDataQueryService_QueryRealTimeData_ValidInput(t *testing.T) {
	// 创建模拟客户端
	mockRedis := &MockRedisClient{}
	mockInflux := &MockInfluxClient{}
	
	// 设置测试数据
	deviceID := "test-device-1"
	testData := models.MetricData{
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Metrics: map[string]interface{}{
			"cpu_usage":    75.5,
			"memory_usage": 60.2,
		},
	}
	
	mockRedis.On("GetJSON", mock.Anything, fmt.Sprintf("device:latest:%s", deviceID), mock.Anything).
		Run(func(args mock.Arguments) {
			dest := args.Get(2).(*models.MetricData)
			*dest = testData
		}).Return(nil)
	
	mockRedis.On("Get", mock.Anything, fmt.Sprintf("device:last_seen:%s", deviceID)).
		Return(time.Now().Format(time.RFC3339), nil)
	
	// 创建服务
	service := NewDataQueryService(mockInflux, mockRedis)
	
	// 执行测试
	req := &RealTimeQueryRequest{
		DeviceIDs: []string{deviceID},
	}
	
	response, err := service.QueryRealTimeData(context.Background(), req)
	
	// 验证结果
	assert.NoError(t, err)
	assert.Len(t, response.Devices, 1)
	assert.Equal(t, deviceID, response.Devices[0].DeviceID)
	assert.Equal(t, "online", response.Devices[0].Status)
	assert.Len(t, response.Devices[0].Metrics, 2)
	
	mockRedis.AssertExpectations(t)
}

func TestDataQueryService_QueryHistoricalData_InvalidDeviceID(t *testing.T) {
	// 创建模拟客户端
	mockRedis := &MockRedisClient{}
	mockInflux := &MockInfluxClient{}
	
	// 创建服务
	service := NewDataQueryService(mockInflux, mockRedis)
	
	// 测试空设备ID
	req := &QueryRequest{
		DeviceID: "",
	}
	
	_, err := service.QueryHistoricalData(context.Background(), req)
	
	// 验证应该返回错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device_id is required")
}

func TestDataQueryService_QueryHistoricalData_InvalidTimeRange(t *testing.T) {
	// 创建模拟客户端
	mockRedis := &MockRedisClient{}
	mockInflux := &MockInfluxClient{}
	
	// 创建服务
	service := NewDataQueryService(mockInflux, mockRedis)
	
	// 测试无效时间范围
	startTime := time.Now()
	endTime := startTime.Add(-1 * time.Hour) // 结束时间早于开始时间
	
	req := &QueryRequest{
		DeviceID:  "test-device",
		StartTime: &startTime,
		EndTime:   &endTime,
	}
	
	_, err := service.QueryHistoricalData(context.Background(), req)
	
	// 验证应该返回错误
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start_time must be before end_time")
}

func TestDataQueryService_ApplyPagination(t *testing.T) {
	// 创建模拟客户端
	mockRedis := &MockRedisClient{}
	mockInflux := &MockInfluxClient{}
	
	// 创建服务
	service := NewDataQueryService(mockInflux, mockRedis)
	
	// 创建测试数据
	dataPoints := make([]models.DataPoint, 25)
	for i := 0; i < 25; i++ {
		dataPoints[i] = models.DataPoint{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Values:    map[string]interface{}{"value": i},
		}
	}
	
	// 测试分页
	paginatedData, pagination := service.applyPagination(dataPoints, 1, 10)
	
	// 验证结果
	assert.Len(t, paginatedData, 10)
	assert.Equal(t, 1, pagination.Page)
	assert.Equal(t, 10, pagination.Limit)
	assert.Equal(t, 25, pagination.Total)
	assert.Equal(t, 3, pagination.TotalPages)
}