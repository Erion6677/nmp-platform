package proxy

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHTunnel SSH 隧道代理
type SSHTunnel struct {
	Host     string
	Port     int
	Username string
	Password string

	client *ssh.Client
	mu     sync.RWMutex
}

// NewSSHTunnel 创建新的 SSH 隧道
func NewSSHTunnel(host string, port int, username, password string) *SSHTunnel {
	return &SSHTunnel{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}
}

// Connect 建立 SSH 连接
func (t *SSHTunnel) Connect() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 如果已连接，先关闭
	if t.client != nil {
		t.client.Close()
		t.client = nil
	}

	config := &ssh.ClientConfig{
		User: t.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(t.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", t.Host, t.Port)
	client, err := ssh.Dial("tcp", address, config)
	if err != nil {
		return t.wrapError(err)
	}

	t.client = client
	return nil
}

// ConnectWithDialer 通过指定的 Dialer 建立 SSH 连接（用于链式代理）
func (t *SSHTunnel) ConnectWithDialer(dialer Dialer) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 如果已连接，先关闭
	if t.client != nil {
		t.client.Close()
		t.client = nil
	}

	config := &ssh.ClientConfig{
		User: t.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(t.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	address := fmt.Sprintf("%s:%d", t.Host, t.Port)
	
	// 通过父代理建立连接
	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("通过代理连接失败: %w", err)
	}

	// 在已有连接上建立 SSH 连接
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		conn.Close()
		return t.wrapError(err)
	}

	t.client = ssh.NewClient(sshConn, chans, reqs)
	return nil
}

// Dial 通过 SSH 隧道建立连接
func (t *SSHTunnel) Dial(network, addr string) (net.Conn, error) {
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	if client == nil {
		return nil, fmt.Errorf("SSH 隧道未连接")
	}

	conn, err := client.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("通过 SSH 隧道连接失败: %w", err)
	}

	return conn, nil
}

// Close 关闭 SSH 隧道
func (t *SSHTunnel) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.client != nil {
		err := t.client.Close()
		t.client = nil
		return err
	}
	return nil
}

// IsConnected 检查是否已连接
func (t *SSHTunnel) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.client != nil
}

// TestConnection 测试连接
func (t *SSHTunnel) TestConnection() error {
	if err := t.Connect(); err != nil {
		return err
	}
	defer t.Close()

	// 执行简单命令验证连接
	t.mu.RLock()
	client := t.client
	t.mu.RUnlock()

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("创建会话失败: %w", err)
	}
	defer session.Close()

	_, err = session.Output("echo test")
	if err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}

	return nil
}

// GetClient 获取 SSH 客户端（用于执行命令）
func (t *SSHTunnel) GetClient() *ssh.Client {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.client
}

// wrapError 包装错误信息
func (t *SSHTunnel) wrapError(err error) error {
	errStr := err.Error()

	// 连接被拒绝
	if contains(errStr, "connection refused") {
		return fmt.Errorf("连接被拒绝，请检查端口配置")
	}

	// 网络不可达
	if contains(errStr, "no route to host") || contains(errStr, "network is unreachable") {
		return fmt.Errorf("无法连接到代理服务器，请检查网络")
	}

	// 连接超时
	if contains(errStr, "timeout") || contains(errStr, "deadline exceeded") {
		return fmt.Errorf("连接超时，请检查网络或代理服务器状态")
	}

	// 认证失败
	if contains(errStr, "unable to authenticate") || contains(errStr, "no supported methods remain") {
		return fmt.Errorf("用户名或密码错误")
	}

	return err
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if matchLower(s[i:i+len(substr)], substr) {
			return true
		}
	}
	return false
}

func matchLower(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
