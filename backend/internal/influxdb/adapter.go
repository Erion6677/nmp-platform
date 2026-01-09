package influxdb

import (
	"nmp-platform/internal/service"
	"time"
)

// ServiceAdapter 适配器，将 influxdb.Client 适配为 service.InfluxClient
type ServiceAdapter struct {
	client *Client
}

// NewServiceAdapter 创建新的服务适配器
func NewServiceAdapter(client *Client) service.InfluxClient {
	return &ServiceAdapter{client: client}
}

// WritePoint 实现 service.InfluxClient 接口
func (a *ServiceAdapter) WritePoint(measurement string, tags map[string]string, fields map[string]interface{}, timestamp time.Time) error {
	return a.client.WritePoint(measurement, tags, fields, timestamp)
}

// Query 实现 service.InfluxClient 接口
func (a *ServiceAdapter) Query(query string) (service.QueryResult, error) {
	result, err := a.client.Query(query)
	if err != nil {
		return nil, err
	}
	return &ServiceQueryResultAdapter{result: result}, nil
}

// Health 实现 service.InfluxClient 接口
func (a *ServiceAdapter) Health() error {
	return a.client.Health()
}

// Close 实现 service.InfluxClient 接口
func (a *ServiceAdapter) Close() {
	a.client.Close()
}

// Flush 实现 service.InfluxClient 接口
func (a *ServiceAdapter) Flush() {
	a.client.Flush()
}

// Delete 实现 service.InfluxClient 接口
func (a *ServiceAdapter) Delete(start, stop time.Time, predicate string) error {
	return a.client.Delete(start, stop, predicate)
}

// ServiceQueryResultAdapter 适配器，将 influxdb.QueryResult 适配为 service.QueryResult
type ServiceQueryResultAdapter struct {
	result QueryResult
}

func (a *ServiceQueryResultAdapter) Next() bool {
	return a.result.Next()
}

func (a *ServiceQueryResultAdapter) Record() service.QueryRecord {
	return &ServiceQueryRecordAdapter{record: a.result.Record()}
}

func (a *ServiceQueryResultAdapter) Err() error {
	return a.result.Err()
}

// ServiceQueryRecordAdapter 适配器，将 influxdb.QueryRecord 适配为 service.QueryRecord
type ServiceQueryRecordAdapter struct {
	record QueryRecord
}

func (a *ServiceQueryRecordAdapter) Time() time.Time {
	return a.record.Time()
}

func (a *ServiceQueryRecordAdapter) Field() string {
	return a.record.Field()
}

func (a *ServiceQueryRecordAdapter) Value() interface{} {
	return a.record.Value()
}

func (a *ServiceQueryRecordAdapter) ValueByKey(key string) interface{} {
	return a.record.ValueByKey(key)
}