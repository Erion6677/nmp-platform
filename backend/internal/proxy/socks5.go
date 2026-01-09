package proxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"
)

// SOCKS5 协议常量
const (
	socks5Version = 0x05

	// 认证方法
	socks5AuthNone     = 0x00
	socks5AuthPassword = 0x02
	socks5AuthNoAccept = 0xFF

	// 命令
	socks5CmdConnect = 0x01

	// 地址类型
	socks5AddrIPv4   = 0x01
	socks5AddrDomain = 0x03
	socks5AddrIPv6   = 0x04

	// 响应状态
	socks5RespSuccess = 0x00
)

// SOCKS5Proxy SOCKS5 代理
type SOCKS5Proxy struct {
	Host     string
	Port     int
	Username string
	Password string

	parentDialer Dialer
	timeout      time.Duration
}

// NewSOCKS5Proxy 创建新的 SOCKS5 代理
func NewSOCKS5Proxy(host string, port int, username, password string) *SOCKS5Proxy {
	return &SOCKS5Proxy{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		timeout:  10 * time.Second,
	}
}

// SetParentDialer 设置父拨号器（用于链式代理）
func (p *SOCKS5Proxy) SetParentDialer(dialer Dialer) {
	p.parentDialer = dialer
}

// SetTimeout 设置超时时间
func (p *SOCKS5Proxy) SetTimeout(timeout time.Duration) {
	p.timeout = timeout
}

// Dial 通过 SOCKS5 代理建立连接
func (p *SOCKS5Proxy) Dial(network, addr string) (net.Conn, error) {
	// 连接到 SOCKS5 代理服务器
	proxyAddr := fmt.Sprintf("%s:%d", p.Host, p.Port)
	
	var conn net.Conn
	var err error
	
	if p.parentDialer != nil {
		conn, err = p.parentDialer.Dial("tcp", proxyAddr)
	} else {
		conn, err = net.DialTimeout("tcp", proxyAddr, p.timeout)
	}
	
	if err != nil {
		return nil, fmt.Errorf("连接 SOCKS5 代理失败: %w", err)
	}

	// 设置超时
	if err := conn.SetDeadline(time.Now().Add(p.timeout)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("设置超时失败: %w", err)
	}

	// 执行 SOCKS5 握手
	if err := p.handshake(conn); err != nil {
		conn.Close()
		return nil, err
	}

	// 发送连接请求
	if err := p.connect(conn, addr); err != nil {
		conn.Close()
		return nil, err
	}

	// 清除超时
	if err := conn.SetDeadline(time.Time{}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("清除超时失败: %w", err)
	}

	return conn, nil
}

// handshake 执行 SOCKS5 握手
func (p *SOCKS5Proxy) handshake(conn net.Conn) error {
	// 构建认证方法请求
	var methods []byte
	if p.Username != "" && p.Password != "" {
		methods = []byte{socks5AuthNone, socks5AuthPassword}
	} else {
		methods = []byte{socks5AuthNone}
	}

	// 发送版本和支持的认证方法
	req := make([]byte, 2+len(methods))
	req[0] = socks5Version
	req[1] = byte(len(methods))
	copy(req[2:], methods)

	if _, err := conn.Write(req); err != nil {
		return fmt.Errorf("发送握手请求失败: %w", err)
	}

	// 读取服务器响应
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return fmt.Errorf("读取握手响应失败: %w", err)
	}

	if resp[0] != socks5Version {
		return errors.New("不支持的 SOCKS 版本")
	}

	// 处理认证
	switch resp[1] {
	case socks5AuthNone:
		// 无需认证
		return nil
	case socks5AuthPassword:
		// 用户名密码认证
		return p.authenticate(conn)
	case socks5AuthNoAccept:
		return errors.New("服务器不接受任何认证方法")
	default:
		return fmt.Errorf("不支持的认证方法: %d", resp[1])
	}
}

// authenticate 执行用户名密码认证
func (p *SOCKS5Proxy) authenticate(conn net.Conn) error {
	if p.Username == "" || p.Password == "" {
		return errors.New("需要用户名和密码")
	}

	// 构建认证请求
	// 格式: VER(1) + ULEN(1) + UNAME(1-255) + PLEN(1) + PASSWD(1-255)
	req := make([]byte, 3+len(p.Username)+len(p.Password))
	req[0] = 0x01 // 认证子协议版本
	req[1] = byte(len(p.Username))
	copy(req[2:], p.Username)
	req[2+len(p.Username)] = byte(len(p.Password))
	copy(req[3+len(p.Username):], p.Password)

	if _, err := conn.Write(req); err != nil {
		return fmt.Errorf("发送认证请求失败: %w", err)
	}

	// 读取认证响应
	resp := make([]byte, 2)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return fmt.Errorf("读取认证响应失败: %w", err)
	}

	if resp[1] != 0x00 {
		return errors.New("用户名或密码错误")
	}

	return nil
}

// connect 发送连接请求
func (p *SOCKS5Proxy) connect(conn net.Conn, addr string) error {
	// 解析目标地址
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("解析地址失败: %w", err)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return fmt.Errorf("解析端口失败: %w", err)
	}

	// 构建连接请求
	// 格式: VER(1) + CMD(1) + RSV(1) + ATYP(1) + DST.ADDR(variable) + DST.PORT(2)
	var req []byte

	// 检查是否为 IP 地址
	ip := net.ParseIP(host)
	if ip != nil {
		if ip4 := ip.To4(); ip4 != nil {
			// IPv4
			req = make([]byte, 10)
			req[0] = socks5Version
			req[1] = socks5CmdConnect
			req[2] = 0x00 // RSV
			req[3] = socks5AddrIPv4
			copy(req[4:8], ip4)
			binary.BigEndian.PutUint16(req[8:], uint16(port))
		} else {
			// IPv6
			req = make([]byte, 22)
			req[0] = socks5Version
			req[1] = socks5CmdConnect
			req[2] = 0x00 // RSV
			req[3] = socks5AddrIPv6
			copy(req[4:20], ip.To16())
			binary.BigEndian.PutUint16(req[20:], uint16(port))
		}
	} else {
		// 域名
		if len(host) > 255 {
			return errors.New("域名过长")
		}
		req = make([]byte, 7+len(host))
		req[0] = socks5Version
		req[1] = socks5CmdConnect
		req[2] = 0x00 // RSV
		req[3] = socks5AddrDomain
		req[4] = byte(len(host))
		copy(req[5:], host)
		binary.BigEndian.PutUint16(req[5+len(host):], uint16(port))
	}

	if _, err := conn.Write(req); err != nil {
		return fmt.Errorf("发送连接请求失败: %w", err)
	}

	// 读取响应
	resp := make([]byte, 4)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return fmt.Errorf("读取连接响应失败: %w", err)
	}

	if resp[0] != socks5Version {
		return errors.New("不支持的 SOCKS 版本")
	}

	if resp[1] != socks5RespSuccess {
		return p.parseError(resp[1])
	}

	// 读取绑定地址（我们不需要使用它，但必须读取）
	switch resp[3] {
	case socks5AddrIPv4:
		_, err = io.ReadFull(conn, make([]byte, 4+2)) // IPv4 + Port
	case socks5AddrIPv6:
		_, err = io.ReadFull(conn, make([]byte, 16+2)) // IPv6 + Port
	case socks5AddrDomain:
		lenBuf := make([]byte, 1)
		if _, err = io.ReadFull(conn, lenBuf); err == nil {
			_, err = io.ReadFull(conn, make([]byte, int(lenBuf[0])+2)) // Domain + Port
		}
	default:
		return fmt.Errorf("不支持的地址类型: %d", resp[3])
	}

	if err != nil {
		return fmt.Errorf("读取绑定地址失败: %w", err)
	}

	return nil
}

// parseError 解析 SOCKS5 错误码
func (p *SOCKS5Proxy) parseError(code byte) error {
	switch code {
	case 0x01:
		return errors.New("SOCKS5: 一般性失败")
	case 0x02:
		return errors.New("SOCKS5: 规则不允许连接")
	case 0x03:
		return errors.New("SOCKS5: 网络不可达")
	case 0x04:
		return errors.New("SOCKS5: 主机不可达")
	case 0x05:
		return errors.New("SOCKS5: 连接被拒绝")
	case 0x06:
		return errors.New("SOCKS5: TTL 过期")
	case 0x07:
		return errors.New("SOCKS5: 不支持的命令")
	case 0x08:
		return errors.New("SOCKS5: 不支持的地址类型")
	default:
		return fmt.Errorf("SOCKS5: 未知错误 (%d)", code)
	}
}

// TestConnection 测试连接
func (p *SOCKS5Proxy) TestConnection() error {
	// 尝试通过代理连接到一个公共地址来测试
	// 使用 Google DNS 作为测试目标
	conn, err := p.Dial("tcp", "8.8.8.8:53")
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// TestConnectionWithTarget 使用指定目标测试连接
func (p *SOCKS5Proxy) TestConnectionWithTarget(target string) error {
	conn, err := p.Dial("tcp", target)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
