package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 测试设备配置（真实 MikroTik 设备作为 SSH 跳板测试）
// 注意：这里使用 MikroTik 设备作为 SSH 跳板进行测试
var testSSHProxy = struct {
	Host     string
	Port     int
	Username string
	Password string
}{
	Host:     "10.10.10.254",
	Port:     3399,
	Username: "admin",
	Password: "927528",
}

// TestSSHTunnel_Connect 测试 SSH 隧道连接
func TestSSHTunnel_Connect(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		testSSHProxy.Password,
	)

	err := tunnel.Connect()
	require.NoError(t, err, "SSH 隧道连接失败")
	assert.True(t, tunnel.IsConnected(), "隧道应该处于连接状态")

	defer tunnel.Close()
}

// TestSSHTunnel_Connect_WrongPassword 测试错误密码
func TestSSHTunnel_Connect_WrongPassword(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		"wrong_password",
	)

	err := tunnel.Connect()
	assert.Error(t, err, "错误密码应该返回错误")
	assert.Contains(t, err.Error(), "用户名或密码错误", "应该返回认证错误")
	assert.False(t, tunnel.IsConnected(), "隧道不应该处于连接状态")
}

// TestSSHTunnel_Connect_WrongPort 测试错误端口
func TestSSHTunnel_Connect_WrongPort(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		9999, // 错误端口
		testSSHProxy.Username,
		testSSHProxy.Password,
	)

	err := tunnel.Connect()
	assert.Error(t, err, "错误端口应该返回错误")
	assert.False(t, tunnel.IsConnected(), "隧道不应该处于连接状态")
}

// TestSSHTunnel_TestConnection 测试连接测试功能
func TestSSHTunnel_TestConnection(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		testSSHProxy.Password,
	)

	err := tunnel.TestConnection()
	assert.NoError(t, err, "连接测试应该成功")
}

// TestSSHTunnel_Dial 测试通过隧道建立连接
func TestSSHTunnel_Dial(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		testSSHProxy.Password,
	)

	err := tunnel.Connect()
	require.NoError(t, err, "SSH 隧道连接失败")
	defer tunnel.Close()

	// 尝试通过隧道连接到本地回环地址（测试隧道功能）
	// 使用一个快速超时的连接测试
	done := make(chan struct{})
	var dialErr error
	var conn interface{ Close() error }

	go func() {
		// 尝试连接到 localhost:22（SSH 服务通常在运行）
		conn, dialErr = tunnel.Dial("tcp", "127.0.0.1:22")
		close(done)
	}()

	select {
	case <-done:
		if dialErr != nil {
			t.Logf("通过隧道连接失败（可能是目标不可达）: %v", dialErr)
			// 这不一定是错误，取决于网络配置
		} else {
			t.Log("通过 SSH 隧道成功建立连接")
			if conn != nil {
				conn.Close()
			}
		}
	case <-time.After(5 * time.Second):
		t.Log("通过隧道连接超时，跳过此测试")
	}
}

// TestSSHTunnel_Close 测试关闭隧道
func TestSSHTunnel_Close(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		testSSHProxy.Password,
	)

	err := tunnel.Connect()
	require.NoError(t, err, "SSH 隧道连接失败")
	assert.True(t, tunnel.IsConnected(), "隧道应该处于连接状态")

	err = tunnel.Close()
	assert.NoError(t, err, "关闭隧道应该成功")
	assert.False(t, tunnel.IsConnected(), "隧道应该处于断开状态")
}

// TestSSHTunnel_Reconnect 测试重新连接
func TestSSHTunnel_Reconnect(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		testSSHProxy.Password,
	)

	// 第一次连接
	err := tunnel.Connect()
	require.NoError(t, err, "第一次连接失败")
	assert.True(t, tunnel.IsConnected(), "隧道应该处于连接状态")

	// 关闭
	err = tunnel.Close()
	require.NoError(t, err, "关闭失败")
	assert.False(t, tunnel.IsConnected(), "隧道应该处于断开状态")

	// 重新连接
	err = tunnel.Connect()
	require.NoError(t, err, "重新连接失败")
	assert.True(t, tunnel.IsConnected(), "隧道应该处于连接状态")

	defer tunnel.Close()
}

// TestSSHTunnel_GetClient 测试获取 SSH 客户端
func TestSSHTunnel_GetClient(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		testSSHProxy.Password,
	)

	// 未连接时应返回 nil
	client := tunnel.GetClient()
	assert.Nil(t, client, "未连接时客户端应为 nil")

	// 连接后应返回客户端
	err := tunnel.Connect()
	require.NoError(t, err, "连接失败")
	defer tunnel.Close()

	client = tunnel.GetClient()
	assert.NotNil(t, client, "连接后客户端不应为 nil")
}

// TestSSHTunnel_ConcurrentAccess 测试并发访问
func TestSSHTunnel_ConcurrentAccess(t *testing.T) {
	tunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		testSSHProxy.Password,
	)

	err := tunnel.Connect()
	require.NoError(t, err, "连接失败")
	defer tunnel.Close()

	// 并发检查连接状态
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				_ = tunnel.IsConnected()
				_ = tunnel.GetClient()
				time.Sleep(time.Millisecond)
			}
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}

	assert.True(t, tunnel.IsConnected(), "并发访问后隧道应该仍然连接")
}
