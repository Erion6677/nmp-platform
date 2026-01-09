package api

import (
	"encoding/json"
	"log"
	"net/http"
	"nmp-platform/internal/models"
	"nmp-platform/internal/service"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// DataReceiverHandler 数据接收处理器
type DataReceiverHandler struct {
	dataReceiverService *service.DataReceiverService
}

// NewDataReceiverHandler 创建数据接收处理器实例
func NewDataReceiverHandler(dataReceiverService *service.DataReceiverService) *DataReceiverHandler {
	return &DataReceiverHandler{
		dataReceiverService: dataReceiverService,
	}
}

// PushData 接收单个设备的监控数据推送
// @Summary 接收监控数据推送
// @Description 接收设备推送的监控数据并存储到InfluxDB和Redis
// @Tags 数据接收
// @Accept json
// @Produce json
// @Param data body models.PushDataRequest true "监控数据"
// @Success 200 {object} models.PushDataResponse "推送成功"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 404 {object} models.ErrorResponse "设备不存在"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/data/push [post]
func (h *DataReceiverHandler) PushData(c *gin.Context) {
	var req models.PushDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// 转换为内部数据结构
	metricData := &models.MetricData{
		DeviceID: req.DeviceID,
		Metrics:  req.Metrics,
		Tags:     req.Tags,
	}

	// 设置时间戳
	if req.Timestamp != nil {
		metricData.Timestamp = *req.Timestamp
	} else {
		metricData.Timestamp = time.Now()
	}

	// 处理数据
	if err := h.dataReceiverService.ReceiveData(c.Request.Context(), metricData); err != nil {
		if validationErr, ok := err.(*models.ValidationError); ok {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Code:    http.StatusBadRequest,
				Message: "Data validation failed",
				Details: validationErr.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to process data",
			Details: err.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, models.PushDataResponse{
		Success:   true,
		Message:   "Data received successfully",
		Timestamp: metricData.Timestamp.Format(time.RFC3339),
	})
}

// PushBatchData 接收批量监控数据推送
// @Summary 批量接收监控数据推送
// @Description 批量接收设备推送的监控数据并存储到InfluxDB和Redis
// @Tags 数据接收
// @Accept json
// @Produce json
// @Param data body models.BatchPushDataRequest true "批量监控数据"
// @Success 200 {object} models.BatchPushDataResponse "批量推送结果"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/data/push/batch [post]
func (h *DataReceiverHandler) PushBatchData(c *gin.Context) {
	var req models.BatchPushDataRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Invalid request format",
			Details: err.Error(),
		})
		return
	}

	// 转换为内部数据结构
	var metricDataList []models.MetricData
	for _, item := range req.Data {
		metricData := models.MetricData{
			DeviceID: item.DeviceID,
			Metrics:  item.Metrics,
			Tags:     item.Tags,
		}

		// 设置时间戳
		if item.Timestamp != nil {
			metricData.Timestamp = *item.Timestamp
		} else {
			metricData.Timestamp = time.Now()
		}

		metricDataList = append(metricDataList, metricData)
	}

	// 处理批量数据
	response, err := h.dataReceiverService.ReceiveBatchData(c.Request.Context(), metricDataList)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to process batch data",
			Details: err.Error(),
		})
		return
	}

	// 根据处理结果返回相应的HTTP状态码
	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusPartialContent
	}

	c.JSON(statusCode, response)
}

// GetLatestData 获取设备最新监控数据
// @Summary 获取设备最新数据
// @Description 从Redis缓存中获取设备的最新监控数据
// @Tags 数据查询
// @Produce json
// @Param device_id path string true "设备ID"
// @Success 200 {object} models.MetricData "最新监控数据"
// @Failure 404 {object} models.ErrorResponse "数据不存在"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/data/latest/{device_id} [get]
func (h *DataReceiverHandler) GetLatestData(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Device ID is required",
		})
		return
	}

	data, err := h.dataReceiverService.GetLatestData(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Latest data not found",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GetMetricValue 获取设备特定指标的最新值
// @Summary 获取特定指标值
// @Description 从Redis缓存中获取设备特定指标的最新值
// @Tags 数据查询
// @Produce json
// @Param device_id path string true "设备ID"
// @Param metric path string true "指标名称"
// @Success 200 {object} map[string]interface{} "指标值"
// @Failure 404 {object} models.ErrorResponse "指标不存在"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/data/metric/{device_id}/{metric} [get]
func (h *DataReceiverHandler) GetMetricValue(c *gin.Context) {
	deviceID := c.Param("device_id")
	metric := c.Param("metric")

	if deviceID == "" || metric == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Device ID and metric name are required",
		})
		return
	}

	value, err := h.dataReceiverService.GetMetricValue(c.Request.Context(), deviceID, metric)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "Metric value not found",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"metric":    metric,
		"value":     value,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// GetDeviceStatus 获取设备在线状态
// @Summary 获取设备状态
// @Description 根据最后数据推送时间判断设备在线状态
// @Tags 数据查询
// @Produce json
// @Param device_id path string true "设备ID"
// @Success 200 {object} map[string]interface{} "设备状态"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/data/status/{device_id} [get]
func (h *DataReceiverHandler) GetDeviceStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "Device ID is required",
		})
		return
	}

	status, err := h.dataReceiverService.GetDeviceStatus(c.Request.Context(), deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get device status",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"device_id": deviceID,
		"status":    status,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// HealthCheck 数据接收服务健康检查
// @Summary 健康检查
// @Description 检查数据接收服务的健康状态
// @Tags 健康检查
// @Produce json
// @Success 200 {object} map[string]interface{} "健康状态"
// @Failure 503 {object} models.ErrorResponse "服务不可用"
// @Router /api/v1/data/health [get]
func (h *DataReceiverHandler) HealthCheck(c *gin.Context) {
	// 这里可以添加更多的健康检查逻辑
	// 比如检查InfluxDB和Redis连接状态
	
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "data-receiver",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// RegisterRoutes 注册数据接收相关路由
func (h *DataReceiverHandler) RegisterRoutes(router *gin.RouterGroup) {
	dataGroup := router.Group("/data")
	{
		// 数据推送接口
		dataGroup.POST("/push", h.PushData)
		dataGroup.POST("/push/batch", h.PushBatchData)
		
		// 数据查询接口
		dataGroup.GET("/latest/:device_id", h.GetLatestData)
		dataGroup.GET("/metric/:device_id/:metric", h.GetMetricValue)
		dataGroup.GET("/status/:device_id", h.GetDeviceStatus)
		dataGroup.GET("/status", h.GetAllDevicesStatus)
		
		// 健康检查
		dataGroup.GET("/health", h.HealthCheck)
	}
	
	// RouterOS设备metrics推送端点（兼容RouterOS格式）
	pushGroup := router.Group("/push")
	{
		pushGroup.POST("/metrics", h.PushRouterOSMetrics)
		pushGroup.POST("/bandwidth", h.PushBandwidthData)
		pushGroup.POST("/ping", h.PushPingData)
	}
}

// PushRouterOSMetrics RouterOS设备metrics推送处理器
// @Summary 接收RouterOS设备推送的metrics数据
// @Description 处理RouterOS设备通过HTTP推送的监控数据
// @Tags 数据接收
// @Accept json
// @Produce json
// @Param data body models.PushMetricsRequest true "RouterOS metrics数据"
// @Success 200 {object} map[string]interface{} "数据接收成功"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 404 {object} models.ErrorResponse "设备不存在"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/push/metrics [post]
func (h *DataReceiverHandler) PushRouterOSMetrics(c *gin.Context) {
	// 获取客户端IP作为设备标识
	clientIP := c.ClientIP()
	
	// 读取原始请求体
	body, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "无法读取请求数据",
			Details: err.Error(),
		})
		return
	}
	
	// 记录接收到的数据（简化日志）
	log.Printf("RouterOS metrics: %s, %d bytes", clientIP, len(body))
	
	// 尝试解析为标准的 PushMetricsRequest 格式
	var req models.PushMetricsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		log.Printf("Parse error for %s: %v", clientIP, err)
		h.handleRawMetrics(c, clientIP, body)
		return
	}
	
	// 使用 device_key 作为设备标识（通常是设备 IP）
	deviceKey := req.DeviceKey
	if deviceKey == "" {
		deviceKey = clientIP
	}
	
	// 验证设备身份
	device, err := h.dataReceiverService.ValidateDeviceByIP(c.Request.Context(), deviceKey)
	if err != nil {
		log.Printf("Device validation failed for %s: %v", deviceKey, err)
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "设备未注册",
			Details: err.Error(),
		})
		return
	}
	
	// 获取采集器配置，用于计算时间间隔
	// 默认采集间隔为 1000ms
	intervalMs := int64(1000)
	
	// 处理每个指标点
	// 如果时间戳为0，需要根据数据点位置倒推时间
	// 假设数据点是按采集顺序排列的，最后一个点是最新的
	processedCount := 0
	now := time.Now().UnixMilli()
	totalPoints := len(req.Metrics)
	
	for i, point := range req.Metrics {
		// 计算时间戳
		var timestamp int64
		if point.Timestamp > 0 {
			// 使用数据点自带的时间戳
			timestamp = point.Timestamp
		} else {
			// 根据位置倒推时间：最后一个点是当前时间，往前每个点减去一个采集间隔
			// 例如：30个点，间隔1秒，第0个点是29秒前，第29个点是当前时间
			offsetMs := int64(totalPoints-1-i) * intervalMs
			timestamp = now - offsetMs
		}
		
		// 处理带宽数据
		if len(point.Interfaces) > 0 {
			if err := h.dataReceiverService.ProcessBandwidthData(c.Request.Context(), device.ID, timestamp, point.Interfaces); err != nil {
				log.Printf("Failed to process bandwidth data for device %d: %v", device.ID, err)
			} else {
				processedCount++
			}
		}
		
		// 处理 Ping 数据
		if len(point.Pings) > 0 {
			if err := h.dataReceiverService.ProcessPingData(c.Request.Context(), device.ID, timestamp, point.Pings); err != nil {
				log.Printf("Failed to process ping data for device %d: %v", device.ID, err)
			} else {
				processedCount++
			}
		}
	}
	
	// 更新设备状态和最后推送时间
	if err := h.dataReceiverService.UpdateDeviceOnlineStatus(c.Request.Context(), device.ID); err != nil {
		log.Printf("Failed to update device status for %d: %v", device.ID, err)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "RouterOS metrics数据接收成功",
		"data": gin.H{
			"device_id":       device.ID,
			"device_key":      deviceKey,
			"timestamp":       time.Now().Unix(),
			"processed_count": processedCount,
		},
	})
}

// handleRawMetrics 处理原始格式的 metrics 数据
func (h *DataReceiverHandler) handleRawMetrics(c *gin.Context, clientIP string, body []byte) {
	// 验证设备身份
	device, err := h.dataReceiverService.ValidateDeviceByIP(c.Request.Context(), clientIP)
	if err != nil {
		log.Printf("Device validation failed for %s: %v", clientIP, err)
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "设备未注册",
			Details: err.Error(),
		})
		return
	}
	
	// 构造标准的监控数据
	now := time.Now()
	metricData := &models.MetricData{
		DeviceID:  strconv.FormatUint(uint64(device.ID), 10),
		Timestamp: now,
		Metrics: map[string]interface{}{
			"raw_data": string(body),
			"source":   "routeros",
			"size":     len(body),
		},
		Tags: map[string]string{
			"device_type": "routeros",
			"client_ip":   clientIP,
			"user_agent":  c.GetHeader("User-Agent"),
		},
	}
	
	// 调用数据接收服务处理数据
	if err := h.dataReceiverService.ReceiveData(c.Request.Context(), metricData); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "数据处理失败",
			Details: err.Error(),
		})
		return
	}
	
	// 更新设备状态
	if err := h.dataReceiverService.UpdateDeviceOnlineStatus(c.Request.Context(), device.ID); err != nil {
		log.Printf("Failed to update device status for %d: %v", device.ID, err)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "RouterOS metrics数据接收成功",
		"data": gin.H{
			"device_id": device.ID,
			"timestamp": now.Unix(),
			"size":      len(body),
		},
	})
}

// PushBandwidthData 接收带宽数据推送
// @Summary 接收带宽数据推送
// @Description 接收设备推送的带宽监控数据
// @Tags 数据接收
// @Accept json
// @Produce json
// @Param data body models.BandwidthPushRequest true "带宽数据"
// @Success 200 {object} map[string]interface{} "数据接收成功"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 404 {object} models.ErrorResponse "设备不存在"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/push/bandwidth [post]
func (h *DataReceiverHandler) PushBandwidthData(c *gin.Context) {
	var req models.BandwidthPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "请求格式错误",
			Details: err.Error(),
		})
		return
	}
	
	// 获取设备标识
	deviceKey := req.DeviceKey
	if deviceKey == "" {
		deviceKey = c.ClientIP()
	}
	
	// 验证设备身份
	device, err := h.dataReceiverService.ValidateDeviceByIP(c.Request.Context(), deviceKey)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "设备未注册",
			Details: err.Error(),
		})
		return
	}
	
	// 处理带宽数据
	timestamp := time.Now()
	if req.Timestamp > 0 {
		timestamp = time.UnixMilli(req.Timestamp)
	}
	
	if err := h.dataReceiverService.ProcessBandwidthData(c.Request.Context(), device.ID, req.Timestamp, req.Interfaces); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "带宽数据处理失败",
			Details: err.Error(),
		})
		return
	}
	
	// 更新设备状态
	if err := h.dataReceiverService.UpdateDeviceOnlineStatus(c.Request.Context(), device.ID); err != nil {
		log.Printf("Failed to update device status for %d: %v", device.ID, err)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "带宽数据接收成功",
		"device_id": device.ID,
		"timestamp": timestamp.Unix(),
	})
}

// PushPingData 接收 Ping 数据推送
// @Summary 接收 Ping 数据推送
// @Description 接收设备推送的 Ping 监控数据
// @Tags 数据接收
// @Accept json
// @Produce json
// @Param data body models.PingPushRequest true "Ping 数据"
// @Success 200 {object} map[string]interface{} "数据接收成功"
// @Failure 400 {object} models.ErrorResponse "请求参数错误"
// @Failure 404 {object} models.ErrorResponse "设备不存在"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/push/ping [post]
func (h *DataReceiverHandler) PushPingData(c *gin.Context) {
	var req models.PingPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Code:    http.StatusBadRequest,
			Message: "请求格式错误",
			Details: err.Error(),
		})
		return
	}
	
	// 获取设备标识
	deviceKey := req.DeviceKey
	if deviceKey == "" {
		deviceKey = c.ClientIP()
	}
	
	// 验证设备身份
	device, err := h.dataReceiverService.ValidateDeviceByIP(c.Request.Context(), deviceKey)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Code:    http.StatusNotFound,
			Message: "设备未注册",
			Details: err.Error(),
		})
		return
	}
	
	// 处理 Ping 数据
	timestamp := time.Now()
	if req.Timestamp > 0 {
		timestamp = time.UnixMilli(req.Timestamp)
	}
	
	if err := h.dataReceiverService.ProcessPingData(c.Request.Context(), device.ID, req.Timestamp, req.Pings); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "Ping 数据处理失败",
			Details: err.Error(),
		})
		return
	}
	
	// 更新设备状态
	if err := h.dataReceiverService.UpdateDeviceOnlineStatus(c.Request.Context(), device.ID); err != nil {
		log.Printf("Failed to update device status for %d: %v", device.ID, err)
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "Ping 数据接收成功",
		"device_id": device.ID,
		"timestamp": timestamp.Unix(),
	})
}

// GetAllDevicesStatus 获取所有设备状态
// @Summary 获取所有设备状态
// @Description 获取所有设备的在线状态
// @Tags 数据查询
// @Produce json
// @Success 200 {object} map[string]interface{} "设备状态列表"
// @Failure 500 {object} models.ErrorResponse "服务器内部错误"
// @Router /api/v1/data/status [get]
func (h *DataReceiverHandler) GetAllDevicesStatus(c *gin.Context) {
	statuses, err := h.dataReceiverService.GetAllDevicesStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Code:    http.StatusInternalServerError,
			Message: "获取设备状态失败",
			Details: err.Error(),
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"devices":   statuses,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}