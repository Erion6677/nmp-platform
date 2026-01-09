package api

import (
	"fmt"
	"net/http"
	"nmp-platform/internal/models"
	"nmp-platform/internal/service"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// 最大查询时间范围（24小时）
const MaxQueryTimeRange = 24 * time.Hour

// DataQueryHandler 数据查询处理器
type DataQueryHandler struct {
	queryService *service.DataQueryService
}

// NewDataQueryHandler 创建数据查询处理器实例
func NewDataQueryHandler(queryService *service.DataQueryService) *DataQueryHandler {
	return &DataQueryHandler{
		queryService: queryService,
	}
}

// QueryRealTimeData 查询实时数据
// @Summary 查询实时数据
// @Description 从Redis缓存查询设备的实时监控数据
// @Tags 数据查询
// @Accept json
// @Produce json
// @Param device_ids query string true "设备ID列表，逗号分隔"
// @Param metrics query string false "指标列表，逗号分隔，为空则返回所有指标"
// @Success 200 {object} service.RealTimeQueryResponse "实时数据"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/query/realtime [get]
func (h *DataQueryHandler) QueryRealTimeData(c *gin.Context) {
	deviceIDsStr := c.Query("device_ids")
	if deviceIDsStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "device_ids parameter is required",
		})
		return
	}

	deviceIDs := strings.Split(deviceIDsStr, ",")
	for i, id := range deviceIDs {
		deviceIDs[i] = strings.TrimSpace(id)
	}

	var metrics []string
	metricsStr := c.Query("metrics")
	if metricsStr != "" {
		metrics = strings.Split(metricsStr, ",")
		for i, metric := range metrics {
			metrics[i] = strings.TrimSpace(metric)
		}
	}

	req := &service.RealTimeQueryRequest{
		DeviceIDs: deviceIDs,
		Metrics:   metrics,
	}

	response, err := h.queryService.QueryRealTimeData(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query real-time data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// QueryHistoricalData 查询历史数据
// @Summary 查询历史数据
// @Description 从InfluxDB查询设备的历史监控数据
// @Tags 数据查询
// @Accept json
// @Produce json
// @Param request body service.QueryRequest true "查询请求"
// @Success 200 {object} service.QueryResponse "历史数据"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/query/historical [post]
func (h *DataQueryHandler) QueryHistoricalData(c *gin.Context) {
	var req service.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.queryService.QueryHistoricalData(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query historical data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// QueryHistoricalDataByParams 通过URL参数查询历史数据
// @Summary 通过参数查询历史数据
// @Description 通过URL参数查询设备的历史监控数据
// @Tags 数据查询
// @Accept json
// @Produce json
// @Param device_id query string true "设备ID"
// @Param metrics query string false "指标列表，逗号分隔"
// @Param start_time query string false "开始时间 (RFC3339格式)"
// @Param end_time query string false "结束时间 (RFC3339格式)"
// @Param granularity query string false "数据粒度 (raw, 1m, 5m, 15m, 1h, 6h, 1d)"
// @Param limit query int false "限制返回的数据点数量"
// @Param page query int false "分页页码"
// @Success 200 {object} service.QueryResponse "历史数据"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/query/historical/{device_id} [get]
func (h *DataQueryHandler) QueryHistoricalDataByParams(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		deviceID = c.Query("device_id")
	}
	
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "device_id is required",
		})
		return
	}

	req := service.QueryRequest{
		DeviceID: deviceID,
	}

	// 解析指标列表
	if metricsStr := c.Query("metrics"); metricsStr != "" {
		req.Metrics = strings.Split(metricsStr, ",")
		for i, metric := range req.Metrics {
			req.Metrics[i] = strings.TrimSpace(metric)
		}
	}

	// 解析时间范围
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		} else {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Invalid start_time format, use RFC3339",
				Details: err.Error(),
			})
			return
		}
	}

	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		} else {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Invalid end_time format, use RFC3339",
				Details: err.Error(),
			})
			return
		}
	}

	// 解析数据粒度
	req.Granularity = c.Query("granularity")

	// 解析分页参数
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = limit
		}
	}

	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			req.Page = page
		}
	}

	response, err := h.queryService.QueryHistoricalData(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query historical data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// QueryAggregatedData 查询聚合数据
// @Summary 查询聚合数据
// @Description 查询设备的聚合监控数据，自动应用合适的聚合粒度
// @Tags 数据查询
// @Accept json
// @Produce json
// @Param request body service.QueryRequest true "查询请求"
// @Success 200 {object} service.QueryResponse "聚合数据"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/query/aggregated [post]
func (h *DataQueryHandler) QueryAggregatedData(c *gin.Context) {
	var req service.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.queryService.QueryAggregatedData(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query aggregated data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// QueryTimeSeriesData 查询时间序列数据
// @Summary 查询时间序列数据
// @Description 查询设备的时间序列数据，适用于图表展示
// @Tags 数据查询
// @Accept json
// @Produce json
// @Param request body service.QueryRequest true "查询请求"
// @Success 200 {object} models.TimeSeriesResponse "时间序列数据"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/query/timeseries [post]
func (h *DataQueryHandler) QueryTimeSeriesData(c *gin.Context) {
	var req service.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	response, err := h.queryService.QueryTimeSeriesData(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query time series data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// QueryMetricSummary 查询指标摘要
// @Summary 查询指标摘要
// @Description 查询指定设备和指标的摘要统计信息
// @Tags 数据查询
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Param metric path string true "指标名称"
// @Param start_time query string true "开始时间 (RFC3339格式)"
// @Param end_time query string true "结束时间 (RFC3339格式)"
// @Success 200 {object} models.MetricSummary "指标摘要"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/query/summary/{device_id}/{metric} [get]
func (h *DataQueryHandler) QueryMetricSummary(c *gin.Context) {
	deviceID := c.Param("device_id")
	metric := c.Param("metric")

	if deviceID == "" || metric == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "device_id and metric are required",
		})
		return
	}

	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "start_time and end_time are required",
		})
		return
	}

	startTime, err := time.Parse(time.RFC3339, startTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid start_time format, use RFC3339",
			Details: err.Error(),
		})
		return
	}

	endTime, err := time.Parse(time.RFC3339, endTimeStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid end_time format, use RFC3339",
			Details: err.Error(),
		})
		return
	}

	summary, err := h.queryService.QueryMetricSummary(c.Request.Context(), deviceID, metric, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query metric summary",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// QueryDeviceStatus 查询设备状态列表
// @Summary 查询设备状态
// @Description 查询多个设备的在线状态和最后活跃时间
// @Tags 数据查询
// @Accept json
// @Produce json
// @Param device_ids query string true "设备ID列表，逗号分隔"
// @Success 200 {object} []models.DeviceStatusInfo "设备状态列表"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/query/device-status [get]
func (h *DataQueryHandler) QueryDeviceStatus(c *gin.Context) {
	deviceIDsStr := c.Query("device_ids")
	if deviceIDsStr == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "device_ids parameter is required",
		})
		return
	}

	deviceIDs := strings.Split(deviceIDsStr, ",")
	for i, id := range deviceIDs {
		deviceIDs[i] = strings.TrimSpace(id)
	}

	statuses, err := h.queryService.QueryDeviceList(c.Request.Context(), deviceIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query device status",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, statuses)
}

// QueryLatestMetric 查询最新指标值
// @Summary 查询最新指标值
// @Description 查询设备指定指标的最新值
// @Tags 数据查询
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Param metric path string true "指标名称"
// @Success 200 {object} map[string]interface{} "最新指标值"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 404 {object} models.ErrorResponse "数据不存在"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/query/latest/{device_id}/{metric} [get]
func (h *DataQueryHandler) QueryLatestMetric(c *gin.Context) {
	deviceID := c.Param("device_id")
	metric := c.Param("metric")

	if deviceID == "" || metric == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "device_id and metric are required",
		})
		return
	}

	// 这里应该调用数据接收服务的方法
	// 为了简化，我们直接使用实时查询
	req := &service.RealTimeQueryRequest{
		DeviceIDs: []string{deviceID},
		Metrics:   []string{metric},
	}

	response, err := h.queryService.QueryRealTimeData(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query latest metric",
			Details: err.Error(),
		})
		return
	}

	if len(response.Devices) == 0 {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Device data not found",
		})
		return
	}

	deviceData := response.Devices[0]
	if value, exists := deviceData.Metrics[metric]; exists {
		c.JSON(http.StatusOK, gin.H{
			"device_id": deviceID,
			"metric":    metric,
			"value":     value,
			"timestamp": deviceData.Timestamp,
		})
	} else {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Metric not found",
		})
	}
}

// RegisterRoutes 注册数据查询相关路由
func (h *DataQueryHandler) RegisterRoutes(router *gin.RouterGroup) {
	queryGroup := router.Group("/query")
	{
		// 实时数据查询
		queryGroup.GET("/realtime", h.QueryRealTimeData)
		queryGroup.GET("/latest/:device_id/:metric", h.QueryLatestMetric)
		queryGroup.GET("/device-status", h.QueryDeviceStatus)

		// 历史数据查询
		queryGroup.POST("/historical", h.QueryHistoricalData)
		queryGroup.GET("/historical/:device_id", h.QueryHistoricalDataByParams)
		queryGroup.POST("/aggregated", h.QueryAggregatedData)
		queryGroup.POST("/timeseries", h.QueryTimeSeriesData)

		// 统计和摘要
		queryGroup.GET("/summary/:device_id/:metric", h.QueryMetricSummary)
	}

	// 监控指标查询路由
	metricsGroup := router.Group("/metrics")
	{
		// 总流量查询（聚合所有设备）- 必须放在参数路由之前
		metricsGroup.GET("/traffic/total", h.QueryTotalTraffic)
		// 带宽数据查询
		metricsGroup.GET("/bandwidth/:device_id", h.QueryBandwidthData)
		// Ping 数据查询
		metricsGroup.GET("/ping/:device_id", h.QueryPingData)
	}
}

// BandwidthQueryResponse 带宽查询响应
type BandwidthQueryResponse struct {
	DeviceID   string                   `json:"device_id"`
	StartTime  time.Time                `json:"start_time"`
	EndTime    time.Time                `json:"end_time"`
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
	DeviceID  string                 `json:"device_id"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Targets   map[string][]PingPoint `json:"targets"`
}

// PingPoint Ping 数据点
type PingPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Latency   float64   `json:"latency"`   // 延迟 ms
	Status    string    `json:"status"`    // up/down
	IsLoss    bool      `json:"is_loss"`   // 是否丢包
}

// QueryBandwidthData 查询带宽数据
// @Summary 查询带宽数据
// @Description 按设备和接口查询带宽历史数据
// @Tags 监控指标
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Param interface query string false "接口名称，为空则查询所有接口"
// @Param start_time query string false "开始时间 (RFC3339格式)"
// @Param end_time query string false "结束时间 (RFC3339格式)"
// @Param range query string false "时间范围 (1h, 6h, 12h, 24h)"
// @Success 200 {object} BandwidthQueryResponse "带宽数据"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/metrics/bandwidth/{device_id} [get]
func (h *DataQueryHandler) QueryBandwidthData(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "device_id is required",
		})
		return
	}

	// 解析时间范围
	startTime, endTime, err := h.parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// 验证时间范围不超过24小时
	if err := h.validateTimeRange(startTime, endTime); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// 获取接口过滤参数
	interfaceName := c.Query("interface")

	// 查询带宽数据
	response, err := h.queryService.QueryBandwidthData(c.Request.Context(), deviceID, interfaceName, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query bandwidth data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// QueryPingData 查询 Ping 数据
// @Summary 查询 Ping 数据
// @Description 按设备和目标查询 Ping 历史数据，丢包点位会被标记
// @Tags 监控指标
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Param target query string false "目标地址，为空则查询所有目标"
// @Param start_time query string false "开始时间 (RFC3339格式)"
// @Param end_time query string false "结束时间 (RFC3339格式)"
// @Param range query string false "时间范围 (1h, 6h, 12h, 24h)"
// @Success 200 {object} PingQueryResponse "Ping 数据"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/metrics/ping/{device_id} [get]
func (h *DataQueryHandler) QueryPingData(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "device_id is required",
		})
		return
	}

	// 解析时间范围
	startTime, endTime, err := h.parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// 验证时间范围不超过24小时
	if err := h.validateTimeRange(startTime, endTime); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// 获取目标过滤参数
	targetAddress := c.Query("target")

	// 查询 Ping 数据
	response, err := h.queryService.QueryPingData(c.Request.Context(), deviceID, targetAddress, startTime, endTime)
	if err != nil {
		// 记录详细错误日志
		fmt.Printf("QueryPingData error for device %s: %v\n", deviceID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query ping data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// parseTimeRange 解析时间范围参数
func (h *DataQueryHandler) parseTimeRange(c *gin.Context) (time.Time, time.Time, error) {
	var startTime, endTime time.Time
	now := time.Now()

	// 首先检查是否有预设的时间范围
	rangeStr := c.Query("range")
	if rangeStr != "" {
		var duration time.Duration
		switch rangeStr {
		case "10m":
			duration = 10 * time.Minute
		case "30m":
			duration = 30 * time.Minute
		case "1h":
			duration = 1 * time.Hour
		case "3h":
			duration = 3 * time.Hour
		case "6h":
			duration = 6 * time.Hour
		case "12h":
			duration = 12 * time.Hour
		case "24h":
			duration = 24 * time.Hour
		default:
			return startTime, endTime, fmt.Errorf("invalid range value, must be one of: 10m, 30m, 1h, 3h, 6h, 12h, 24h")
		}
		startTime = now.Add(-duration)
		endTime = now
		return startTime, endTime, nil
	}

	// 解析自定义时间范围
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr != "" {
		var err error
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return startTime, endTime, fmt.Errorf("invalid start_time format, use RFC3339")
		}
	} else {
		// 默认查询最近12小时
		startTime = now.Add(-12 * time.Hour)
	}

	if endTimeStr != "" {
		var err error
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return startTime, endTime, fmt.Errorf("invalid end_time format, use RFC3339")
		}
	} else {
		endTime = now
	}

	return startTime, endTime, nil
}

// validateTimeRange 验证时间范围不超过24小时
func (h *DataQueryHandler) validateTimeRange(startTime, endTime time.Time) error {
	if startTime.After(endTime) {
		return fmt.Errorf("start_time must be before end_time")
	}

	duration := endTime.Sub(startTime)
	if duration > MaxQueryTimeRange {
		return fmt.Errorf("time range cannot exceed 24 hours, current range: %v", duration)
	}

	return nil
}

// TotalTrafficResponse 总流量响应
type TotalTrafficResponse struct {
	StartTime time.Time             `json:"start_time"`
	EndTime   time.Time             `json:"end_time"`
	Points    []TotalTrafficPoint   `json:"points"`
}

// TotalTrafficPoint 总流量数据点
type TotalTrafficPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Inbound   float64   `json:"inbound"`  // 总入站流量 bps
	Outbound  float64   `json:"outbound"` // 总出站流量 bps
}

// QueryTotalTraffic 查询总流量数据
// @Summary 查询总流量数据
// @Description 查询所有设备的聚合流量数据，用于概览页展示
// @Tags 监控指标
// @Accept json
// @Produce json
// @Param start_time query string false "开始时间 (RFC3339格式)"
// @Param end_time query string false "结束时间 (RFC3339格式)"
// @Param range query string false "时间范围 (10m, 30m, 1h, 3h, 6h, 12h, 24h)"
// @Success 200 {object} TotalTrafficResponse "总流量数据"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/metrics/traffic/total [get]
func (h *DataQueryHandler) QueryTotalTraffic(c *gin.Context) {
	// 解析时间范围
	startTime, endTime, err := h.parseTimeRange(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// 验证时间范围不超过24小时
	if err := h.validateTimeRange(startTime, endTime); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: err.Error(),
		})
		return
	}

	// 查询总流量数据
	response, err := h.queryService.QueryTotalTraffic(c.Request.Context(), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to query total traffic data",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
	})
}