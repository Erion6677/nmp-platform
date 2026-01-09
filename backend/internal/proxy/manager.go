package proxy

import (
	"fmt"
	"sync"

	"nmp-platform/internal/models"
	"nmp-platform/internal/repository"
)

// Manager 代理管理器
type Manager struct {
	proxyRepo repository.ProxyRepository
	tunnels   map[uint]*SSHTunnel
	socks5    map[uint]*SOCKS5Proxy
	mu        sync.RWMutex
}

// NewManager 创建代理管理器
func NewManager(proxyRepo repository.ProxyRepository) *Manager {
	return &Manager{
		proxyRepo: proxyRepo,
		tunnels:   make(map[uint]*SSHTunnel),
		socks5:    make(map[uint]*SOCKS5Proxy),
	}
}

// GetDialer 获取代理拨号器
func (m *Manager) GetDialer(proxyID uint) (Dialer, error) {
	if proxyID == 0 {
		return NewDirectDialer(), nil
	}

	proxy, err := m.proxyRepo.GetByID(proxyID)
	if err != nil {
		return nil, fmt.Errorf("获取代理配置失败: %w", err)
	}

	if !proxy.Enabled {
		return nil, fmt.Errorf("代理已禁用")
	}

	// 如果有父代理，先获取父代理的 Dialer
	var parentDialer Dialer
	if proxy.ParentProxyID != nil && *proxy.ParentProxyID > 0 {
		parentDialer, err = m.GetDialer(*proxy.ParentProxyID)
		if err != nil {
			return nil, fmt.Errorf("获取父代理失败: %w", err)
		}
	}

	switch proxy.Type {
	case models.ProxyTypeSSH:
		return m.getSSHTunnelDialer(proxy, parentDialer)
	case models.ProxyTypeSOCKS5:
		return m.getSOCKS5Dialer(proxy, parentDialer)
	default:
		return nil, fmt.Errorf("不支持的代理类型: %s", proxy.Type)
	}
}

// getSSHTunnelDialer 获取 SSH 隧道拨号器
func (m *Manager) getSSHTunnelDialer(proxy *models.Proxy, parentDialer Dialer) (Dialer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已有连接
	if tunnel, ok := m.tunnels[proxy.ID]; ok && tunnel.IsConnected() {
		return tunnel, nil
	}

	// 创建新的 SSH 隧道
	tunnel := NewSSHTunnel(proxy.SSHHost, proxy.SSHPort, proxy.SSHUsername, proxy.SSHPassword)

	// 建立连接
	var err error
	if parentDialer != nil {
		err = tunnel.ConnectWithDialer(parentDialer)
	} else {
		err = tunnel.Connect()
	}

	if err != nil {
		return nil, err
	}

	m.tunnels[proxy.ID] = tunnel
	return tunnel, nil
}

// getSOCKS5Dialer 获取 SOCKS5 拨号器
func (m *Manager) getSOCKS5Dialer(proxy *models.Proxy, parentDialer Dialer) (Dialer, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 创建 SOCKS5 代理
	socks5 := NewSOCKS5Proxy(proxy.SOCKS5Host, proxy.SOCKS5Port, proxy.SOCKS5Username, proxy.SOCKS5Password)

	// 如果有父代理，设置父拨号器
	if parentDialer != nil {
		socks5.SetParentDialer(parentDialer)
	}

	m.socks5[proxy.ID] = socks5
	return socks5, nil
}

// TestProxy 测试代理连接
func (m *Manager) TestProxy(proxyID uint) error {
	proxy, err := m.proxyRepo.GetByID(proxyID)
	if err != nil {
		return fmt.Errorf("获取代理配置失败: %w", err)
	}

	// 如果有父代理，先获取父代理的 Dialer
	var parentDialer Dialer
	if proxy.ParentProxyID != nil && *proxy.ParentProxyID > 0 {
		parentDialer, err = m.GetDialer(*proxy.ParentProxyID)
		if err != nil {
			return fmt.Errorf("获取父代理失败: %w", err)
		}
	}

	switch proxy.Type {
	case models.ProxyTypeSSH:
		tunnel := NewSSHTunnel(proxy.SSHHost, proxy.SSHPort, proxy.SSHUsername, proxy.SSHPassword)
		if parentDialer != nil {
			if err := tunnel.ConnectWithDialer(parentDialer); err != nil {
				return err
			}
		} else {
			if err := tunnel.Connect(); err != nil {
				return err
			}
		}
		defer tunnel.Close()
		return nil

	case models.ProxyTypeSOCKS5:
		socks5 := NewSOCKS5Proxy(proxy.SOCKS5Host, proxy.SOCKS5Port, proxy.SOCKS5Username, proxy.SOCKS5Password)
		if parentDialer != nil {
			socks5.SetParentDialer(parentDialer)
		}
		return socks5.TestConnection()

	default:
		return fmt.Errorf("不支持的代理类型: %s", proxy.Type)
	}
}

// CloseProxy 关闭指定代理的连接
func (m *Manager) CloseProxy(proxyID uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if tunnel, ok := m.tunnels[proxyID]; ok {
		if err := tunnel.Close(); err != nil {
			return err
		}
		delete(m.tunnels, proxyID)
	}

	delete(m.socks5, proxyID)
	return nil
}

// CloseAll 关闭所有代理连接
func (m *Manager) CloseAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, tunnel := range m.tunnels {
		tunnel.Close()
	}
	m.tunnels = make(map[uint]*SSHTunnel)
	m.socks5 = make(map[uint]*SOCKS5Proxy)
}

// GetChainedDialer 获取链式代理拨号器（支持多级跳板）
func (m *Manager) GetChainedDialer(proxyID uint) (Dialer, error) {
	return m.GetDialer(proxyID)
}
