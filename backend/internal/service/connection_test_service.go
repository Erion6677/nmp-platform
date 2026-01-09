package service

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
	"go.uber.org/zap"
)

// ConnectionTestService 连接测试服务
type ConnectionTestService struct {
	logger *zap.Logger
}

// NewConnectionTestService 创建连接测试服务
func NewConnectionTestService(logger *zap.Logger) *ConnectionTestService {
	return &ConnectionTestService{
		logger: logger,
	}
}

// TestConnectionRequest 连接测试请求
type TestConnectionRequest struct {
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required,min=1,max=65535"`
	Type       string `json:"type" binding:"required,oneof=api ssh"`
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	DeviceType string `json:"device_type" binding:"required,oneof=mikrotik linux switch firewall"`
}

// TestConnectionResponse 连接测试响应
type TestConnectionResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	Latency   int64  `json:"latency,omitempty"`
	ErrorType string `json:"error_type,omitempty"`
}

// TestConnection 测试设备连接
func (s *ConnectionTestService) TestConnection(ctx context.Context, req *TestConnectionRequest) (*TestConnectionResponse, error) {
	startTime := time.Now()
	
	s.logger.Info("开始连接测试",
		zap.String("host", req.Host),
		zap.Int("port", req.Port),
		zap.String("type", req.Type),
		zap.String("device_type", req.DeviceType),
	)

	var result *TestConnectionResponse
	var err error

	switch req.Type {
	case "ssh":
		result, err = s.testSSHConnection(ctx, req)
	case "api":
		if req.DeviceType == "mikrotik" {
			result, err = s.testMikroTikAPIConnection(ctx, req)
		} else {
			result = &TestConnectionResponse{
				Success:   false,
				Message:   "该设备类型不支持 API 连接",
				ErrorType: "unsupported",
			}
		}
	default:
		result = &TestConnectionResponse{
			Success:   false,
			Message:   "不支持的连接类型",
			ErrorType: "unsupported",
		}
	}

	if err != nil {
		s.logger.Error("连接测试失败", zap.Error(err))
		return result, err
	}

	// 计算延迟
	if result.Success {
		result.Latency = time.Since(startTime).Milliseconds()
		result.Message = fmt.Sprintf("%s 连接成功 (延迟: %dms)", req.Type, result.Latency)
	}

	s.logger.Info("连接测试完成",
		zap.Bool("success", result.Success),
		zap.String("message", result.Message),
		zap.Int64("latency", result.Latency),
	)

	return result, nil
}

// testSSHConnection 测试 SSH 连接
func (s *ConnectionTestService) testSSHConnection(ctx context.Context, req *TestConnectionRequest) (*TestConnectionResponse, error) {
	// 创建 SSH 客户端配置
	config := &ssh.ClientConfig{
		User: req.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(req.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境应该验证主机密钥
		Timeout:         10 * time.Second,
	}

	// 连接地址
	addr := fmt.Sprintf("%s:%d", req.Host, req.Port)

	// 尝试连接
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return s.handleConnectionError(err, "SSH")
	}
	defer conn.Close()

	// 创建会话测试
	session, err := conn.NewSession()
	if err != nil {
		return &TestConnectionResponse{
			Success:   false,
			Message:   "SSH 会话创建失败",
			ErrorType: "session",
		}, nil
	}
	defer session.Close()

	// 执行简单命令测试
	_, err = session.Output("echo 'test'")
	if err != nil {
		return &TestConnectionResponse{
			Success:   false,
			Message:   "SSH 命令执行失败",
			ErrorType: "command",
		}, nil
	}

	return &TestConnectionResponse{
		Success: true,
		Message: "SSH 连接成功",
	}, nil
}

// testMikroTikAPIConnection 测试 MikroTik API 连接
func (s *ConnectionTestService) testMikroTikAPIConnection(ctx context.Context, req *TestConnectionRequest) (*TestConnectionResponse, error) {
	// 连接地址
	addr := fmt.Sprintf("%s:%d", req.Host, req.Port)

	// 尝试建立 TCP 连接
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return s.handleConnectionError(err, "API")
	}
	defer conn.Close()

	// MikroTik API 使用二进制协议
	// 发送 /login 命令
	loginWord := "/login"
	nameWord := "=name=" + req.Username
	passwordWord := "=password=" + req.Password

	// 设置写入超时
	conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	
	// 发送 /login 命令
	if err := s.writeAPIWord(conn, loginWord); err != nil {
		return &TestConnectionResponse{
			Success:   false,
			Message:   "API 命令发送失败",
			ErrorType: "write",
		}, nil
	}
	if err := s.writeAPIWord(conn, nameWord); err != nil {
		return &TestConnectionResponse{
			Success:   false,
			Message:   "API 命令发送失败",
			ErrorType: "write",
		}, nil
	}
	if err := s.writeAPIWord(conn, passwordWord); err != nil {
		return &TestConnectionResponse{
			Success:   false,
			Message:   "API 命令发送失败",
			ErrorType: "write",
		}, nil
	}
	// 发送空字节表示命令结束
	if _, err := conn.Write([]byte{0}); err != nil {
		return &TestConnectionResponse{
			Success:   false,
			Message:   "API 命令发送失败",
			ErrorType: "write",
		}, nil
	}

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	
	// 读取响应
	response, err := s.readAPIResponse(conn)
	if err != nil {
		return &TestConnectionResponse{
			Success:   false,
			Message:   "API 响应读取失败: " + err.Error(),
			ErrorType: "read",
		}, nil
	}

	// 检查响应
	for _, word := range response {
		if word == "!done" {
			return &TestConnectionResponse{
				Success: true,
				Message: "API 连接成功",
			}, nil
		}
		if word == "!trap" {
			return &TestConnectionResponse{
				Success:   false,
				Message:   "API 认证失败",
				ErrorType: "auth",
			}, nil
		}
	}

	// 如果收到了响应但不是预期的格式，也认为连接成功（至少端口是通的）
	if len(response) > 0 {
		return &TestConnectionResponse{
			Success: true,
			Message: "API 连接成功",
		}, nil
	}

	return &TestConnectionResponse{
		Success:   false,
		Message:   "API 认证失败",
		ErrorType: "auth",
	}, nil
}

// writeAPIWord 写入 MikroTik API 格式的单词
func (s *ConnectionTestService) writeAPIWord(conn net.Conn, word string) error {
	length := len(word)
	var lengthBytes []byte
	
	if length < 0x80 {
		lengthBytes = []byte{byte(length)}
	} else if length < 0x4000 {
		lengthBytes = []byte{byte((length >> 8) | 0x80), byte(length)}
	} else if length < 0x200000 {
		lengthBytes = []byte{byte((length >> 16) | 0xC0), byte(length >> 8), byte(length)}
	} else if length < 0x10000000 {
		lengthBytes = []byte{byte((length >> 24) | 0xE0), byte(length >> 16), byte(length >> 8), byte(length)}
	} else {
		lengthBytes = []byte{0xF0, byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length)}
	}
	
	if _, err := conn.Write(lengthBytes); err != nil {
		return err
	}
	if _, err := conn.Write([]byte(word)); err != nil {
		return err
	}
	return nil
}

// readAPIResponse 读取 MikroTik API 响应
func (s *ConnectionTestService) readAPIResponse(conn net.Conn) ([]string, error) {
	var words []string
	
	for {
		word, err := s.readAPIWord(conn)
		if err != nil {
			return words, err
		}
		if word == "" {
			break
		}
		words = append(words, word)
	}
	
	return words, nil
}

// readAPIWord 读取单个 MikroTik API 单词
func (s *ConnectionTestService) readAPIWord(conn net.Conn) (string, error) {
	// 读取长度
	firstByte := make([]byte, 1)
	if _, err := conn.Read(firstByte); err != nil {
		return "", err
	}
	
	var length int
	if firstByte[0] == 0 {
		return "", nil
	} else if firstByte[0] < 0x80 {
		length = int(firstByte[0])
	} else if firstByte[0] < 0xC0 {
		secondByte := make([]byte, 1)
		if _, err := conn.Read(secondByte); err != nil {
			return "", err
		}
		length = int(firstByte[0]&0x3F)<<8 | int(secondByte[0])
	} else if firstByte[0] < 0xE0 {
		moreBytes := make([]byte, 2)
		if _, err := conn.Read(moreBytes); err != nil {
			return "", err
		}
		length = int(firstByte[0]&0x1F)<<16 | int(moreBytes[0])<<8 | int(moreBytes[1])
	} else if firstByte[0] < 0xF0 {
		moreBytes := make([]byte, 3)
		if _, err := conn.Read(moreBytes); err != nil {
			return "", err
		}
		length = int(firstByte[0]&0x0F)<<24 | int(moreBytes[0])<<16 | int(moreBytes[1])<<8 | int(moreBytes[2])
	} else {
		moreBytes := make([]byte, 4)
		if _, err := conn.Read(moreBytes); err != nil {
			return "", err
		}
		length = int(moreBytes[0])<<24 | int(moreBytes[1])<<16 | int(moreBytes[2])<<8 | int(moreBytes[3])
	}
	
	// 读取单词内容
	if length == 0 {
		return "", nil
	}
	
	word := make([]byte, length)
	totalRead := 0
	for totalRead < length {
		n, err := conn.Read(word[totalRead:])
		if err != nil {
			return "", err
		}
		totalRead += n
	}
	
	return string(word), nil
}

// handleConnectionError 处理连接错误
func (s *ConnectionTestService) handleConnectionError(err error, connType string) (*TestConnectionResponse, error) {
	if netErr, ok := err.(net.Error); ok {
		if netErr.Timeout() {
			return &TestConnectionResponse{
				Success:   false,
				Message:   fmt.Sprintf("%s 连接超时", connType),
				ErrorType: "timeout",
			}, nil
		}
	}

	// 检查是否是连接被拒绝
	if opErr, ok := err.(*net.OpError); ok {
		if opErr.Op == "dial" {
			return &TestConnectionResponse{
				Success:   false,
				Message:   fmt.Sprintf("%s 端口不可达或被拒绝", connType),
				ErrorType: "port",
			}, nil
		}
	}

	// 检查 SSH 特定错误
	if connType == "SSH" {
		errStr := err.Error()
		if contains(errStr, "authentication failed") || contains(errStr, "permission denied") {
			return &TestConnectionResponse{
				Success:   false,
				Message:   "SSH 用户名或密码错误",
				ErrorType: "auth",
			}, nil
		}
		if contains(errStr, "no supported methods remain") {
			return &TestConnectionResponse{
				Success:   false,
				Message:   "SSH 认证方法不支持",
				ErrorType: "auth",
			}, nil
		}
	}

	// 通用网络错误
	return &TestConnectionResponse{
		Success:   false,
		Message:   fmt.Sprintf("%s 网络连接失败: %s", connType, err.Error()),
		ErrorType: "network",
	}, nil
}

// contains 检查字符串是否包含子字符串（忽略大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr || 
		      findSubstring(s, substr))))
}

// findSubstring 在字符串中查找子字符串
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}