// Package collector 提供设备数据采集功能
package collector

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-routeros/routeros/v3"
)

// SystemInfo 系统信息结构
type SystemInfo struct {
	DeviceName   string  `json:"device_name"`
	DeviceIP     string  `json:"device_ip"`
	CPUCount     int     `json:"cpu_count"`
	Version      string  `json:"version"`
	License      string  `json:"license"`
	Uptime       int64   `json:"uptime"`       // 秒
	CPUUsage     float64 `json:"cpu_usage"`    // 百分比
	MemoryUsage  float64 `json:"memory_usage"` // 百分比
	MemoryTotal  int64   `json:"memory_total"` // 字节
	MemoryFree   int64   `json:"memory_free"`  // 字节
}

// InterfaceInfo 接口信息结构
type InterfaceInfo struct {
	Name   string `json:"name"`
	Status string `json:"status"` // up, down
}

// RouterOSCollector MikroTik RouterOS API 采集器
type RouterOSCollector struct {
	Timeout time.Duration
}

// NewRouterOSCollector 创建新的 RouterOS 采集器
func NewRouterOSCollector(timeout time.Duration) *RouterOSCollector {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &RouterOSCollector{
		Timeout: timeout,
	}
}

// Connect 连接到 RouterOS 设备
func (c *RouterOSCollector) Connect(ip string, port int, username, password string) (*routeros.Client, error) {
	address := fmt.Sprintf("%s:%d", ip, port)
	
	client, err := routeros.DialTimeout(address, username, password, c.Timeout)
	if err != nil {
		return nil, c.wrapError(err)
	}
	
	return client, nil
}

// TestConnection 测试连接
func (c *RouterOSCollector) TestConnection(ip string, port int, username, password string) error {
	client, err := c.Connect(ip, port, username, password)
	if err != nil {
		return err
	}
	defer client.Close()
	
	// 执行简单命令验证连接
	_, err = client.Run("/system/identity/print")
	if err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}
	
	return nil
}

// GetSystemInfo 获取系统信息
func (c *RouterOSCollector) GetSystemInfo(client *routeros.Client) (*SystemInfo, error) {
	info := &SystemInfo{}
	
	// 获取设备名称
	reply, err := client.Run("/system/identity/print")
	if err != nil {
		return nil, fmt.Errorf("获取设备名称失败: %w", err)
	}
	if len(reply.Re) > 0 {
		info.DeviceName = reply.Re[0].Map["name"]
	}
	
	// 获取系统资源信息
	reply, err = client.Run("/system/resource/print")
	if err != nil {
		return nil, fmt.Errorf("获取系统资源失败: %w", err)
	}
	if len(reply.Re) > 0 {
		re := reply.Re[0].Map
		
		// CPU 核心数
		if cpuCount, ok := re["cpu-count"]; ok {
			info.CPUCount, _ = strconv.Atoi(cpuCount)
		}
		
		// 系统版本
		info.Version = re["version"]
		
		// 运行时间
		if uptime, ok := re["uptime"]; ok {
			info.Uptime = c.parseUptime(uptime)
		}
		
		// CPU 使用率
		if cpuLoad, ok := re["cpu-load"]; ok {
			info.CPUUsage, _ = strconv.ParseFloat(cpuLoad, 64)
		}
		
		// 内存信息
		if totalMem, ok := re["total-memory"]; ok {
			info.MemoryTotal, _ = strconv.ParseInt(totalMem, 10, 64)
		}
		if freeMem, ok := re["free-memory"]; ok {
			info.MemoryFree, _ = strconv.ParseInt(freeMem, 10, 64)
		}
		
		// 计算内存使用率
		if info.MemoryTotal > 0 {
			usedMem := info.MemoryTotal - info.MemoryFree
			info.MemoryUsage = float64(usedMem) / float64(info.MemoryTotal) * 100
		}
	}
	
	// 获取授权信息
	reply, err = client.Run("/system/license/print")
	if err == nil && len(reply.Re) > 0 {
		if level, ok := reply.Re[0].Map["level"]; ok {
			info.License = level
		} else if nlevel, ok := reply.Re[0].Map["nlevel"]; ok {
			info.License = nlevel
		}
	}
	
	return info, nil
}

// GetInterfaces 获取接口列表
func (c *RouterOSCollector) GetInterfaces(client *routeros.Client) ([]InterfaceInfo, error) {
	reply, err := client.Run("/interface/print")
	if err != nil {
		return nil, fmt.Errorf("获取接口列表失败: %w", err)
	}
	
	interfaces := make([]InterfaceInfo, 0, len(reply.Re))
	for _, re := range reply.Re {
		iface := InterfaceInfo{
			Name:   re.Map["name"],
			Status: "down",
		}
		
		// 检查接口状态
		if running, ok := re.Map["running"]; ok && running == "true" {
			iface.Status = "up"
		}
		
		interfaces = append(interfaces, iface)
	}
	
	return interfaces, nil
}


// GetInterfaceTraffic 获取接口流量数据
func (c *RouterOSCollector) GetInterfaceTraffic(client *routeros.Client, interfaceName string) (rxRate, txRate int64, err error) {
	reply, err := client.Run("/interface/monitor-traffic", "=interface="+interfaceName, "=once=")
	if err != nil {
		return 0, 0, fmt.Errorf("获取接口流量失败: %w", err)
	}
	
	if len(reply.Re) > 0 {
		re := reply.Re[0].Map
		if rx, ok := re["rx-bits-per-second"]; ok {
			rxRate, _ = strconv.ParseInt(rx, 10, 64)
		}
		if tx, ok := re["tx-bits-per-second"]; ok {
			txRate, _ = strconv.ParseInt(tx, 10, 64)
		}
	}
	
	return rxRate, txRate, nil
}

// parseUptime 解析运行时间字符串
// 格式: 1w2d3h4m5s 或 1d2h3m4s 等
func (c *RouterOSCollector) parseUptime(uptime string) int64 {
	var total int64
	var num string
	
	for _, ch := range uptime {
		switch ch {
		case 'w':
			n, _ := strconv.ParseInt(num, 10, 64)
			total += n * 7 * 24 * 3600
			num = ""
		case 'd':
			n, _ := strconv.ParseInt(num, 10, 64)
			total += n * 24 * 3600
			num = ""
		case 'h':
			n, _ := strconv.ParseInt(num, 10, 64)
			total += n * 3600
			num = ""
		case 'm':
			n, _ := strconv.ParseInt(num, 10, 64)
			total += n * 60
			num = ""
		case 's':
			n, _ := strconv.ParseInt(num, 10, 64)
			total += n
			num = ""
		default:
			if ch >= '0' && ch <= '9' {
				num += string(ch)
			}
		}
	}
	
	return total
}

// wrapError 包装错误信息，提供更友好的错误提示
func (c *RouterOSCollector) wrapError(err error) error {
	errStr := err.Error()
	
	// 连接被拒绝
	if strings.Contains(errStr, "connection refused") {
		return fmt.Errorf("连接被拒绝，请检查端口配置")
	}
	
	// 网络不可达
	if strings.Contains(errStr, "no route to host") || strings.Contains(errStr, "network is unreachable") {
		return fmt.Errorf("无法连接到设备，请检查网络")
	}
	
	// 连接超时
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		return fmt.Errorf("连接超时，请检查网络或设备状态")
	}
	
	// 认证失败
	if strings.Contains(errStr, "cannot log in") || strings.Contains(errStr, "invalid user") {
		return fmt.Errorf("用户名或密码错误")
	}
	
	return err
}

// ConnectionError 连接错误类型
type ConnectionError struct {
	Code    string
	Message string
	Err     error
}

func (e *ConnectionError) Error() string {
	return e.Message
}

func (e *ConnectionError) Unwrap() error {
	return e.Err
}

// 错误码常量
const (
	ErrCodeNetworkUnreachable = "NETWORK_UNREACHABLE"
	ErrCodeConnectionRefused  = "CONNECTION_REFUSED"
	ErrCodeAuthFailed         = "AUTH_FAILED"
	ErrCodeTimeout            = "TIMEOUT"
	ErrCodeAPINotSupported    = "API_NOT_SUPPORTED"
)
