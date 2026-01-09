package collector

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHCollector SSH 采集器
type SSHCollector struct {
	Timeout time.Duration
}

// NewSSHCollector 创建新的 SSH 采集器
func NewSSHCollector(timeout time.Duration) *SSHCollector {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &SSHCollector{
		Timeout: timeout,
	}
}

// Connect 连接到设备
func (c *SSHCollector) Connect(ip string, port int, username, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.Timeout,
	}

	address := fmt.Sprintf("%s:%d", ip, port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return nil, c.wrapError(err)
	}

	return client, nil
}

// TestConnection 测试连接
func (c *SSHCollector) TestConnection(ip string, port int, username, password string) error {
	client, err := c.Connect(ip, port, username, password)
	if err != nil {
		return err
	}
	defer client.Close()

	// 执行简单命令验证连接
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	defer session.Close()

	// 尝试执行 echo 命令
	_, err = session.Output("echo test")
	if err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}

	return nil
}

// runCommand 执行 SSH 命令
func (c *SSHCollector) runCommand(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	err = session.Run(cmd)
	if err != nil {
		// 某些命令可能返回非零退出码但仍有输出
		if stdout.Len() > 0 {
			return stdout.String(), nil
		}
		return "", fmt.Errorf("执行命令失败: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// GetMikroTikSystemInfo 获取 MikroTik 系统信息（通过 SSH）
func (c *SSHCollector) GetMikroTikSystemInfo(client *ssh.Client) (*SystemInfo, error) {
	info := &SystemInfo{}

	// 获取设备名称
	output, err := c.runCommand(client, "/system identity print")
	if err == nil {
		if name := c.parseMikroTikValue(output, "name"); name != "" {
			info.DeviceName = name
		}
	}

	// 获取系统资源信息
	output, err = c.runCommand(client, "/system resource print")
	if err != nil {
		return nil, fmt.Errorf("获取系统资源失败: %w", err)
	}

	// 解析系统资源
	info.CPUCount = c.parseMikroTikInt(output, "cpu-count")
	info.Version = c.parseMikroTikValue(output, "version")
	info.Uptime = c.parseMikroTikUptime(output)
	info.CPUUsage = float64(c.parseMikroTikInt(output, "cpu-load"))
	info.MemoryTotal = c.parseMikroTikInt64(output, "total-memory")
	info.MemoryFree = c.parseMikroTikInt64(output, "free-memory")

	if info.MemoryTotal > 0 {
		usedMem := info.MemoryTotal - info.MemoryFree
		info.MemoryUsage = float64(usedMem) / float64(info.MemoryTotal) * 100
	}

	// 获取授权信息
	output, err = c.runCommand(client, "/system license print")
	if err == nil {
		if level := c.parseMikroTikValue(output, "level"); level != "" {
			info.License = level
		} else if nlevel := c.parseMikroTikValue(output, "nlevel"); nlevel != "" {
			info.License = nlevel
		}
	}

	return info, nil
}


// GetMikroTikInterfaces 获取 MikroTik 接口列表（通过 SSH）
func (c *SSHCollector) GetMikroTikInterfaces(client *ssh.Client) ([]InterfaceInfo, error) {
	output, err := c.runCommand(client, "/interface print terse")
	if err != nil {
		return nil, fmt.Errorf("获取接口列表失败: %w", err)
	}

	interfaces := make([]InterfaceInfo, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		iface := InterfaceInfo{
			Status: "down",
		}

		// 解析接口名称
		nameMatch := regexp.MustCompile(`name=([^\s]+)`).FindStringSubmatch(line)
		if len(nameMatch) > 1 {
			iface.Name = nameMatch[1]
		}

		// 检查接口状态 - MikroTik terse 格式中 R 表示 running
		// 格式: " 0 RS name=..." 或 " 0 R  name=..." 或 " 0    name=..."
		// R 在第一个数字后面的标志位中
		flagsMatch := regexp.MustCompile(`^\s*\d+\s+([A-Z\s]{1,3})\s+name=`).FindStringSubmatch(line)
		if len(flagsMatch) > 1 {
			flags := flagsMatch[1]
			if strings.Contains(flags, "R") {
				iface.Status = "up"
			}
		}

		if iface.Name != "" {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces, nil
}

// GetLinuxSystemInfo 获取 Linux 系统信息
func (c *SSHCollector) GetLinuxSystemInfo(client *ssh.Client) (*SystemInfo, error) {
	info := &SystemInfo{}

	// 获取主机名
	output, err := c.runCommand(client, "hostname")
	if err == nil {
		info.DeviceName = strings.TrimSpace(output)
	}

	// 获取 CPU 核心数
	output, err = c.runCommand(client, "nproc")
	if err == nil {
		info.CPUCount, _ = strconv.Atoi(strings.TrimSpace(output))
	}

	// 获取系统版本
	output, err = c.runCommand(client, "cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'\"' -f2")
	if err == nil && strings.TrimSpace(output) != "" {
		info.Version = strings.TrimSpace(output)
	} else {
		output, err = c.runCommand(client, "uname -r")
		if err == nil {
			info.Version = strings.TrimSpace(output)
		}
	}

	// 获取运行时间
	output, err = c.runCommand(client, "cat /proc/uptime | cut -d' ' -f1")
	if err == nil {
		uptime, _ := strconv.ParseFloat(strings.TrimSpace(output), 64)
		info.Uptime = int64(uptime)
	}

	// 获取 CPU 使用率
	output, err = c.runCommand(client, "top -bn1 | grep 'Cpu(s)' | awk '{print $2}'")
	if err == nil {
		cpuStr := strings.TrimSpace(output)
		cpuStr = strings.Replace(cpuStr, ",", ".", 1)
		info.CPUUsage, _ = strconv.ParseFloat(cpuStr, 64)
	}

	// 获取内存信息
	output, err = c.runCommand(client, "free -b | grep Mem")
	if err == nil {
		fields := strings.Fields(output)
		if len(fields) >= 4 {
			info.MemoryTotal, _ = strconv.ParseInt(fields[1], 10, 64)
			info.MemoryFree, _ = strconv.ParseInt(fields[3], 10, 64)
			if info.MemoryTotal > 0 {
				usedMem := info.MemoryTotal - info.MemoryFree
				info.MemoryUsage = float64(usedMem) / float64(info.MemoryTotal) * 100
			}
		}
	}

	return info, nil
}

// GetLinuxInterfaces 获取 Linux 接口列表
func (c *SSHCollector) GetLinuxInterfaces(client *ssh.Client) ([]InterfaceInfo, error) {
	output, err := c.runCommand(client, "ip -o link show")
	if err != nil {
		return nil, fmt.Errorf("获取接口列表失败: %w", err)
	}

	interfaces := make([]InterfaceInfo, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析格式: 1: lo: <LOOPBACK,UP,LOWER_UP> ...
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}

		iface := InterfaceInfo{
			Name:   strings.TrimSpace(parts[1]),
			Status: "down",
		}

		// 检查接口状态
		if strings.Contains(line, "state UP") || strings.Contains(line, ",UP,") {
			iface.Status = "up"
		}

		if iface.Name != "" && iface.Name != "lo" {
			interfaces = append(interfaces, iface)
		}
	}

	return interfaces, nil
}

// parseMikroTikValue 解析 MikroTik 输出中的值
func (c *SSHCollector) parseMikroTikValue(output, key string) string {
	pattern := regexp.MustCompile(fmt.Sprintf(`%s:\s*(.+)`, key))
	match := pattern.FindStringSubmatch(output)
	if len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// parseMikroTikInt 解析 MikroTik 输出中的整数值
func (c *SSHCollector) parseMikroTikInt(output, key string) int {
	value := c.parseMikroTikValue(output, key)
	if value == "" {
		return 0
	}
	n, _ := strconv.Atoi(value)
	return n
}

// parseMikroTikInt64 解析 MikroTik 输出中的 int64 值
func (c *SSHCollector) parseMikroTikInt64(output, key string) int64 {
	value := c.parseMikroTikValue(output, key)
	if value == "" {
		return 0
	}
	return c.parseMemoryValue(value)
}

// parseMemoryValue 解析内存值（支持 MiB, GiB, KiB 等单位）
func (c *SSHCollector) parseMemoryValue(value string) int64 {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	// 尝试直接解析为数字
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		return n
	}

	// 解析带单位的值
	var multiplier int64 = 1
	var numStr string

	if strings.HasSuffix(value, "GiB") {
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(value, "GiB")
	} else if strings.HasSuffix(value, "MiB") {
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(value, "MiB")
	} else if strings.HasSuffix(value, "KiB") {
		multiplier = 1024
		numStr = strings.TrimSuffix(value, "KiB")
	} else if strings.HasSuffix(value, "GB") {
		multiplier = 1000 * 1000 * 1000
		numStr = strings.TrimSuffix(value, "GB")
	} else if strings.HasSuffix(value, "MB") {
		multiplier = 1000 * 1000
		numStr = strings.TrimSuffix(value, "MB")
	} else if strings.HasSuffix(value, "KB") {
		multiplier = 1000
		numStr = strings.TrimSuffix(value, "KB")
	} else if strings.HasSuffix(value, "B") {
		numStr = strings.TrimSuffix(value, "B")
	} else {
		numStr = value
	}

	numStr = strings.TrimSpace(numStr)
	f, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	return int64(f * float64(multiplier))
}

// parseMikroTikUptime 解析 MikroTik 运行时间
func (c *SSHCollector) parseMikroTikUptime(output string) int64 {
	value := c.parseMikroTikValue(output, "uptime")
	if value == "" {
		return 0
	}

	// 使用 RouterOS 采集器的解析方法
	rosCollector := &RouterOSCollector{}
	return rosCollector.parseUptime(value)
}

// wrapError 包装错误信息
func (c *SSHCollector) wrapError(err error) error {
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
	if strings.Contains(errStr, "unable to authenticate") || strings.Contains(errStr, "no supported methods remain") {
		return fmt.Errorf("用户名或密码错误")
	}

	return err
}
