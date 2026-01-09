# InfluxDB 监控数据存储配置

## 概述

本文档描述了 NMP 2.0 监控系统使用的 InfluxDB 数据结构。

## Measurements

### 1. bandwidth - 带宽数据

用于存储设备接口的带宽监控数据。

**Tags (标签):**
- `device_id`: 设备ID
- `interface`: 接口名称

**Fields (字段):**
- `rx_rate`: 接收速率 (bps, float64)
- `tx_rate`: 发送速率 (bps, float64)

**示例数据点:**
```
bandwidth,device_id=1,interface=ether1 rx_rate=1000000,tx_rate=500000 1704067200000000000
```

### 2. ping - Ping 延迟数据

用于存储设备到目标地址的 Ping 监控数据。

**Tags (标签):**
- `device_id`: 设备ID
- `target_id`: Ping 目标ID
- `target_address`: 目标地址

**Fields (字段):**
- `latency`: 延迟 (ms, float64)，-1 表示丢包
- `status`: 状态 (string, "up" 或 "down")

**示例数据点:**
```
ping,device_id=1,target_id=1,target_address=8.8.8.8 latency=10.5,status="up" 1704067200000000000
ping,device_id=1,target_id=1,target_address=8.8.8.8 latency=-1,status="down" 1704067201000000000
```

## Retention Policy (保留策略)

默认配置：
- 数据保留天数: 10 天
- 可通过系统设置 `data_retention_days` 配置

## 手动设置 (如果需要)

### 使用 InfluxDB CLI

```bash
# 创建存储桶（如果不存在）
influx bucket create \
  --name monitoring \
  --org nmp \
  --retention 10d

# 或者更新现有存储桶的保留策略
influx bucket update \
  --name monitoring \
  --retention 10d
```

### 使用 InfluxDB API

```bash
# 创建存储桶
curl -X POST "http://localhost:8086/api/v2/buckets" \
  -H "Authorization: Token YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "monitoring",
    "orgID": "YOUR_ORG_ID",
    "retentionRules": [
      {
        "type": "expire",
        "everySeconds": 864000
      }
    ]
  }'
```

## 查询示例

### 查询最近 1 小时的带宽数据

```flux
from(bucket: "monitoring")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "bandwidth")
  |> filter(fn: (r) => r.device_id == "1")
  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
```

### 查询最近 1 小时的 Ping 数据

```flux
from(bucket: "monitoring")
  |> range(start: -1h)
  |> filter(fn: (r) => r._measurement == "ping")
  |> filter(fn: (r) => r.device_id == "1")
  |> pivot(rowKey:["_time"], columnKey: ["_field"], valueColumn: "_value")
```

### 查询丢包数据

```flux
from(bucket: "monitoring")
  |> range(start: -24h)
  |> filter(fn: (r) => r._measurement == "ping")
  |> filter(fn: (r) => r._field == "latency")
  |> filter(fn: (r) => r._value < 0)
```

## 数据清理

### 删除指定设备的数据

```flux
// 使用 Delete API
// predicate: device_id="1"
```

### 删除超过保留期的数据

系统会自动根据 Retention Policy 清理过期数据，也可以手动触发清理。

## 注意事项

1. InfluxDB 2.x 使用 Flux 查询语言
2. 时间戳使用纳秒精度
3. 丢包用 latency=-1 表示，便于查询和统计
4. 建议定期检查数据量，避免存储空间不足
