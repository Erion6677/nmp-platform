package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 注意：SOCKS5 测试需要一个可用的 SOCKS5 代理服务器
// 如果没有可用的 SOCKS5 代理，这些测试会被跳过

// 测试 SOCKS5 代理配置（如果有可用的 SOCKS5 代理，请修改这些值）
var testSOCKS5Proxy = struct {
	Host     string
	Port     int
	Username string
	Password string
	Available bool
}{
	Host:      "127.0.0.1",
	Port:      1080,
	Username:  "",
	Password:  "",
	Available: false, // 设置为 true 如果有可用的 SOCKS5 代理
}

// TestSOCKS5Proxy_NewSOCKS5Proxy 测试创建 SOCKS5 代理
func TestSOCKS5Proxy_NewSOCKS5Proxy(t *testing.T) {
	proxy := NewSOCKS5Proxy("127.0.0.1", 1080, "user", "pass")
	
	assert.Equal(t, "127.0.0.1", proxy.Host, "主机应该正确设置")
	assert.Equal(t, 1080, proxy.Port, "端口应该正确设置")
	assert.Equal(t, "user", proxy.Username, "用户名应该正确设置")
	assert.Equal(t, "pass", proxy.Password, "密码应该正确设置")
}

// TestSOCKS5Proxy_SetTimeout 测试设置超时
func TestSOCKS5Proxy_SetTimeout(t *testing.T) {
	proxy := NewSOCKS5Proxy("127.0.0.1", 1080, "", "")
	
	// 默认超时应该是 10 秒
	assert.Equal(t, 10*time.Second, proxy.timeout, "默认超时应该是 10 秒")
	
	// 设置新的超时
	proxy.SetTimeout(30 * time.Second)
	assert.Equal(t, 30*time.Second, proxy.timeout, "超时应该被更新")
}

// TestSOCKS5Proxy_SetParentDialer 测试设置父拨号器
func TestSOCKS5Proxy_SetParentDialer(t *testing.T) {
	proxy := NewSOCKS5Proxy("127.0.0.1", 1080, "", "")
	
	// 初始时父拨号器应该为 nil
	assert.Nil(t, proxy.parentDialer, "初始父拨号器应该为 nil")
	
	// 设置父拨号器
	directDialer := NewDirectDialer()
	proxy.SetParentDialer(directDialer)
	assert.NotNil(t, proxy.parentDialer, "父拨号器应该被设置")
}

// TestSOCKS5Proxy_Dial_NoProxy 测试没有代理时的连接（应该失败）
func TestSOCKS5Proxy_Dial_NoProxy(t *testing.T) {
	if testSOCKS5Proxy.Available {
		t.Skip("跳过此测试，因为有可用的 SOCKS5 代理")
	}
	
	proxy := NewSOCKS5Proxy("127.0.0.1", 1080, "", "")
	proxy.SetTimeout(2 * time.Second)
	
	_, err := proxy.Dial("tcp", "8.8.8.8:53")
	assert.Error(t, err, "没有代理时应该返回错误")
}

// TestSOCKS5Proxy_TestConnection_NoProxy 测试没有代理时的连接测试（应该失败）
func TestSOCKS5Proxy_TestConnection_NoProxy(t *testing.T) {
	if testSOCKS5Proxy.Available {
		t.Skip("跳过此测试，因为有可用的 SOCKS5 代理")
	}
	
	proxy := NewSOCKS5Proxy("127.0.0.1", 1080, "", "")
	proxy.SetTimeout(2 * time.Second)
	
	err := proxy.TestConnection()
	assert.Error(t, err, "没有代理时连接测试应该失败")
}

// TestSOCKS5Proxy_Dial_WithProxy 测试有代理时的连接
func TestSOCKS5Proxy_Dial_WithProxy(t *testing.T) {
	if !testSOCKS5Proxy.Available {
		t.Skip("跳过此测试，因为没有可用的 SOCKS5 代理")
	}
	
	proxy := NewSOCKS5Proxy(
		testSOCKS5Proxy.Host,
		testSOCKS5Proxy.Port,
		testSOCKS5Proxy.Username,
		testSOCKS5Proxy.Password,
	)
	
	conn, err := proxy.Dial("tcp", "8.8.8.8:53")
	assert.NoError(t, err, "通过代理连接应该成功")
	if conn != nil {
		conn.Close()
	}
}

// TestSOCKS5Proxy_TestConnection_WithProxy 测试有代理时的连接测试
func TestSOCKS5Proxy_TestConnection_WithProxy(t *testing.T) {
	if !testSOCKS5Proxy.Available {
		t.Skip("跳过此测试，因为没有可用的 SOCKS5 代理")
	}
	
	proxy := NewSOCKS5Proxy(
		testSOCKS5Proxy.Host,
		testSOCKS5Proxy.Port,
		testSOCKS5Proxy.Username,
		testSOCKS5Proxy.Password,
	)
	
	err := proxy.TestConnection()
	assert.NoError(t, err, "代理连接测试应该成功")
}

// TestSOCKS5Proxy_TestConnectionWithTarget 测试使用指定目标的连接测试
func TestSOCKS5Proxy_TestConnectionWithTarget(t *testing.T) {
	if !testSOCKS5Proxy.Available {
		t.Skip("跳过此测试，因为没有可用的 SOCKS5 代理")
	}
	
	proxy := NewSOCKS5Proxy(
		testSOCKS5Proxy.Host,
		testSOCKS5Proxy.Port,
		testSOCKS5Proxy.Username,
		testSOCKS5Proxy.Password,
	)
	
	err := proxy.TestConnectionWithTarget("1.1.1.1:53")
	assert.NoError(t, err, "使用指定目标的连接测试应该成功")
}

// TestSOCKS5Proxy_ParseError 测试错误解析
func TestSOCKS5Proxy_ParseError(t *testing.T) {
	proxy := NewSOCKS5Proxy("127.0.0.1", 1080, "", "")
	
	tests := []struct {
		code     byte
		expected string
	}{
		{0x01, "一般性失败"},
		{0x02, "规则不允许连接"},
		{0x03, "网络不可达"},
		{0x04, "主机不可达"},
		{0x05, "连接被拒绝"},
		{0x06, "TTL 过期"},
		{0x07, "不支持的命令"},
		{0x08, "不支持的地址类型"},
		{0xFF, "未知错误"},
	}
	
	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := proxy.parseError(tt.code)
			assert.Contains(t, err.Error(), tt.expected, "错误消息应该包含预期内容")
		})
	}
}

// TestSOCKS5Proxy_ChainedProxy 测试链式代理（通过 SSH 隧道连接 SOCKS5）
func TestSOCKS5Proxy_ChainedProxy(t *testing.T) {
	if !testSOCKS5Proxy.Available {
		t.Skip("跳过此测试，因为没有可用的 SOCKS5 代理")
	}
	
	// 首先建立 SSH 隧道
	sshTunnel := NewSSHTunnel(
		testSSHProxy.Host,
		testSSHProxy.Port,
		testSSHProxy.Username,
		testSSHProxy.Password,
	)
	
	err := sshTunnel.Connect()
	if err != nil {
		t.Skipf("跳过此测试，因为 SSH 隧道连接失败: %v", err)
	}
	defer sshTunnel.Close()
	
	// 通过 SSH 隧道连接 SOCKS5 代理
	socks5 := NewSOCKS5Proxy(
		testSOCKS5Proxy.Host,
		testSOCKS5Proxy.Port,
		testSOCKS5Proxy.Username,
		testSOCKS5Proxy.Password,
	)
	socks5.SetParentDialer(sshTunnel)
	
	err = socks5.TestConnection()
	assert.NoError(t, err, "链式代理连接测试应该成功")
}
