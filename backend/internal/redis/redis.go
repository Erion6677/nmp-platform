package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"nmp-platform/internal/config"
	"time"

	"github.com/go-redis/redis/v8"
)

// Client Redis客户端包装器
type Client struct {
	client *redis.Client
	config *config.RedisConfig
}

// Connect 连接到Redis
func Connect(cfg *config.RedisConfig) (*Client, error) {
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	log.Printf("Successfully connected to Redis at %s:%d", cfg.Host, cfg.Port)

	return &Client{
		client: rdb,
		config: cfg,
	}, nil
}

// Set 设置键值对
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

// Get 获取值
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, key).Result()
}

// GetJSON 获取JSON值并反序列化
func (c *Client) GetJSON(ctx context.Context, key string, dest interface{}) error {
	val, err := c.client.Get(ctx, key).Result()
	if err != nil {
		return err
	}

	return json.Unmarshal([]byte(val), dest)
}

// SetJSON 序列化并设置JSON值
func (c *Client) SetJSON(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return c.client.Set(ctx, key, jsonData, expiration).Err()
}

// Delete 删除键
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// Exists 检查键是否存在
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, key).Result()
	return count > 0, err
}

// Expire 设置键的过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

// TTL 获取键的剩余生存时间
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

// HSet 设置哈希字段
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) error {
	return c.client.HSet(ctx, key, values...).Err()
}

// HGet 获取哈希字段值
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	return c.client.HGet(ctx, key, field).Result()
}

// HGetAll 获取哈希的所有字段和值
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, key).Result()
}

// HDel 删除哈希字段
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.HDel(ctx, key, fields...).Err()
}

// LPush 从列表左侧推入元素
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.LPush(ctx, key, values...).Err()
}

// RPush 从列表右侧推入元素
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.RPush(ctx, key, values...).Err()
}

// LPop 从列表左侧弹出元素
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	return c.client.LPop(ctx, key).Result()
}

// RPop 从列表右侧弹出元素
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	return c.client.RPop(ctx, key).Result()
}

// LRange 获取列表指定范围的元素
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, key, start, stop).Result()
}

// LLen 获取列表长度
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.client.LLen(ctx, key).Result()
}

// SAdd 向集合添加成员
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, key, members...).Err()
}

// SMembers 获取集合的所有成员
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, key).Result()
}

// SIsMember 判断成员是否在集合中
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.client.SIsMember(ctx, key, member).Result()
}

// SRem 从集合中移除成员
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SRem(ctx, key, members...).Err()
}

// ZAdd 向有序集合添加成员
func (c *Client) ZAdd(ctx context.Context, key string, members ...*redis.Z) error {
	return c.client.ZAdd(ctx, key, members...).Err()
}

// ZRange 获取有序集合指定范围的成员
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.ZRange(ctx, key, start, stop).Result()
}

// ZRangeByScore 根据分数范围获取有序集合成员
func (c *Client) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return c.client.ZRangeByScore(ctx, key, opt).Result()
}

// Incr 递增键的值
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

// IncrBy 按指定值递增键的值
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value).Result()
}

// Decr 递减键的值
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

// DecrBy 按指定值递减键的值
func (c *Client) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.DecrBy(ctx, key, value).Result()
}

// Keys 获取匹配模式的所有键
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, pattern).Result()
}

// FlushDB 清空当前数据库
func (c *Client) FlushDB(ctx context.Context) error {
	return c.client.FlushDB(ctx).Err()
}

// FlushAll 清空所有数据库
func (c *Client) FlushAll(ctx context.Context) error {
	return c.client.FlushAll(ctx).Err()
}

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Info 获取Redis服务器信息
func (c *Client) Info(ctx context.Context, section ...string) (string, error) {
	return c.client.Info(ctx, section...).Result()
}

// Close 关闭连接
func (c *Client) Close() error {
	err := c.client.Close()
	if err == nil {
		log.Println("Redis connection closed")
	}
	return err
}

// Health 检查Redis健康状态
func (c *Client) Health() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return c.Ping(ctx)
}


// ========== 设备状态管理方法 ==========

// UpdateDeviceStatus 更新设备在线状态
func (c *Client) UpdateDeviceStatus(ctx context.Context, deviceID uint, status string) error {
	key := fmt.Sprintf("device:status:%d", deviceID)
	return c.Set(ctx, key, status, 24*time.Hour)
}

// GetDeviceStatus 获取设备在线状态
func (c *Client) GetDeviceStatus(ctx context.Context, deviceID uint) (string, error) {
	key := fmt.Sprintf("device:status:%d", deviceID)
	return c.Get(ctx, key)
}

// UpdateDeviceLastSeen 更新设备最后在线时间
func (c *Client) UpdateDeviceLastSeen(ctx context.Context, deviceID uint) error {
	key := fmt.Sprintf("device:last_seen:%d", deviceID)
	return c.Set(ctx, key, time.Now().Format(time.RFC3339), 24*time.Hour)
}

// GetDeviceLastSeen 获取设备最后在线时间
func (c *Client) GetDeviceLastSeen(ctx context.Context, deviceID uint) (time.Time, error) {
	key := fmt.Sprintf("device:last_seen:%d", deviceID)
	val, err := c.Get(ctx, key)
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, val)
}

// UpdateDeviceMetrics 更新设备最新指标数据
func (c *Client) UpdateDeviceMetrics(ctx context.Context, deviceID uint, metrics interface{}) error {
	key := fmt.Sprintf("device:metrics:%d", deviceID)
	return c.SetJSON(ctx, key, metrics, 5*time.Minute)
}

// GetDeviceMetrics 获取设备最新指标数据
func (c *Client) GetDeviceMetrics(ctx context.Context, deviceID uint, dest interface{}) error {
	key := fmt.Sprintf("device:metrics:%d", deviceID)
	return c.GetJSON(ctx, key, dest)
}

// UpdateBandwidthData 更新带宽数据缓存
func (c *Client) UpdateBandwidthData(ctx context.Context, deviceID uint, data interface{}) error {
	key := fmt.Sprintf("device:bandwidth:%d", deviceID)
	return c.SetJSON(ctx, key, data, 5*time.Minute)
}

// GetBandwidthData 获取带宽数据缓存
func (c *Client) GetBandwidthData(ctx context.Context, deviceID uint, dest interface{}) error {
	key := fmt.Sprintf("device:bandwidth:%d", deviceID)
	return c.GetJSON(ctx, key, dest)
}

// UpdatePingData 更新 Ping 数据缓存
func (c *Client) UpdatePingData(ctx context.Context, deviceID uint, data interface{}) error {
	key := fmt.Sprintf("device:ping:%d", deviceID)
	return c.SetJSON(ctx, key, data, 5*time.Minute)
}

// GetPingData 获取 Ping 数据缓存
func (c *Client) GetPingData(ctx context.Context, deviceID uint, dest interface{}) error {
	key := fmt.Sprintf("device:ping:%d", deviceID)
	return c.GetJSON(ctx, key, dest)
}

// CacheDeviceByIP 缓存设备信息（按 IP）
func (c *Client) CacheDeviceByIP(ctx context.Context, ip string, device interface{}) error {
	key := fmt.Sprintf("device:ip:%s", ip)
	return c.SetJSON(ctx, key, device, time.Hour)
}

// GetDeviceByIP 获取缓存的设备信息（按 IP）
func (c *Client) GetDeviceByIP(ctx context.Context, ip string, dest interface{}) error {
	key := fmt.Sprintf("device:ip:%s", ip)
	return c.GetJSON(ctx, key, dest)
}

// GetAllDeviceStatusKeys 获取所有设备状态键
func (c *Client) GetAllDeviceStatusKeys(ctx context.Context) ([]string, error) {
	return c.Keys(ctx, "device:status:*")
}

// GetAllDeviceLastSeenKeys 获取所有设备最后在线时间键
func (c *Client) GetAllDeviceLastSeenKeys(ctx context.Context) ([]string, error) {
	return c.Keys(ctx, "device:last_seen:*")
}

// BatchUpdateDeviceStatus 批量更新设备状态
func (c *Client) BatchUpdateDeviceStatus(ctx context.Context, statuses map[uint]string) error {
	for deviceID, status := range statuses {
		if err := c.UpdateDeviceStatus(ctx, deviceID, status); err != nil {
			log.Printf("Failed to update status for device %d: %v", deviceID, err)
		}
	}
	return nil
}

// ClearDeviceCache 清除设备相关的所有缓存
func (c *Client) ClearDeviceCache(ctx context.Context, deviceID uint) error {
	patterns := []string{
		fmt.Sprintf("device:status:%d", deviceID),
		fmt.Sprintf("device:last_seen:%d", deviceID),
		fmt.Sprintf("device:metrics:%d", deviceID),
		fmt.Sprintf("device:bandwidth:%d", deviceID),
		fmt.Sprintf("device:ping:%d", deviceID),
		fmt.Sprintf("device:latest:%d", deviceID),
		fmt.Sprintf("device:exists:%d", deviceID),
	}
	
	return c.Delete(ctx, patterns...)
}
