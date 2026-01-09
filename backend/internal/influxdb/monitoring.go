package influxdb

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// 监控数据 measurement 名称
const (
	MeasurementBandwidth = "bandwidth"
	MeasurementPing      = "ping"
)

// BandwidthData 带宽数据结构
type BandwidthData struct {
	DeviceID  string    `json:"device_id"`
	Interface string    `json:"interface"`
	RxRate    float64   `json:"rx_rate"` // 接收速率 (bps)
	TxRate    float64   `json:"tx_rate"` // 发送速率 (bps)
	Timestamp time.Time `json:"timestamp"`
}

// PingData Ping 数据结构
type PingData struct {
	DeviceID      string    `json:"device_id"`
	TargetID      string    `json:"target_id"`
	TargetAddress string    `json:"target_address"`
	Latency       *float64  `json:"latency"` // 延迟 (ms)，nil 表示丢包
	Status        string    `json:"status"`  // up/down
	Timestamp     time.Time `json:"timestamp"`
}

// WriteBandwidthData 写入带宽数据
func (c *Client) WriteBandwidthData(data *BandwidthData) error {
	tags := map[string]string{
		"device_id": data.DeviceID,
		"interface": data.Interface,
	}
	fields := map[string]interface{}{
		"rx_rate": data.RxRate,
		"tx_rate": data.TxRate,
	}
	return c.WritePoint(MeasurementBandwidth, tags, fields, data.Timestamp)
}

// WriteBandwidthDataBatch 批量写入带宽数据
func (c *Client) WriteBandwidthDataBatch(dataList []*BandwidthData) error {
	for _, data := range dataList {
		if err := c.WriteBandwidthData(data); err != nil {
			return err
		}
	}
	return nil
}

// WritePingData 写入 Ping 数据
func (c *Client) WritePingData(data *PingData) error {
	tags := map[string]string{
		"device_id":      data.DeviceID,
		"target_id":      data.TargetID,
		"target_address": data.TargetAddress,
	}
	fields := map[string]interface{}{
		"status": data.Status,
	}
	// 如果有延迟值，添加到字段中
	if data.Latency != nil {
		fields["latency"] = *data.Latency
	} else {
		// 丢包时设置延迟为 -1 表示丢包
		fields["latency"] = float64(-1)
	}
	return c.WritePoint(MeasurementPing, tags, fields, data.Timestamp)
}

// WritePingDataBatch 批量写入 Ping 数据
func (c *Client) WritePingDataBatch(dataList []*PingData) error {
	for _, data := range dataList {
		if err := c.WritePingData(data); err != nil {
			return err
		}
	}
	return nil
}

// QueryBandwidthData 查询带宽数据
func (c *Client) QueryBandwidthData(deviceID string, interfaceName string, start, end time.Time) ([]*BandwidthData, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "%s")
		|> filter(fn: (r) => r.device_id == "%s")`,
		c.config.Bucket,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		MeasurementBandwidth,
		deviceID,
	)

	if interfaceName != "" {
		query += fmt.Sprintf(`|> filter(fn: (r) => r.interface == "%s")`, interfaceName)
	}

	query += `|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")`

	result, err := c.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query bandwidth data: %w", err)
	}

	var dataList []*BandwidthData
	for result.Next() {
		record := result.Record()
		data := &BandwidthData{
			DeviceID:  deviceID,
			Timestamp: record.Time(),
		}
		
		if iface, ok := record.ValueByKey("interface").(string); ok {
			data.Interface = iface
		}
		if rxRate, ok := record.ValueByKey("rx_rate").(float64); ok {
			data.RxRate = rxRate
		}
		if txRate, ok := record.ValueByKey("tx_rate").(float64); ok {
			data.TxRate = txRate
		}
		
		dataList = append(dataList, data)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query result error: %w", result.Err())
	}

	return dataList, nil
}

// QueryPingData 查询 Ping 数据
func (c *Client) QueryPingData(deviceID string, targetID string, start, end time.Time) ([]*PingData, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "%s")
		|> filter(fn: (r) => r.device_id == "%s")`,
		c.config.Bucket,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		MeasurementPing,
		deviceID,
	)

	if targetID != "" {
		query += fmt.Sprintf(`|> filter(fn: (r) => r.target_id == "%s")`, targetID)
	}

	query += `|> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")`

	result, err := c.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("failed to query ping data: %w", err)
	}

	var dataList []*PingData
	for result.Next() {
		record := result.Record()
		data := &PingData{
			DeviceID:  deviceID,
			Timestamp: record.Time(),
		}
		
		if tid, ok := record.ValueByKey("target_id").(string); ok {
			data.TargetID = tid
		}
		if addr, ok := record.ValueByKey("target_address").(string); ok {
			data.TargetAddress = addr
		}
		if latency, ok := record.ValueByKey("latency").(float64); ok {
			if latency >= 0 {
				data.Latency = &latency
				data.Status = "up"
			} else {
				data.Status = "down"
			}
		}
		if status, ok := record.ValueByKey("status").(string); ok {
			data.Status = status
		}
		
		dataList = append(dataList, data)
	}

	if result.Err() != nil {
		return nil, fmt.Errorf("query result error: %w", result.Err())
	}

	return dataList, nil
}

// DeleteDeviceData 删除指定设备的所有数据
func (c *Client) DeleteDeviceData(deviceID string) error {
	ctx := context.Background()
	deleteAPI := c.client.DeleteAPI()

	// 删除带宽数据
	predicate := fmt.Sprintf(`device_id="%s"`, deviceID)
	start := time.Unix(0, 0)
	stop := time.Now().Add(24 * time.Hour) // 包含未来的数据

	err := deleteAPI.DeleteWithName(ctx, c.config.Org, c.config.Bucket, start, stop, predicate)
	if err != nil {
		return fmt.Errorf("failed to delete device data: %w", err)
	}

	log.Printf("Deleted all data for device %s", deviceID)
	return nil
}

// DeleteOldData 删除超过保留期的数据
func (c *Client) DeleteOldData(retentionDays int) error {
	ctx := context.Background()
	deleteAPI := c.client.DeleteAPI()

	stop := time.Now().AddDate(0, 0, -retentionDays)
	start := time.Unix(0, 0)

	err := deleteAPI.DeleteWithName(ctx, c.config.Org, c.config.Bucket, start, stop, "")
	if err != nil {
		return fmt.Errorf("failed to delete old data: %w", err)
	}

	log.Printf("Deleted data older than %d days", retentionDays)
	return nil
}

// SetupMonitoringBucket 设置监控数据存储桶和保留策略
func (c *Client) SetupMonitoringBucket(retentionDays int) error {
	ctx := context.Background()
	bucketsAPI := c.client.BucketsAPI()

	// 查找现有的存储桶
	bucket, err := bucketsAPI.FindBucketByName(ctx, c.config.Bucket)
	if err != nil {
		// 存储桶不存在，创建新的
		return c.createMonitoringBucket(retentionDays)
	}

	// 更新保留策略
	retentionSeconds := int64(retentionDays * 24 * 60 * 60)
	expireType := domain.RetentionRuleTypeExpire
	if len(bucket.RetentionRules) > 0 {
		bucket.RetentionRules[0].EverySeconds = retentionSeconds
	} else {
		bucket.RetentionRules = []domain.RetentionRule{
			{
				EverySeconds: retentionSeconds,
				Type:         &expireType,
			},
		}
	}

	_, err = bucketsAPI.UpdateBucket(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to update bucket retention policy: %w", err)
	}

	log.Printf("Updated bucket '%s' retention policy to %d days", c.config.Bucket, retentionDays)
	return nil
}

// createMonitoringBucket 创建监控数据存储桶
func (c *Client) createMonitoringBucket(retentionDays int) error {
	ctx := context.Background()
	bucketsAPI := c.client.BucketsAPI()
	orgAPI := c.client.OrganizationsAPI()

	// 获取组织
	org, err := orgAPI.FindOrganizationByName(ctx, c.config.Org)
	if err != nil {
		return fmt.Errorf("failed to find organization: %w", err)
	}

	// 计算保留时间（秒）
	retentionSeconds := int64(retentionDays * 24 * 60 * 60)
	expireType := domain.RetentionRuleTypeExpire

	// 创建存储桶
	bucket := &domain.Bucket{
		Name:  c.config.Bucket,
		OrgID: org.Id,
		RetentionRules: []domain.RetentionRule{
			{
				EverySeconds: retentionSeconds,
				Type:         &expireType,
			},
		},
	}

	_, err = bucketsAPI.CreateBucket(ctx, bucket)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	log.Printf("Created bucket '%s' with %d days retention", c.config.Bucket, retentionDays)
	return nil
}

// GetBucketInfo 获取存储桶信息
func (c *Client) GetBucketInfo() (*BucketInfo, error) {
	ctx := context.Background()
	bucketsAPI := c.client.BucketsAPI()

	bucket, err := bucketsAPI.FindBucketByName(ctx, c.config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("failed to find bucket: %w", err)
	}

	info := &BucketInfo{
		Name:      bucket.Name,
		ID:        *bucket.Id,
		CreatedAt: bucket.CreatedAt,
	}

	if len(bucket.RetentionRules) > 0 {
		info.RetentionDays = int(bucket.RetentionRules[0].EverySeconds / (24 * 60 * 60))
	}

	return info, nil
}

// BucketInfo 存储桶信息
type BucketInfo struct {
	Name          string     `json:"name"`
	ID            string     `json:"id"`
	RetentionDays int        `json:"retention_days"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
}
