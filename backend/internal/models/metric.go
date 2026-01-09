package models

import (
	"time"
)

// MetricData 监控指标数据结构
type MetricData struct {
	DeviceID  string                 `json:"device_id" binding:"required"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics" binding:"required"`
	Tags      map[string]string      `json:"tags"`
}

// PushDataRequest 数据推送请求结构
type PushDataRequest struct {
	DeviceID  string                 `json:"device_id" binding:"required"`
	Timestamp *time.Time             `json:"timestamp,omitempty"`
	Metrics   map[string]interface{} `json:"metrics" binding:"required"`
	Tags      map[string]string      `json:"tags,omitempty"`
}

// PushDataResponse 数据推送响应结构
type PushDataResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
}

// BatchPushDataRequest 批量数据推送请求结构
type BatchPushDataRequest struct {
	Data []PushDataRequest `json:"data" binding:"required,min=1,max=100"`
}

// BatchPushDataResponse 批量数据推送响应结构
type BatchPushDataResponse struct {
	Success     bool     `json:"success"`
	Message     string   `json:"message"`
	ProcessedCount int   `json:"processed_count"`
	FailedCount    int   `json:"failed_count"`
	Errors      []string `json:"errors,omitempty"`
}

// ValidateMetricData 验证监控数据
func (m *MetricData) Validate() error {
	if m.DeviceID == "" {
		return NewValidationError("device_id is required")
	}
	
	if len(m.Metrics) == 0 {
		return NewValidationError("metrics cannot be empty")
	}
	
	// 设置默认时间戳
	if m.Timestamp.IsZero() {
		m.Timestamp = time.Now()
	}
	
	// 初始化标签映射
	if m.Tags == nil {
		m.Tags = make(map[string]string)
	}
	
	return nil
}

// ToInfluxPoint 转换为InfluxDB数据点格式
func (m *MetricData) ToInfluxPoint(measurement string) map[string]interface{} {
	return map[string]interface{}{
		"measurement": measurement,
		"tags":        m.Tags,
		"fields":      m.Metrics,
		"time":        m.Timestamp,
	}
}

// ToRedisKey 生成Redis缓存键
func (m *MetricData) ToRedisKey(prefix string) string {
	return prefix + ":" + m.DeviceID
}

// DataPoint 数据点结构（用于历史数据查询）
type DataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Values    map[string]interface{} `json:"values"`
}

// DeviceRealTimeData 设备实时数据结构
type DeviceRealTimeData struct {
	DeviceID  string                 `json:"device_id"`
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// DeviceStatusInfo 设备状态信息结构（用于查询响应）
type DeviceStatusInfo struct {
	DeviceID string     `json:"device_id"`
	Status   string     `json:"status"`
	LastSeen *time.Time `json:"last_seen,omitempty"`
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// QuerySummary 查询摘要信息
type QuerySummary struct {
	TotalPoints   int        `json:"total_points"`
	StartTime     *time.Time `json:"start_time,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Granularity   string     `json:"granularity,omitempty"`
	QueryDuration string     `json:"query_duration"`
}

// MetricSummary 指标摘要信息
type MetricSummary struct {
	DeviceID  string    `json:"device_id"`
	Metric    string    `json:"metric"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Average   float64   `json:"average"`
	Count     int       `json:"count"`
	Min       float64   `json:"min,omitempty"`
	Max       float64   `json:"max,omitempty"`
}

// TimeSeriesPoint 时间序列数据点
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// TimeSeriesResponse 时间序列响应
type TimeSeriesResponse struct {
	DeviceID string                         `json:"device_id"`
	Metrics  map[string][]TimeSeriesPoint   `json:"metrics"`
}


// PushMetricsRequest 推送数据请求结构（设计文档定义）
type PushMetricsRequest struct {
	DeviceKey string            `json:"device_key"` // 设备 IP 作为标识
	Metrics   []PushMetricPoint `json:"metrics"`
}

// PushMetricPoint 推送指标点
type PushMetricPoint struct {
	Timestamp  int64                       `json:"ts"`         // Unix 毫秒时间戳
	Interfaces map[string]InterfaceMetrics `json:"interfaces"` // 接口带宽数据
	Pings      []PingMetric                `json:"pings"`      // Ping 数据
}

// InterfaceMetrics 接口带宽指标
type InterfaceMetrics struct {
	RxRate int64 `json:"rx_rate"` // 接收速率 bps
	TxRate int64 `json:"tx_rate"` // 发送速率 bps
}

// PingMetric Ping 指标
type PingMetric struct {
	Target    string `json:"target"`     // 目标地址
	SrcIface  string `json:"src_iface"`  // 源接口
	Latency   int64  `json:"latency"`    // 延迟 ms，0 表示丢包
	Status    string `json:"status"`     // up/down
}

// BandwidthPushRequest 带宽数据推送请求
type BandwidthPushRequest struct {
	DeviceKey  string                      `json:"device_key"`
	Timestamp  int64                       `json:"ts"`
	Interfaces map[string]InterfaceMetrics `json:"interfaces"`
}

// PingPushRequest Ping 数据推送请求
type PingPushRequest struct {
	DeviceKey string       `json:"device_key"`
	Timestamp int64        `json:"ts"`
	Pings     []PingMetric `json:"pings"`
}

// DeviceOnlineStatus 设备在线状态
type DeviceOnlineStatus struct {
	DeviceID   uint       `json:"device_id"`
	DeviceName string     `json:"device_name"`
	DeviceIP   string     `json:"device_ip"`
	Status     string     `json:"status"`      // online/offline/unknown
	LastSeen   *time.Time `json:"last_seen"`
	LastPushAt *time.Time `json:"last_push_at"`
}
