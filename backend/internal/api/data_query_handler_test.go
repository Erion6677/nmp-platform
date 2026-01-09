package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/stretchr/testify/assert"
)

// Feature: device-monitoring, Property 9: 时间范围限制
// **Validates: Requirements 10.7**

// TestTimeRangeValidation_Property 属性测试：时间范围限制
// 验证自定义时间范围查询，时间跨度不应超过 24 小时
func TestTimeRangeValidation_Property(t *testing.T) {
	gin.SetMode(gin.TestMode)

	properties := gopter.NewProperties(gopter.DefaultTestParameters())

	// 属性: 对于任意时间范围，如果超过24小时，应该返回错误
	properties.Property("time range exceeding 24 hours should be rejected",
		prop.ForAll(
			func(startOffset, endOffset int64) bool {
				// 创建测试处理器
				handler := &DataQueryHandler{}

				// 计算时间范围
				now := time.Now()
				startTime := now.Add(-time.Duration(startOffset) * time.Hour)
				endTime := now.Add(-time.Duration(endOffset) * time.Hour)

				// 确保 startTime < endTime
				if startTime.After(endTime) {
					startTime, endTime = endTime, startTime
				}

				// 验证时间范围
				err := handler.validateTimeRange(startTime, endTime)

				// 计算实际时间跨度
				duration := endTime.Sub(startTime)

				// 如果时间跨度超过24小时，应该返回错误
				if duration > MaxQueryTimeRange {
					return err != nil
				}

				// 如果时间跨度在24小时内，应该不返回错误
				return err == nil
			},
			gen.Int64Range(0, 72),  // 开始时间偏移（小时）
			gen.Int64Range(0, 72),  // 结束时间偏移（小时）
		))

	// 属性: 对于任意有效时间范围（<=24小时），应该通过验证
	properties.Property("valid time range within 24 hours should pass validation",
		prop.ForAll(
			func(durationHours int64) bool {
				// 创建测试处理器
				handler := &DataQueryHandler{}

				// 创建有效的时间范围
				now := time.Now()
				startTime := now.Add(-time.Duration(durationHours) * time.Hour)
				endTime := now

				// 验证时间范围
				err := handler.validateTimeRange(startTime, endTime)

				// 24小时内的时间范围应该通过验证
				return err == nil
			},
			gen.Int64Range(0, 24), // 时间跨度（小时），0-24小时
		))

	// 属性: 开始时间晚于结束时间应该返回错误
	properties.Property("start time after end time should be rejected",
		prop.ForAll(
			func(offsetHours int64) bool {
				if offsetHours <= 0 {
					offsetHours = 1
				}

				// 创建测试处理器
				handler := &DataQueryHandler{}

				// 创建无效的时间范围（开始时间晚于结束时间）
				now := time.Now()
				startTime := now
				endTime := now.Add(-time.Duration(offsetHours) * time.Hour)

				// 验证时间范围
				err := handler.validateTimeRange(startTime, endTime)

				// 应该返回错误
				return err != nil
			},
			gen.Int64Range(1, 48), // 偏移小时数
		))

	properties.TestingRun(t)
}

// TestParseTimeRange_PresetRanges 测试预设时间范围解析
func TestParseTimeRange_PresetRanges(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		rangeParam    string
		expectedHours int
		shouldError   bool
	}{
		{"1 hour range", "1h", 1, false},
		{"6 hours range", "6h", 6, false},
		{"12 hours range", "12h", 12, false},
		{"24 hours range", "24h", 24, false},
		{"invalid range", "48h", 0, true},
		{"invalid format", "abc", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建测试请求
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/?range="+tc.rangeParam, nil)

			handler := &DataQueryHandler{}
			startTime, endTime, err := handler.parseTimeRange(c)

			if tc.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				duration := endTime.Sub(startTime)
				expectedDuration := time.Duration(tc.expectedHours) * time.Hour
				// 允许1秒的误差
				assert.InDelta(t, expectedDuration.Seconds(), duration.Seconds(), 1)
			}
		})
	}
}

// TestParseTimeRange_CustomRange 测试自定义时间范围解析
func TestParseTimeRange_CustomRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	startTime := now.Add(-6 * time.Hour)
	endTime := now

	// 创建测试请求，使用 URL 编码
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/", nil)
	q := req.URL.Query()
	q.Add("start_time", startTime.Format(time.RFC3339))
	q.Add("end_time", endTime.Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()
	c.Request = req

	handler := &DataQueryHandler{}
	parsedStart, parsedEnd, err := handler.parseTimeRange(c)

	assert.NoError(t, err)
	// 允许1秒的误差
	assert.InDelta(t, startTime.Unix(), parsedStart.Unix(), 1)
	assert.InDelta(t, endTime.Unix(), parsedEnd.Unix(), 1)
}

// TestParseTimeRange_DefaultRange 测试默认时间范围（12小时）
func TestParseTimeRange_DefaultRange(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试请求（不带任何时间参数）
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)

	handler := &DataQueryHandler{}
	startTime, endTime, err := handler.parseTimeRange(c)

	assert.NoError(t, err)
	duration := endTime.Sub(startTime)
	// 默认应该是12小时
	assert.InDelta(t, 12*time.Hour.Seconds(), duration.Seconds(), 1)
}

// TestValidateTimeRange_ExceedsMax 测试超过最大时间范围
func TestValidateTimeRange_ExceedsMax(t *testing.T) {
	handler := &DataQueryHandler{}

	now := time.Now()
	startTime := now.Add(-25 * time.Hour) // 25小时前
	endTime := now

	err := handler.validateTimeRange(startTime, endTime)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time range cannot exceed 24 hours")
}

// TestValidateTimeRange_ExactlyMax 测试恰好等于最大时间范围
func TestValidateTimeRange_ExactlyMax(t *testing.T) {
	handler := &DataQueryHandler{}

	now := time.Now()
	startTime := now.Add(-24 * time.Hour) // 恰好24小时
	endTime := now

	err := handler.validateTimeRange(startTime, endTime)

	assert.NoError(t, err)
}

// TestValidateTimeRange_StartAfterEnd 测试开始时间晚于结束时间
func TestValidateTimeRange_StartAfterEnd(t *testing.T) {
	handler := &DataQueryHandler{}

	now := time.Now()
	startTime := now
	endTime := now.Add(-1 * time.Hour) // 结束时间早于开始时间

	err := handler.validateTimeRange(startTime, endTime)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "start_time must be before end_time")
}

// TestQueryBandwidthData_TimeRangeValidation 测试带宽查询的时间范围验证
func TestQueryBandwidthData_TimeRangeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试请求，时间范围超过24小时
	now := time.Now()
	startTime := now.Add(-48 * time.Hour)
	endTime := now

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/metrics/bandwidth/1", nil)
	q := req.URL.Query()
	q.Add("start_time", startTime.Format(time.RFC3339))
	q.Add("end_time", endTime.Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()
	c.Request = req
	c.Params = gin.Params{{Key: "device_id", Value: "1"}}

	handler := &DataQueryHandler{}
	handler.QueryBandwidthData(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "time range cannot exceed 24 hours")
}

// TestQueryPingData_TimeRangeValidation 测试 Ping 查询的时间范围验证
func TestQueryPingData_TimeRangeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试请求，时间范围超过24小时
	now := time.Now()
	startTime := now.Add(-48 * time.Hour)
	endTime := now

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest("GET", "/metrics/ping/1", nil)
	q := req.URL.Query()
	q.Add("start_time", startTime.Format(time.RFC3339))
	q.Add("end_time", endTime.Format(time.RFC3339))
	req.URL.RawQuery = q.Encode()
	c.Request = req
	c.Params = gin.Params{{Key: "device_id", Value: "1"}}

	handler := &DataQueryHandler{}
	handler.QueryPingData(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "time range cannot exceed 24 hours")
}
