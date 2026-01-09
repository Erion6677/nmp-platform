package proxy

import (
	"net"
)

// Dialer 代理拨号器接口
type Dialer interface {
	// Dial 建立网络连接
	Dial(network, addr string) (net.Conn, error)
}

// DirectDialer 直接连接拨号器（不使用代理）
type DirectDialer struct{}

// NewDirectDialer 创建直接连接拨号器
func NewDirectDialer() *DirectDialer {
	return &DirectDialer{}
}

// Dial 直接建立网络连接
func (d *DirectDialer) Dial(network, addr string) (net.Conn, error) {
	return net.Dial(network, addr)
}
