package service

import (
	"context"
	"fmt"
	"log"
	"nmp-platform/internal/models"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DataQueryService 数据查询服务
type DataQueryService struct {
	influxClient InfluxClient
	redisClient  RedisClient
}

// NewDataQueryService 创建数据查询服务实例
func NewDataQueryService(influxClient InfluxClient, redisClient RedisClient) *DataQueryService {
	return &DataQueryService{
		influxClient: influxClient,
		redisClient:  redisClient,
	}
}

// QueryRequest 数据查询请求
type QueryRequest struct {
	DeviceID    string            `json:"device_id" binding:"required"`
	Metrics     []string          `json:"metrics,omitempty"`     // 指定查询的指标，为空则查询所有
	StartTime   *time.Time        `json:"start_time,omitempty"`  // 开始时间，为空则查询实时数据
	EndTime     *time.Time        `json:"end_time,omitempty"`    // 结束时间，为空则到当前时间
	Granularity string            `json:"granularity,omitempty"` // 数据粒度：raw, 1m, 5m, 1h, 1d
	Limit       int               `json:"limit,omitempty"`       // 限制返回的数据点数量
	Page        int               `json:"page,omitempty"`        // 分页页码
	Tags        map[string]string `json:"tags,omitempty"`        // 额外的标签过滤
}

// QueryResponse 数据查询响应
type QueryResponse struct {
	DeviceID   string                   `json:"device_id"`
	Metrics    []string                 `json:"metrics"`
	DataPoints []models.DataPoint       `json:"data_points"`
	Pagination *models.PaginationInfo   `json:"pagination,omitempty"`
	Summary    *models.QuerySummary     `json:"summary"`
}

// RealTimeQueryRequest 实时数据查询请求
type RealTimeQueryRequest struct {
	DeviceIDs []string `json:"device_ids" binding:"required"`
	Metrics   []string `json:"metrics,omitempty"`
}

// RealTimeQueryResponse 实时数据查询响应
type RealTimeQueryResponse struct {
	Devices   []models.DeviceRealTimeData `json:"devices"`
	Timestamp time.Time                   `json:"timestamp"`
}

// QueryRealTimeData 查询实时数据（从Redis）
func (s *DataQueryService) QueryRealTimeData(ctx context.Context, req *RealTimeQueryRequest) (*RealTimeQueryResponse, error) {
	response := &RealTimeQueryResponse{
		Devices:   make([]models.DeviceRealTimeData, 0, len(req.DeviceIDs)),
		Timestamp: time.Now(),
	}

	for _, deviceID := range req.DeviceIDs {
		deviceData, err := s.getDeviceRealTimeData(ctx, deviceID, req.Metrics)
		if err != nil {
			log.Printf("Failed to get real-time data for device %s: %v", deviceID, err)
			// 继续处理其他设备，不因为单个设备失败而中断
			continue
		}

		if deviceData != nil {
			response.Devices = append(response.Devices, *deviceData)
		}
	}

	return response, nil
}

// getDeviceRealTimeData 获取单个设备的实时数据
func (s *DataQueryService) getDeviceRealTimeData(ctx context.Context, deviceID string, metrics []string) (*models.DeviceRealTimeData, error) {
	// 获取设备最新数据
	latestKey := fmt.Sprintf("device:latest:%s", deviceID)
	var latestData models.MetricData
	
	err := s.redisClient.GetJSON(ctx, latestKey, &latestData)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest data: %w", err)
	}

	// 获取设备状态
	status, err := s.getDeviceStatus(ctx, deviceID)
	if err != nil {
		log.Printf("Failed to get device status: %v", err)
		status = "unknown"
	}

	deviceData := &models.DeviceRealTimeData{
		DeviceID:  deviceID,
		Status:    status,
		Timestamp: latestData.Timestamp,
		Metrics:   make(map[string]interface{}),
	}

	// 如果指定了特定指标，只返回这些指标
	if len(metrics) > 0 {
		for _, metric := range metrics {
			if value, exists := latestData.Metrics[metric]; exists {
				deviceData.Metrics[metric] = value
			}
		}
	} else {
		// 返回所有指标
		deviceData.Metrics = latestData.Metrics
	}

	return deviceData, nil
}

// QueryHistoricalData 查询历史数据（从InfluxDB）
func (s *DataQueryService) QueryHistoricalData(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	// 验证请求参数
	if err := s.validateQueryRequest(req); err != nil {
		return nil, fmt.Errorf("invalid query request: %w", err)
	}

	// 构建InfluxDB查询
	query, err := s.buildInfluxQuery(req)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// 执行查询
	result, err := s.influxClient.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	// 解析查询结果
	dataPoints, err := s.parseQueryResult(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query result: %w", err)
	}

	// 应用分页
	paginatedData, pagination := s.applyPagination(dataPoints, req.Page, req.Limit)

	// 构建响应
	response := &QueryResponse{
		DeviceID:   req.DeviceID,
		Metrics:    req.Metrics,
		DataPoints: paginatedData,
		Pagination: pagination,
		Summary: &models.QuerySummary{
			TotalPoints:   len(dataPoints),
			StartTime:     req.StartTime,
			EndTime:       req.EndTime,
			Granularity:   req.Granularity,
			QueryDuration: time.Since(time.Now()).String(),
		},
	}

	return response, nil
}

// validateQueryRequest 验证查询请求
func (s *DataQueryService) validateQueryRequest(req *QueryRequest) error {
	if req.DeviceID == "" {
		return models.NewValidationError("device_id is required")
	}

	// 验证时间范围
	if req.StartTime != nil && req.EndTime != nil {
		if req.StartTime.After(*req.EndTime) {
			return models.NewValidationError("start_time must be before end_time")
		}
	}

	// 验证数据粒度
	validGranularities := []string{"raw", "1m", "5m", "15m", "1h", "6h", "1d"}
	if req.Granularity != "" {
		valid := false
		for _, g := range validGranularities {
			if req.Granularity == g {
				valid = true
				break
			}
		}
		if !valid {
			return models.NewValidationError(fmt.Sprintf("invalid granularity, must be one of: %s", strings.Join(validGranularities, ", ")))
		}
	}

	// 验证分页参数
	if req.Page < 0 {
		return models.NewValidationError("page must be non-negative")
	}
	if req.Limit < 0 {
		return models.NewValidationError("limit must be non-negative")
	}
	if req.Limit > 10000 {
		return models.NewValidationError("limit cannot exceed 10000")
	}

	return nil
}

// buildInfluxQuery 构建InfluxDB查询语句
func (s *DataQueryService) buildInfluxQuery(req *QueryRequest) (string, error) {
	// 设置默认时间范围
	startTime := time.Now().Add(-24 * time.Hour) // 默认查询最近24小时
	if req.StartTime != nil {
		startTime = *req.StartTime
	}

	endTime := time.Now()
	if req.EndTime != nil {
		endTime = *req.EndTime
	}

	// 基础查询
	query := fmt.Sprintf(`
		from(bucket: "monitoring")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "device_metrics")
		|> filter(fn: (r) => r.device_id == "%s")`,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		req.DeviceID,
	)

	// 添加指标过滤
	if len(req.Metrics) > 0 {
		metricFilters := make([]string, len(req.Metrics))
		for i, metric := range req.Metrics {
			metricFilters[i] = fmt.Sprintf(`r._field == "%s"`, metric)
		}
		query += fmt.Sprintf(`|> filter(fn: (r) => %s)`, strings.Join(metricFilters, " or "))
	}

	// 添加标签过滤
	for key, value := range req.Tags {
		query += fmt.Sprintf(`|> filter(fn: (r) => r.%s == "%s")`, key, value)
	}

	// 添加数据聚合（如果指定了粒度）
	if req.Granularity != "" && req.Granularity != "raw" {
		aggregateWindow := s.getAggregateWindow(req.Granularity)
		query += fmt.Sprintf(`|> aggregateWindow(every: %s, fn: mean, createEmpty: false)`, aggregateWindow)
	}

	// 添加排序
	query += `|> sort(columns: ["_time"])`

	// 添加限制（如果指定）
	if req.Limit > 0 {
		offset := req.Page * req.Limit
		query += fmt.Sprintf(`|> limit(n: %d, offset: %d)`, req.Limit, offset)
	}

	return query, nil
}

// getAggregateWindow 获取聚合窗口大小
func (s *DataQueryService) getAggregateWindow(granularity string) string {
	switch granularity {
	case "1m":
		return "1m"
	case "5m":
		return "5m"
	case "15m":
		return "15m"
	case "1h":
		return "1h"
	case "6h":
		return "6h"
	case "1d":
		return "1d"
	default:
		return "1m"
	}
}

// getAutoAggregateWindow 根据时间范围自动选择聚合粒度
// 监控系统聚合策略：
// - 12小时内：不聚合，保持原始数据（秒级精度）
// - 24小时内：每2秒聚合一次
// - 48小时内：每10秒聚合一次
// - 72小时内：每30秒聚合一次
// - 超过72小时：每1分钟聚合一次
func (s *DataQueryService) getAutoAggregateWindow(duration time.Duration) string {
	switch {
	case duration <= 12*time.Hour:
		// 12小时内，不聚合，保持原始数据
		return ""
	case duration <= 24*time.Hour:
		// 24小时内，每2秒聚合一次
		return "2s"
	case duration <= 48*time.Hour:
		// 48小时内，每10秒聚合一次
		return "10s"
	case duration <= 72*time.Hour:
		// 72小时内，每30秒聚合一次
		return "30s"
	default:
		// 超过72小时，每1分钟聚合一次
		return "1m"
	}
}

// parseQueryResult 解析InfluxDB查询结果
func (s *DataQueryService) parseQueryResult(result QueryResult) ([]models.DataPoint, error) {
	var dataPoints []models.DataPoint
	pointsMap := make(map[time.Time]*models.DataPoint)

	for result.Next() {
		record := result.Record()
		
		timestamp := record.Time()
		field := record.Field()
		value := record.Value()

		// 获取或创建数据点
		point, exists := pointsMap[timestamp]
		if !exists {
			point = &models.DataPoint{
				Timestamp: timestamp,
				Values:    make(map[string]interface{}),
			}
			pointsMap[timestamp] = point
		}

		// 添加字段值
		point.Values[field] = value
	}

	// 检查查询错误
	if result.Err() != nil {
		return nil, fmt.Errorf("query execution error: %w", result.Err())
	}

	// 转换为切片并排序
	for _, point := range pointsMap {
		dataPoints = append(dataPoints, *point)
	}

	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	return dataPoints, nil
}

// applyPagination 应用分页
func (s *DataQueryService) applyPagination(dataPoints []models.DataPoint, page, limit int) ([]models.DataPoint, *models.PaginationInfo) {
	total := len(dataPoints)
	
	// 如果没有指定限制，返回所有数据
	if limit <= 0 {
		return dataPoints, &models.PaginationInfo{
			Page:       0,
			Limit:      total,
			Total:      total,
			TotalPages: 1,
		}
	}

	// 计算分页信息
	totalPages := (total + limit - 1) / limit
	if page < 0 {
		page = 0
	}
	if page >= totalPages {
		page = totalPages - 1
	}

	start := page * limit
	end := start + limit
	if end > total {
		end = total
	}

	paginatedData := dataPoints[start:end]

	pagination := &models.PaginationInfo{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}

	return paginatedData, pagination
}

// getDeviceStatus 获取设备状态
func (s *DataQueryService) getDeviceStatus(ctx context.Context, deviceID string) (string, error) {
	key := fmt.Sprintf("device:last_seen:%s", deviceID)
	
	lastSeenStr, err := s.redisClient.Get(ctx, key)
	if err != nil {
		return "unknown", nil
	}

	lastSeen, err := time.Parse(time.RFC3339, lastSeenStr)
	if err != nil {
		return "unknown", nil
	}

	// 如果超过10分钟没有数据，认为离线
	if time.Since(lastSeen) > 10*time.Minute {
		return "offline", nil
	}

	return "online", nil
}

// QueryAggregatedData 查询聚合数据
func (s *DataQueryService) QueryAggregatedData(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	// 强制设置聚合粒度
	if req.Granularity == "" || req.Granularity == "raw" {
		req.Granularity = "1h" // 默认按小时聚合
	}

	return s.QueryHistoricalData(ctx, req)
}

// QueryMetricSummary 查询指标摘要信息
func (s *DataQueryService) QueryMetricSummary(ctx context.Context, deviceID string, metric string, startTime, endTime time.Time) (*models.MetricSummary, error) {
	query := fmt.Sprintf(`
		from(bucket: "monitoring")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "device_metrics")
		|> filter(fn: (r) => r.device_id == "%s")
		|> filter(fn: (r) => r._field == "%s")
		|> group()
		|> aggregateWindow(every: inf, fn: mean, createEmpty: false)
		|> yield(name: "mean")`,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		deviceID,
		metric,
	)

	result, err := s.influxClient.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute summary query: %w", err)
	}

	summary := &models.MetricSummary{
		DeviceID:  deviceID,
		Metric:    metric,
		StartTime: startTime,
		EndTime:   endTime,
	}

	// 解析结果
	for result.Next() {
		record := result.Record()
		if record.Field() == metric {
			if value, ok := record.Value().(float64); ok {
				summary.Average = value
				summary.Count++
			}
		}
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("summary query execution error: %w", result.Err())
	}

	return summary, nil
}

// QueryDeviceList 查询设备列表及其最新状态
func (s *DataQueryService) QueryDeviceList(ctx context.Context, deviceIDs []string) ([]models.DeviceStatusInfo, error) {
	var deviceStatuses []models.DeviceStatusInfo

	for _, deviceID := range deviceIDs {
		status, err := s.getDeviceStatus(ctx, deviceID)
		if err != nil {
			log.Printf("Failed to get status for device %s: %v", deviceID, err)
			status = "unknown"
		}

		// 获取最后活跃时间
		lastSeenKey := fmt.Sprintf("device:last_seen:%s", deviceID)
		lastSeenStr, err := s.redisClient.Get(ctx, lastSeenKey)
		var lastSeen *time.Time
		if err == nil {
			if t, parseErr := time.Parse(time.RFC3339, lastSeenStr); parseErr == nil {
				lastSeen = &t
			}
		}

		deviceStatus := models.DeviceStatusInfo{
			DeviceID: deviceID,
			Status:   status,
			LastSeen: lastSeen,
		}

		deviceStatuses = append(deviceStatuses, deviceStatus)
	}

	return deviceStatuses, nil
}

// QueryTimeSeriesData 查询时间序列数据（用于图表展示）
func (s *DataQueryService) QueryTimeSeriesData(ctx context.Context, req *QueryRequest) (*models.TimeSeriesResponse, error) {
	// 查询历史数据
	queryResp, err := s.QueryHistoricalData(ctx, req)
	if err != nil {
		return nil, err
	}

	// 转换为时间序列格式
	timeSeries := &models.TimeSeriesResponse{
		DeviceID: req.DeviceID,
		Metrics:  make(map[string][]models.TimeSeriesPoint),
	}

	// 按指标组织数据
	for _, point := range queryResp.DataPoints {
		for metric, value := range point.Values {
			if timeSeries.Metrics[metric] == nil {
				timeSeries.Metrics[metric] = make([]models.TimeSeriesPoint, 0)
			}

			// 转换值为数字类型
			var numValue float64
			switch v := value.(type) {
			case float64:
				numValue = v
			case int64:
				numValue = float64(v)
			case string:
				if parsed, parseErr := strconv.ParseFloat(v, 64); parseErr == nil {
					numValue = parsed
				}
			default:
				continue // 跳过无法转换的值
			}

			tsPoint := models.TimeSeriesPoint{
				Timestamp: point.Timestamp,
				Value:     numValue,
			}

			timeSeries.Metrics[metric] = append(timeSeries.Metrics[metric], tsPoint)
		}
	}

	return timeSeries, nil
}

// BandwidthQueryResponse 带宽查询响应
type BandwidthQueryResponse struct {
	DeviceID   string                      `json:"device_id"`
	StartTime  time.Time                   `json:"start_time"`
	EndTime    time.Time                   `json:"end_time"`
	Interfaces map[string][]BandwidthPoint `json:"interfaces"`
}

// BandwidthPoint 带宽数据点
type BandwidthPoint struct {
	Timestamp time.Time `json:"timestamp"`
	RxRate    float64   `json:"rx_rate"` // 接收速率 bps
	TxRate    float64   `json:"tx_rate"` // 发送速率 bps
}

// PingQueryResponse Ping 查询响应
type PingQueryResponse struct {
	DeviceID  string                    `json:"device_id"`
	StartTime time.Time                 `json:"start_time"`
	EndTime   time.Time                 `json:"end_time"`
	Targets   map[string][]PingPoint    `json:"targets"`
	Stats     map[string]*PingStats     `json:"stats"` // 每个目标的统计信息
}

// PingPoint Ping 数据点
type PingPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Latency   float64   `json:"latency"`   // 延迟 ms
	Status    string    `json:"status"`    // up/down
	IsLoss    bool      `json:"is_loss"`   // 是否丢包
}

// PingStats Ping 统计信息
type PingStats struct {
	TotalCount int     `json:"total_count"` // 总包数
	LossCount  int     `json:"loss_count"`  // 丢包数
	LossRate   float64 `json:"loss_rate"`   // 丢包率 (%)
	AvgLatency float64 `json:"avg_latency"` // 平均延迟 ms
	MinLatency float64 `json:"min_latency"` // 最小延迟 ms
	MaxLatency float64 `json:"max_latency"` // 最大延迟 ms
}

// QueryBandwidthData 查询带宽数据
func (s *DataQueryService) QueryBandwidthData(ctx context.Context, deviceID string, interfaceName string, startTime, endTime time.Time) (*BandwidthQueryResponse, error) {
	// 根据时间范围自动选择聚合粒度
	duration := endTime.Sub(startTime)
	aggregateWindow := s.getAutoAggregateWindow(duration)
	
	// 构建 InfluxDB 查询
	query := fmt.Sprintf(`
		from(bucket: "monitoring")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "bandwidth")
		|> filter(fn: (r) => r.device_id == "%s")`,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		deviceID,
	)

	// 如果指定了接口名称，添加过滤条件
	if interfaceName != "" {
		query += fmt.Sprintf(`|> filter(fn: (r) => r.interface == "%s")`, interfaceName)
	}

	// 添加聚合（如果需要）
	if aggregateWindow != "" {
		query += fmt.Sprintf(`|> aggregateWindow(every: %s, fn: mean, createEmpty: false)`, aggregateWindow)
	}

	// 添加排序
	query += `|> sort(columns: ["_time"])`

	// 执行查询
	result, err := s.influxClient.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute bandwidth query: %w", err)
	}

	// 解析结果
	response := &BandwidthQueryResponse{
		DeviceID:   deviceID,
		StartTime:  startTime,
		EndTime:    endTime,
		Interfaces: make(map[string][]BandwidthPoint),
	}

	// 临时存储，用于合并同一时间点的 rx_rate 和 tx_rate
	type tempPoint struct {
		RxRate *float64
		TxRate *float64
	}
	interfaceData := make(map[string]map[time.Time]*tempPoint)

	for result.Next() {
		record := result.Record()
		timestamp := record.Time()
		field := record.Field()
		value := record.Value()

		// 获取接口名称（从 tag 中获取）
		ifaceName := ""
		if v := record.ValueByKey("interface"); v != nil {
			ifaceName = fmt.Sprintf("%v", v)
		}
		if ifaceName == "" {
			continue
		}

		// 初始化接口数据映射
		if interfaceData[ifaceName] == nil {
			interfaceData[ifaceName] = make(map[time.Time]*tempPoint)
		}
		if interfaceData[ifaceName][timestamp] == nil {
			interfaceData[ifaceName][timestamp] = &tempPoint{}
		}

		// 转换值为 float64
		var floatValue float64
		switch v := value.(type) {
		case float64:
			floatValue = v
		case int64:
			floatValue = float64(v)
		default:
			continue
		}

		// 根据字段名设置值
		switch field {
		case "rx_rate":
			interfaceData[ifaceName][timestamp].RxRate = &floatValue
		case "tx_rate":
			interfaceData[ifaceName][timestamp].TxRate = &floatValue
		}
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("bandwidth query execution error: %w", result.Err())
	}

	// 转换为响应格式
	for ifaceName, timePoints := range interfaceData {
		points := make([]BandwidthPoint, 0, len(timePoints))
		for ts, tp := range timePoints {
			point := BandwidthPoint{
				Timestamp: ts,
			}
			if tp.RxRate != nil {
				point.RxRate = *tp.RxRate
			}
			if tp.TxRate != nil {
				point.TxRate = *tp.TxRate
			}
			points = append(points, point)
		}
		// 按时间排序
		sort.Slice(points, func(i, j int) bool {
			return points[i].Timestamp.Before(points[j].Timestamp)
		})
		response.Interfaces[ifaceName] = points
	}

	return response, nil
}

// TotalTrafficResponse 总流量响应
type TotalTrafficResponse struct {
	StartTime time.Time           `json:"start_time"`
	EndTime   time.Time           `json:"end_time"`
	Points    []TotalTrafficPoint `json:"points"`
}

// TotalTrafficPoint 总流量数据点
type TotalTrafficPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Inbound   float64   `json:"inbound"`  // 总入站流量 bps
	Outbound  float64   `json:"outbound"` // 总出站流量 bps
}

// QueryTotalTraffic 查询所有设备的总流量（聚合）
func (s *DataQueryService) QueryTotalTraffic(ctx context.Context, startTime, endTime time.Time) (*TotalTrafficResponse, error) {
	// 根据时间范围自动选择聚合粒度
	duration := endTime.Sub(startTime)
	aggregateWindow := s.getTrafficAggregateWindow(duration)

	// 构建 InfluxDB 查询 - 查询所有设备的带宽数据并按时间聚合
	query := fmt.Sprintf(`
		from(bucket: "monitoring")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "bandwidth")
		|> filter(fn: (r) => r._field == "rx_rate" or r._field == "tx_rate")
		|> group(columns: ["_time", "_field"])
		|> aggregateWindow(every: %s, fn: sum, createEmpty: false)
		|> group(columns: ["_field"])
		|> sort(columns: ["_time"])`,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		aggregateWindow,
	)

	// 执行查询
	result, err := s.influxClient.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute total traffic query: %w", err)
	}

	// 解析结果
	response := &TotalTrafficResponse{
		StartTime: startTime,
		EndTime:   endTime,
		Points:    make([]TotalTrafficPoint, 0),
	}

	// 临时存储，用于合并同一时间点的 rx_rate 和 tx_rate
	type tempPoint struct {
		Inbound  *float64
		Outbound *float64
	}
	timePoints := make(map[time.Time]*tempPoint)

	for result.Next() {
		record := result.Record()
		timestamp := record.Time()
		field := record.Field()
		value := record.Value()

		// 初始化时间点
		if timePoints[timestamp] == nil {
			timePoints[timestamp] = &tempPoint{}
		}

		// 转换值为 float64
		var floatValue float64
		switch v := value.(type) {
		case float64:
			floatValue = v
		case int64:
			floatValue = float64(v)
		default:
			continue
		}

		// 根据字段名设置值
		switch field {
		case "rx_rate":
			timePoints[timestamp].Inbound = &floatValue
		case "tx_rate":
			timePoints[timestamp].Outbound = &floatValue
		}
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("total traffic query execution error: %w", result.Err())
	}

	// 转换为响应格式
	for ts, tp := range timePoints {
		point := TotalTrafficPoint{
			Timestamp: ts,
		}
		if tp.Inbound != nil {
			point.Inbound = *tp.Inbound
		}
		if tp.Outbound != nil {
			point.Outbound = *tp.Outbound
		}
		response.Points = append(response.Points, point)
	}

	// 按时间排序
	sort.Slice(response.Points, func(i, j int) bool {
		return response.Points[i].Timestamp.Before(response.Points[j].Timestamp)
	})

	return response, nil
}

// getTrafficAggregateWindow 根据时间范围选择流量聚合粒度
func (s *DataQueryService) getTrafficAggregateWindow(duration time.Duration) string {
	switch {
	case duration <= 30*time.Minute:
		return "30s" // 30分钟内，每30秒聚合
	case duration <= 1*time.Hour:
		return "1m" // 1小时内，每1分钟聚合
	case duration <= 6*time.Hour:
		return "2m" // 6小时内，每2分钟聚合
	case duration <= 12*time.Hour:
		return "5m" // 12小时内，每5分钟聚合
	default:
		return "10m" // 超过12小时，每10分钟聚合
	}
}

// QueryPingData 查询 Ping 数据
// 新逻辑：延迟数据可以聚合，但丢包点必须保留精确时间，统计基于原始数据
func (s *DataQueryService) QueryPingData(ctx context.Context, deviceID string, targetAddress string, startTime, endTime time.Time) (*PingQueryResponse, error) {
	// 根据时间范围自动选择聚合粒度
	duration := endTime.Sub(startTime)
	aggregateWindow := s.getAutoAggregateWindow(duration)
	
	// 构建基础查询条件
	baseFilter := fmt.Sprintf(`
		from(bucket: "monitoring")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "ping")
		|> filter(fn: (r) => r.device_id == "%s")`,
		startTime.Format(time.RFC3339),
		endTime.Format(time.RFC3339),
		deviceID,
	)

	// 如果指定了目标地址，添加过滤条件
	if targetAddress != "" {
		baseFilter += fmt.Sprintf(`|> filter(fn: (r) => r.target_address == "%s")`, targetAddress)
	}

	// 初始化响应
	response := &PingQueryResponse{
		DeviceID:  deviceID,
		StartTime: startTime,
		EndTime:   endTime,
		Targets:   make(map[string][]PingPoint),
		Stats:     make(map[string]*PingStats),
	}

	// 第一步：查询所有原始数据（用于统计和丢包点）
	rawQuery := baseFilter + `|> sort(columns: ["_time"])`
	
	rawResult, err := s.influxClient.Query(rawQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to execute raw ping query: %w", err)
	}

	// 解析原始数据
	type rawPingPoint struct {
		Latency *float64
		Status  string
	}
	rawData := make(map[string]map[time.Time]*rawPingPoint)

	for rawResult.Next() {
		record := rawResult.Record()
		timestamp := record.Time()
		field := record.Field()
		value := record.Value()

		// 获取目标地址和源接口
		targetAddr := ""
		if v := record.ValueByKey("target_address"); v != nil {
			targetAddr = fmt.Sprintf("%v", v)
		}
		if targetAddr == "" {
			continue
		}
		
		srcInterface := ""
		if v := record.ValueByKey("src_interface"); v != nil {
			srcInterface = fmt.Sprintf("%v", v)
		}
		
		target := targetAddr
		if srcInterface != "" {
			target = fmt.Sprintf("%s_%s", targetAddr, srcInterface)
		}

		if rawData[target] == nil {
			rawData[target] = make(map[time.Time]*rawPingPoint)
		}
		if rawData[target][timestamp] == nil {
			rawData[target][timestamp] = &rawPingPoint{}
		}

		switch field {
		case "latency":
			var floatValue float64
			switch v := value.(type) {
			case float64:
				floatValue = v
			case int64:
				floatValue = float64(v)
			}
			rawData[target][timestamp].Latency = &floatValue
		case "status":
			if strValue, ok := value.(string); ok {
				rawData[target][timestamp].Status = strValue
			}
		}
	}

	if rawResult.Err() != nil {
		return nil, fmt.Errorf("raw ping query execution error: %w", rawResult.Err())
	}

	// 计算统计信息并提取丢包点
	lossPoints := make(map[string][]PingPoint) // 存储所有丢包点
	
	for target, timePoints := range rawData {
		stats := &PingStats{
			MinLatency: -1, // 用 -1 表示未初始化
		}
		
		for ts, rp := range timePoints {
			stats.TotalCount++
			
			isLoss := rp.Status == "down" || (rp.Latency != nil && *rp.Latency == 0)
			
			if isLoss {
				stats.LossCount++
				// 保存丢包点
				lossPoints[target] = append(lossPoints[target], PingPoint{
					Timestamp: ts,
					Latency:   0,
					Status:    "down",
					IsLoss:    true,
				})
			} else if rp.Latency != nil {
				latency := *rp.Latency
				stats.AvgLatency += latency
				if stats.MinLatency < 0 || latency < stats.MinLatency {
					stats.MinLatency = latency
				}
				if latency > stats.MaxLatency {
					stats.MaxLatency = latency
				}
			}
		}
		
		// 计算平均延迟和丢包率
		successCount := stats.TotalCount - stats.LossCount
		if successCount > 0 {
			stats.AvgLatency = stats.AvgLatency / float64(successCount)
		}
		if stats.MinLatency < 0 {
			stats.MinLatency = 0
		}
		if stats.TotalCount > 0 {
			stats.LossRate = float64(stats.LossCount) / float64(stats.TotalCount) * 100
		}
		
		response.Stats[target] = stats
	}

	// 第二步：如果需要聚合，查询聚合后的延迟数据
	if aggregateWindow != "" {
		// 只聚合延迟数据（成功的 ping）
		aggQuery := fmt.Sprintf(`%s
			|> filter(fn: (r) => r._field == "latency")
			|> filter(fn: (r) => r._value > 0)
			|> aggregateWindow(every: %s, fn: mean, createEmpty: false)
			|> sort(columns: ["_time"])`,
			baseFilter, aggregateWindow,
		)
		
		aggResult, err := s.influxClient.Query(aggQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to execute aggregated ping query: %w", err)
		}

		// 解析聚合数据
		aggData := make(map[string]map[time.Time]float64)

		for aggResult.Next() {
			record := aggResult.Record()
			timestamp := record.Time()
			value := record.Value()

			targetAddr := ""
			if v := record.ValueByKey("target_address"); v != nil {
				targetAddr = fmt.Sprintf("%v", v)
			}
			if targetAddr == "" {
				continue
			}
			
			srcInterface := ""
			if v := record.ValueByKey("src_interface"); v != nil {
				srcInterface = fmt.Sprintf("%v", v)
			}
			
			target := targetAddr
			if srcInterface != "" {
				target = fmt.Sprintf("%s_%s", targetAddr, srcInterface)
			}

			if aggData[target] == nil {
				aggData[target] = make(map[time.Time]float64)
			}

			var floatValue float64
			switch v := value.(type) {
			case float64:
				floatValue = v
			case int64:
				floatValue = float64(v)
			}
			aggData[target][timestamp] = floatValue
		}

		if aggResult.Err() != nil {
			return nil, fmt.Errorf("aggregated ping query execution error: %w", aggResult.Err())
		}

		// 合并聚合数据和丢包点
		for target, latencies := range aggData {
			points := make([]PingPoint, 0)
			
			// 添加聚合后的延迟点
			for ts, latency := range latencies {
				points = append(points, PingPoint{
					Timestamp: ts,
					Latency:   latency,
					Status:    "up",
					IsLoss:    false,
				})
			}
			
			// 添加所有丢包点（保持原始时间戳）
			if losses, ok := lossPoints[target]; ok {
				points = append(points, losses...)
			}
			
			// 按时间排序
			sort.Slice(points, func(i, j int) bool {
				return points[i].Timestamp.Before(points[j].Timestamp)
			})
			
			response.Targets[target] = points
		}
		
		// 处理只有丢包没有成功数据的目标
		for target, losses := range lossPoints {
			if _, exists := response.Targets[target]; !exists {
				sort.Slice(losses, func(i, j int) bool {
					return losses[i].Timestamp.Before(losses[j].Timestamp)
				})
				response.Targets[target] = losses
			}
		}
	} else {
		// 不需要聚合，直接使用原始数据
		for target, timePoints := range rawData {
			points := make([]PingPoint, 0, len(timePoints))
			for ts, rp := range timePoints {
				point := PingPoint{
					Timestamp: ts,
					Status:    rp.Status,
				}
				if rp.Latency != nil {
					point.Latency = *rp.Latency
				}
				// 标记丢包：latency 为 0 或 status 为 "down" 表示丢包
				point.IsLoss = (rp.Latency != nil && *rp.Latency == 0) || rp.Status == "down"
				points = append(points, point)
			}
			// 按时间排序
			sort.Slice(points, func(i, j int) bool {
				return points[i].Timestamp.Before(points[j].Timestamp)
			})
			response.Targets[target] = points
		}
	}

	return response, nil
}