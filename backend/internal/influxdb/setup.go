package influxdb

import (
	"log"
)

// DefaultRetentionDays 默认数据保留天数
const DefaultRetentionDays = 10

// SetupMonitoring 初始化监控数据存储
// 这个函数应该在应用启动时调用
func (c *Client) SetupMonitoring() error {
	log.Println("Setting up InfluxDB monitoring storage...")

	// 设置存储桶和保留策略
	if err := c.SetupMonitoringBucket(DefaultRetentionDays); err != nil {
		log.Printf("Warning: Failed to setup monitoring bucket: %v", err)
		// 不返回错误，因为存储桶可能已经存在
	}

	log.Println("InfluxDB monitoring storage setup completed")
	return nil
}

// InitializeMonitoringData 初始化监控数据结构
// 这个函数用于验证 InfluxDB 配置是否正确
func (c *Client) InitializeMonitoringData() error {
	// 检查健康状态
	if err := c.Health(); err != nil {
		return err
	}

	// 获取存储桶信息
	info, err := c.GetBucketInfo()
	if err != nil {
		log.Printf("Warning: Could not get bucket info: %v", err)
		return nil
	}

	log.Printf("InfluxDB bucket '%s' is ready (retention: %d days)", info.Name, info.RetentionDays)
	return nil
}

// MonitoringMeasurements 返回所有监控相关的 measurement 名称
func MonitoringMeasurements() []string {
	return []string{
		MeasurementBandwidth,
		MeasurementPing,
	}
}

// MonitoringTags 返回监控数据使用的标签
func MonitoringTags() map[string][]string {
	return map[string][]string{
		MeasurementBandwidth: {"device_id", "interface"},
		MeasurementPing:      {"device_id", "target_id", "target_address"},
	}
}

// MonitoringFields 返回监控数据使用的字段
func MonitoringFields() map[string][]string {
	return map[string][]string{
		MeasurementBandwidth: {"rx_rate", "tx_rate"},
		MeasurementPing:      {"latency", "status"},
	}
}
