package influxdb

import (
	"context"
	"fmt"
	"log"
	"nmp-platform/internal/config"
	"sync"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/query"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// 为了避免循环导入，我们在这里定义接口
type QueryResult interface {
	Next() bool
	Record() QueryRecord
	Err() error
}

type QueryRecord interface {
	Time() time.Time
	Field() string
	Value() interface{}
	ValueByKey(key string) interface{} // 获取 tag 或其他字段的值
}

// QueryResult 适配器，实现 QueryResult 接口
type QueryResultAdapter struct {
	result *api.QueryTableResult
}

func (q *QueryResultAdapter) Next() bool {
	return q.result.Next()
}

func (q *QueryResultAdapter) Record() QueryRecord {
	return &QueryRecordAdapter{record: q.result.Record()}
}

func (q *QueryResultAdapter) Err() error {
	return q.result.Err()
}

// QueryRecord 适配器，实现 service.QueryRecord 接口
type QueryRecordAdapter struct {
	record *query.FluxRecord
}

func (q *QueryRecordAdapter) Time() time.Time {
	return q.record.Time()
}

func (q *QueryRecordAdapter) Field() string {
	return q.record.Field()
}

func (q *QueryRecordAdapter) Value() interface{} {
	return q.record.Value()
}

func (q *QueryRecordAdapter) ValueByKey(key string) interface{} {
	return q.record.ValueByKey(key)
}

// Client InfluxDB客户端包装器
type Client struct {
	client          influxdb2.Client
	writeAPI        api.WriteAPI
	writeAPIBlocking api.WriteAPIBlocking
	queryAPI        api.QueryAPI
	config          *config.InfluxConfig
	mu              sync.Mutex
}

// Connect 连接到InfluxDB
func Connect(cfg *config.InfluxConfig) (*Client, error) {
	// 创建InfluxDB客户端
	client := influxdb2.NewClient(cfg.URL, cfg.Token)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to InfluxDB: %w", err)
	}

	if health.Status != "pass" {
		return nil, fmt.Errorf("InfluxDB health check failed: %s", health.Status)
	}

	// 创建写入和查询API
	writeAPI := client.WriteAPI(cfg.Org, cfg.Bucket)
	writeAPIBlocking := client.WriteAPIBlocking(cfg.Org, cfg.Bucket)
	queryAPI := client.QueryAPI(cfg.Org)

	// 设置错误处理
	errorsCh := writeAPI.Errors()
	go func() {
		for err := range errorsCh {
			log.Printf("InfluxDB async write error: %v", err)
		}
	}()

	log.Printf("Successfully connected to InfluxDB at %s (org: %s, bucket: %s)", cfg.URL, cfg.Org, cfg.Bucket)

	return &Client{
		client:           client,
		writeAPI:         writeAPI,
		writeAPIBlocking: writeAPIBlocking,
		queryAPI:         queryAPI,
		config:           cfg,
	}, nil
}

// WritePoint 写入数据点（使用同步 API 确保数据被写入）
func (c *Client) WritePoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	point := influxdb2.NewPointWithMeasurement(measurement)
	
	// 添加标签
	for k, v := range tags {
		point = point.AddTag(k, v)
	}
	
	// 添加字段
	for k, v := range fields {
		point = point.AddField(k, v)
	}
	
	// 设置时间戳
	point = point.SetTime(timestamp)
	
	// 使用同步 API 写入，确保数据被写入并能捕获错误
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := c.writeAPIBlocking.WritePoint(ctx, point); err != nil {
		log.Printf("InfluxDB WritePoint error: measurement=%s, tags=%v, fields=%v, error=%v", 
			measurement, tags, fields, err)
		return fmt.Errorf("failed to write point to InfluxDB: %w", err)
	}
	
	return nil
}

// WriteBatch 批量写入数据点
func (c *Client) WriteBatch(points []*write.Point) error {
	for _, point := range points {
		c.writeAPI.WritePoint(point)
	}
	return nil
}

// Query 执行查询，返回 QueryResult 接口
func (c *Client) Query(query string) (QueryResult, error) {
	result, err := c.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	return &QueryResultAdapter{result: result}, nil
}

// QueryRange 查询指定时间范围的数据
func (c *Client) QueryRange(measurement string, start, end time.Time, filters map[string]string) (*api.QueryTableResult, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: %s, stop: %s)
		|> filter(fn: (r) => r._measurement == "%s")`,
		c.config.Bucket,
		start.Format(time.RFC3339),
		end.Format(time.RFC3339),
		measurement,
	)

	// 添加额外的过滤条件
	for key, value := range filters {
		query += fmt.Sprintf(`|> filter(fn: (r) => r.%s == "%s")`, key, value)
	}

	return c.queryAPI.Query(context.Background(), query)
}

// QueryLatest 查询最新数据
func (c *Client) QueryLatest(measurement string, filters map[string]string, limit int) (*api.QueryTableResult, error) {
	query := fmt.Sprintf(`
		from(bucket: "%s")
		|> range(start: -1h)
		|> filter(fn: (r) => r._measurement == "%s")`,
		c.config.Bucket,
		measurement,
	)

	// 添加过滤条件
	for key, value := range filters {
		query += fmt.Sprintf(`|> filter(fn: (r) => r.%s == "%s")`, key, value)
	}

	// 添加排序和限制
	query += fmt.Sprintf(`|> sort(columns: ["_time"], desc: true) |> limit(n: %d)`, limit)

	return c.queryAPI.Query(context.Background(), query)
}

// Flush 强制刷新写入缓冲区
func (c *Client) Flush() {
	c.writeAPI.Flush()
}

// Close 关闭连接
func (c *Client) Close() {
	c.writeAPI.Flush()
	c.client.Close()
	log.Println("InfluxDB connection closed")
}

// Health 检查InfluxDB健康状态
func (c *Client) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := c.client.Health(ctx)
	if err != nil {
		return fmt.Errorf("InfluxDB health check failed: %w", err)
	}

	if health.Status != "pass" {
		return fmt.Errorf("InfluxDB is not healthy: %s", health.Status)
	}

	return nil
}

// Delete 删除指定时间范围和条件的数据
func (c *Client) Delete(start, stop time.Time, predicate string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	deleteAPI := c.client.DeleteAPI()
	
	log.Printf("Deleting data: bucket=%s, org=%s, start=%v, stop=%v, predicate=%s",
		c.config.Bucket, c.config.Org, start, stop, predicate)
	
	err := deleteAPI.DeleteWithName(ctx, c.config.Org, c.config.Bucket, start, stop, predicate)
	if err != nil {
		return fmt.Errorf("failed to delete data from InfluxDB: %w", err)
	}
	
	log.Printf("Successfully deleted data with predicate: %s", predicate)
	return nil
}

// CreateBucket 创建存储桶（如果不存在）
func (c *Client) CreateBucket(bucketName string, retentionPeriod time.Duration) error {
	ctx := context.Background()
	bucketsAPI := c.client.BucketsAPI()

	// 检查存储桶是否存在
	bucket, err := bucketsAPI.FindBucketByName(ctx, bucketName)
	if err == nil && bucket != nil {
		log.Printf("Bucket '%s' already exists", bucketName)
		return nil
	}

	// 创建新存储桶
	orgAPI := c.client.OrganizationsAPI()
	org, err := orgAPI.FindOrganizationByName(ctx, c.config.Org)
	if err != nil {
		return fmt.Errorf("failed to find organization: %w", err)
	}

	// 创建存储桶（简化版本，不设置保留期）
	_, err = bucketsAPI.CreateBucketWithName(ctx, org, bucketName)
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	log.Printf("Created bucket '%s'", bucketName)
	return nil
}