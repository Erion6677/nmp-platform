package api

import (
	"net/http"
	"strconv"

	"nmp-platform/internal/models"
	"nmp-platform/internal/proxy"
	"nmp-platform/internal/repository"

	"github.com/gin-gonic/gin"
)

// ProxyHandler 代理处理器
type ProxyHandler struct {
	proxyRepo    repository.ProxyRepository
	proxyManager *proxy.Manager
}

// NewProxyHandler 创建新的代理处理器
func NewProxyHandler(proxyRepo repository.ProxyRepository, proxyManager *proxy.Manager) *ProxyHandler {
	return &ProxyHandler{
		proxyRepo:    proxyRepo,
		proxyManager: proxyManager,
	}
}

// CreateProxyRequest 创建代理请求
type CreateProxyRequest struct {
	Name           string           `json:"name" binding:"required"`
	Type           models.ProxyType `json:"type" binding:"required"`
	
	// SSH 代理字段
	SSHHost        string `json:"ssh_host"`
	SSHPort        int    `json:"ssh_port"`
	SSHUsername    string `json:"ssh_username"`
	SSHPassword    string `json:"ssh_password"`
	
	// SOCKS5 代理字段
	SOCKS5Host     string `json:"socks5_host"`
	SOCKS5Port     int    `json:"socks5_port"`
	SOCKS5Username string `json:"socks5_username"`
	SOCKS5Password string `json:"socks5_password"`
	
	// 链式代理
	ParentProxyID  *uint  `json:"parent_proxy_id"`
	
	Enabled        bool   `json:"enabled"`
}

// UpdateProxyRequest 更新代理请求
type UpdateProxyRequest struct {
	Name           string           `json:"name" binding:"required"`
	Type           models.ProxyType `json:"type" binding:"required"`
	
	// SSH 代理字段
	SSHHost        string `json:"ssh_host"`
	SSHPort        int    `json:"ssh_port"`
	SSHUsername    string `json:"ssh_username"`
	SSHPassword    string `json:"ssh_password"`
	
	// SOCKS5 代理字段
	SOCKS5Host     string `json:"socks5_host"`
	SOCKS5Port     int    `json:"socks5_port"`
	SOCKS5Username string `json:"socks5_username"`
	SOCKS5Password string `json:"socks5_password"`
	
	// 链式代理
	ParentProxyID  *uint  `json:"parent_proxy_id"`
	
	Enabled        bool   `json:"enabled"`
}

// ListProxiesRequest 代理列表请求
type ListProxiesRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Type     string `form:"type"`
	Status   string `form:"status"`
	Enabled  *bool  `form:"enabled"`
	Search   string `form:"search"`
}

// TestProxyRequest 测试代理请求
type TestProxyRequest struct {
	Type           models.ProxyType `json:"type" binding:"required"`
	
	// SSH 代理字段
	SSHHost        string `json:"ssh_host"`
	SSHPort        int    `json:"ssh_port"`
	SSHUsername    string `json:"ssh_username"`
	SSHPassword    string `json:"ssh_password"`
	
	// SOCKS5 代理字段
	SOCKS5Host     string `json:"socks5_host"`
	SOCKS5Port     int    `json:"socks5_port"`
	SOCKS5Username string `json:"socks5_username"`
	SOCKS5Password string `json:"socks5_password"`
	
	// 链式代理
	ParentProxyID  *uint  `json:"parent_proxy_id"`
}

// CreateProxy 创建代理
func (h *ProxyHandler) CreateProxy(c *gin.Context) {
	var req CreateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	// 验证代理类型和必填字段
	if err := h.validateProxyRequest(req.Type, req.SSHHost, req.SSHPort, req.SSHUsername, req.SSHPassword,
		req.SOCKS5Host, req.SOCKS5Port); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 创建代理对象
	proxyModel := &models.Proxy{
		Name:           req.Name,
		Type:           req.Type,
		SSHHost:        req.SSHHost,
		SSHPort:        req.SSHPort,
		SSHUsername:    req.SSHUsername,
		SSHPassword:    req.SSHPassword,
		SOCKS5Host:     req.SOCKS5Host,
		SOCKS5Port:     req.SOCKS5Port,
		SOCKS5Username: req.SOCKS5Username,
		SOCKS5Password: req.SOCKS5Password,
		ParentProxyID:  req.ParentProxyID,
		Enabled:        req.Enabled,
		Status:         models.ProxyStatusDisconnected,
	}

	// 设置默认端口
	if proxyModel.Type == models.ProxyTypeSSH && proxyModel.SSHPort == 0 {
		proxyModel.SSHPort = 22
	}
	if proxyModel.Type == models.ProxyTypeSOCKS5 && proxyModel.SOCKS5Port == 0 {
		proxyModel.SOCKS5Port = 1080
	}

	// 创建代理
	if err := h.proxyRepo.Create(proxyModel); err != nil {
		BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    proxyModel,
	})
}

// GetProxy 获取代理详情
func (h *ProxyHandler) GetProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的代理 ID")
		return
	}

	proxyModel, err := h.proxyRepo.GetByID(uint(id))
	if err != nil {
		NotFound(c, err.Error())
		return
	}

	Success(c, proxyModel)
}

// UpdateProxy 更新代理
func (h *ProxyHandler) UpdateProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的代理 ID")
		return
	}

	var req UpdateProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	// 验证代理类型和必填字段
	if err := h.validateProxyRequest(req.Type, req.SSHHost, req.SSHPort, req.SSHUsername, req.SSHPassword,
		req.SOCKS5Host, req.SOCKS5Port); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 获取现有代理
	proxyModel, err := h.proxyRepo.GetByID(uint(id))
	if err != nil {
		NotFound(c, err.Error())
		return
	}

	// 关闭现有连接
	h.proxyManager.CloseProxy(uint(id))

	// 更新代理信息
	proxyModel.Name = req.Name
	proxyModel.Type = req.Type
	proxyModel.SSHHost = req.SSHHost
	proxyModel.SSHPort = req.SSHPort
	proxyModel.SSHUsername = req.SSHUsername
	if req.SSHPassword != "" {
		proxyModel.SSHPassword = req.SSHPassword
	}
	proxyModel.SOCKS5Host = req.SOCKS5Host
	proxyModel.SOCKS5Port = req.SOCKS5Port
	proxyModel.SOCKS5Username = req.SOCKS5Username
	if req.SOCKS5Password != "" {
		proxyModel.SOCKS5Password = req.SOCKS5Password
	}
	proxyModel.ParentProxyID = req.ParentProxyID
	proxyModel.Enabled = req.Enabled

	// 设置默认端口
	if proxyModel.Type == models.ProxyTypeSSH && proxyModel.SSHPort == 0 {
		proxyModel.SSHPort = 22
	}
	if proxyModel.Type == models.ProxyTypeSOCKS5 && proxyModel.SOCKS5Port == 0 {
		proxyModel.SOCKS5Port = 1080
	}

	if err := h.proxyRepo.Update(proxyModel); err != nil {
		BadRequest(c, err.Error())
		return
	}

	Success(c, proxyModel)
}

// DeleteProxy 删除代理
func (h *ProxyHandler) DeleteProxy(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的代理 ID")
		return
	}

	// 关闭连接
	h.proxyManager.CloseProxy(uint(id))

	// 删除代理
	if err := h.proxyRepo.Delete(uint(id)); err != nil {
		BadRequest(c, err.Error())
		return
	}

	SuccessWithMessage(c, nil, "代理删除成功")
}

// ListProxies 获取代理列表
func (h *ProxyHandler) ListProxies(c *gin.Context) {
	var req ListProxiesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		BadRequest(c, "无效的查询参数")
		return
	}

	// 计算偏移量
	offset := (req.Page - 1) * req.PageSize
	if offset < 0 {
		offset = 0
	}

	// 构建过滤条件
	filters := make(map[string]interface{})
	if req.Type != "" {
		filters["type"] = models.ProxyType(req.Type)
	}
	if req.Status != "" {
		filters["status"] = models.ProxyStatus(req.Status)
	}
	if req.Enabled != nil {
		filters["enabled"] = *req.Enabled
	}
	if req.Search != "" {
		filters["search"] = req.Search
	}

	proxies, total, err := h.proxyRepo.List(offset, req.PageSize, filters)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	// 计算分页信息
	totalPages := (int(total) + req.PageSize - 1) / req.PageSize

	Success(c, gin.H{
		"proxies":     proxies,
		"total":       total,
		"page":        req.Page,
		"page_size":   req.PageSize,
		"total_pages": totalPages,
	})
}

// TestProxy 测试代理连接（不保存）
func (h *ProxyHandler) TestProxy(c *gin.Context) {
	var req TestProxyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, "无效的请求格式")
		return
	}

	// 验证代理类型和必填字段
	if err := h.validateProxyRequest(req.Type, req.SSHHost, req.SSHPort, req.SSHUsername, req.SSHPassword,
		req.SOCKS5Host, req.SOCKS5Port); err != nil {
		BadRequest(c, err.Error())
		return
	}

	// 获取父代理的 Dialer（如果有）
	var parentDialer proxy.Dialer
	if req.ParentProxyID != nil && *req.ParentProxyID > 0 {
		var err error
		parentDialer, err = h.proxyManager.GetDialer(*req.ParentProxyID)
		if err != nil {
			BadRequest(c, "获取父代理失败: "+err.Error())
			return
		}
	}

	// 测试连接
	var testErr error
	switch req.Type {
	case models.ProxyTypeSSH:
		port := req.SSHPort
		if port == 0 {
			port = 22
		}
		tunnel := proxy.NewSSHTunnel(req.SSHHost, port, req.SSHUsername, req.SSHPassword)
		if parentDialer != nil {
			testErr = tunnel.ConnectWithDialer(parentDialer)
		} else {
			testErr = tunnel.Connect()
		}
		if testErr == nil {
			tunnel.Close()
		}

	case models.ProxyTypeSOCKS5:
		port := req.SOCKS5Port
		if port == 0 {
			port = 1080
		}
		socks5 := proxy.NewSOCKS5Proxy(req.SOCKS5Host, port, req.SOCKS5Username, req.SOCKS5Password)
		if parentDialer != nil {
			socks5.SetParentDialer(parentDialer)
		}
		testErr = socks5.TestConnection()
	}

	if testErr != nil {
		Success(c, gin.H{
			"connected": false,
			"error":     testErr.Error(),
		})
		return
	}

	Success(c, gin.H{
		"connected": true,
	})
}

// TestProxyByID 测试已保存代理的连接
func (h *ProxyHandler) TestProxyByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		BadRequest(c, "无效的代理 ID")
		return
	}

	// 测试代理连接
	testErr := h.proxyManager.TestProxy(uint(id))

	// 更新代理状态
	if testErr != nil {
		h.proxyRepo.UpdateStatus(uint(id), models.ProxyStatusError, testErr.Error())
		Success(c, gin.H{
			"connected": false,
			"error":     testErr.Error(),
		})
		return
	}

	h.proxyRepo.UpdateStatus(uint(id), models.ProxyStatusConnected, "")
	Success(c, gin.H{
		"connected": true,
	})
}

// validateProxyRequest 验证代理请求
func (h *ProxyHandler) validateProxyRequest(proxyType models.ProxyType, sshHost string, sshPort int, sshUsername, sshPassword string,
	socks5Host string, socks5Port int) error {
	switch proxyType {
	case models.ProxyTypeSSH:
		if sshHost == "" {
			return &ValidationError{Field: "ssh_host", Message: "SSH 主机地址不能为空"}
		}
		if sshUsername == "" {
			return &ValidationError{Field: "ssh_username", Message: "SSH 用户名不能为空"}
		}
		if sshPassword == "" {
			return &ValidationError{Field: "ssh_password", Message: "SSH 密码不能为空"}
		}
	case models.ProxyTypeSOCKS5:
		if socks5Host == "" {
			return &ValidationError{Field: "socks5_host", Message: "SOCKS5 主机地址不能为空"}
		}
	default:
		return &ValidationError{Field: "type", Message: "不支持的代理类型"}
	}
	return nil
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// RegisterRoutes 注册代理相关路由
func (h *ProxyHandler) RegisterRoutes(router *gin.RouterGroup) {
	proxies := router.Group("/proxies")
	{
		proxies.POST("", h.CreateProxy)
		proxies.GET("", h.ListProxies)
		proxies.GET("/:id", h.GetProxy)
		proxies.PUT("/:id", h.UpdateProxy)
		proxies.DELETE("/:id", h.DeleteProxy)
		
		// 连接测试
		proxies.POST("/test", h.TestProxy)         // 测试新代理连接（不需要先保存）
		proxies.POST("/:id/test", h.TestProxyByID) // 测试已保存代理连接
	}
}
