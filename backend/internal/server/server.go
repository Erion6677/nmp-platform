package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"

	"nmp-platform/internal/api"
	"nmp-platform/internal/auth"
	"nmp-platform/internal/backup"
	"nmp-platform/internal/config"
	"nmp-platform/internal/database"
	"nmp-platform/internal/health"
	"nmp-platform/internal/influxdb"
	"nmp-platform/internal/marketplace"
	"nmp-platform/internal/proxy"
	"nmp-platform/internal/redis"
	"nmp-platform/internal/repository"
	"nmp-platform/internal/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server HTTP服务器
type Server struct {
	config         *config.Config
	logger         *zap.Logger
	router         *gin.Engine
	httpServer     *http.Server
	redisClient    *redis.Client
	influxdbClient *influxdb.Client
	healthChecker  *health.HealthChecker
	authService    *auth.AuthService
	authHandler    *auth.AuthHandler
	rbacService    *auth.RBACService
	rbacHandler    *auth.RBACHandler
	adminHandler   *auth.AdminHandler
	startTime      time.Time  // 服务器启动时间
	
	// 设备权限检查器
	devicePermChecker *auth.DevicePermissionChecker
	
	// 设备管理相关
	deviceService      service.DeviceService
	tagService         service.TagService
	deviceGroupService service.DeviceGroupService
	deviceHandler      *api.DeviceHandler
	tagHandler         *api.TagHandler
	deviceGroupHandler *api.DeviceGroupHandler
	
	// 数据接收相关
	dataReceiverService *service.DataReceiverService
	dataReceiverHandler *api.DataReceiverHandler
	
	// 数据存储管理相关
	dataCompressionService *service.DataCompressionService
	dataStorageHandler     *api.DataStorageHandler
	
	// 数据查询相关
	dataQueryService *service.DataQueryService
	dataQueryHandler *api.DataQueryHandler
	
	// 连接测试相关
	connectionTestService *service.ConnectionTestService
	connectionTestHandler *api.ConnectionTestHandler
	
	// Ping 目标管理相关
	pingTargetHandler *api.PingTargetHandler
	
	// 系统设置相关
	settingsRepo    repository.SettingsRepository
	settingsHandler *api.SettingsHandler
	
	// 采集器管理相关
	collectorHandler *api.CollectorHandler
	
	// 代理管理相关
	proxyHandler *api.ProxyHandler
	
	// 系统备份相关
	backupService *backup.Service
	backupHandler *api.SystemBackupHandler
	
	// 插件市场相关
	mp                 *marketplace.Marketplace
	marketplaceHandler *api.MarketplaceHandler
}

// New 创建新的服务器实例
func New(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	// 设置Gin模式
	gin.SetMode(cfg.Server.Mode)

	// 连接数据库
	_, err := database.Connect(&cfg.Database)
	if err != nil {
		return nil, err
	}

	// 执行数据库迁移
	if err := database.Migrate(database.DB); err != nil {
		logger.Error("Failed to migrate database", zap.Error(err))
		return nil, err
	}

	// 连接Redis
	redisClient, err := redis.Connect(&cfg.Redis)
	if err != nil {
		logger.Error("Failed to connect to Redis", zap.Error(err))
		return nil, err
	}

	// 连接InfluxDB
	influxdbClient, err := influxdb.Connect(&cfg.InfluxDB)
	if err != nil {
		logger.Error("Failed to connect to InfluxDB", zap.Error(err))
		return nil, err
	}

	// 创建健康检查器
	healthChecker := health.NewHealthChecker(redisClient, influxdbClient)

	// 创建用户仓库
	userRepo := repository.NewUserRepository(database.DB)

	// 创建角色和权限仓库
	roleRepo := repository.NewRoleRepository(database.DB)
	permRepo := repository.NewPermissionRepository(database.DB)

	// 创建认证服务
	authService := auth.NewAuthService(userRepo, cfg.Auth.JWTSecret, cfg.Auth.TokenExpiry)

	// 创建RBAC服务
	rbacService, err := auth.NewRBACService(database.DB, userRepo, roleRepo, permRepo)
	if err != nil {
		logger.Error("Failed to create RBAC service", zap.Error(err))
		return nil, err
	}

	// 创建认证处理器
	authHandler := auth.NewAuthHandler(authService)

	// 创建RBAC处理器
	rbacHandler := auth.NewRBACHandler(rbacService)

	// 创建管理员处理器（带仓库依赖）
	adminHandler := auth.NewAdminHandlerWithRepos(
		authService,
		rbacService,
		userRepo,
		roleRepo,
		permRepo,
		service.GetPasswordService(),
	)

	// 创建设备管理相关仓库
	deviceRepo := repository.NewDeviceRepository(database.DB)
	interfaceRepo := repository.NewInterfaceRepository(database.DB)
	tagRepo := repository.NewTagRepository(database.DB)
	deviceGroupRepo := repository.NewDeviceGroupRepository(database.DB)

	// 创建设备管理相关服务
	deviceService := service.NewDeviceService(deviceRepo, interfaceRepo, tagRepo, deviceGroupRepo)
	tagService := service.NewTagService(tagRepo)
	deviceGroupService := service.NewDeviceGroupService(deviceGroupRepo)

	// 创建采集器仓库（提前创建，供数据接收服务使用）
	collectorRepo := repository.NewCollectorRepository(database.DB)

	// 创建数据接收服务和处理器（使用带采集器仓库的版本，以便更新推送统计）
	influxAdapter := influxdb.NewServiceAdapter(influxdbClient)
	dataReceiverService := service.NewDataReceiverServiceWithCollector(influxAdapter, redisClient, deviceRepo, collectorRepo)
	dataReceiverHandler := api.NewDataReceiverHandler(dataReceiverService)

	// 创建数据压缩服务和处理器
	dataCompressionService := service.NewDataCompressionService(influxAdapter, redisClient)
	dataStorageHandler := api.NewDataStorageHandler(dataCompressionService)

	// 创建数据查询服务和处理器
	dataQueryService := service.NewDataQueryService(influxAdapter, redisClient)
	dataQueryHandler := api.NewDataQueryHandler(dataQueryService)

	// 创建连接测试服务和处理器
	connectionTestService := service.NewConnectionTestService(logger)
	connectionTestHandler := api.NewConnectionTestHandler(connectionTestService, logger)

	// 创建 Ping 目标仓库
	pingTargetRepo := repository.NewPingTargetRepository(database.DB)

	// 创建系统设置仓库和处理器
	settingsRepo := repository.NewSettingsRepository(database.DB)
	settingsHandler := api.NewSettingsHandler(settingsRepo)
	
	// 初始化默认设置
	if err := settingsRepo.InitDefaults(); err != nil {
		logger.Warn("Failed to initialize default settings", zap.Error(err))
	}

	// 创建数据清理服务（带 Redis 支持）
	dataCleanupService := service.NewDataCleanupServiceWithRedis(
		influxAdapter,
		redisClient,
		settingsRepo,
		deviceRepo,
		&service.DataCleanupConfig{
			Bucket: cfg.InfluxDB.Bucket,
			Org:    cfg.InfluxDB.Org,
		},
	)
	
	// 构建服务器 URL（用于采集器脚本）
	// 优先使用配置的 public_url，否则尝试自动检测
	serverURL := cfg.Server.PublicURL
	if serverURL == "" {
		// 如果没有配置 public_url，尝试获取本机 IP
		serverURL = fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)
		if cfg.Server.Host == "" || cfg.Server.Host == "0.0.0.0" {
			// 尝试获取本机的非回环 IP 地址
			if localIP := getLocalIP(); localIP != "" {
				serverURL = fmt.Sprintf("http://%s:%d", localIP, cfg.Server.Port)
			} else {
				serverURL = fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
			}
		}
	}
	
	// 创建设备管理相关处理器（带数据清理服务和自动重新部署功能）
	deviceHandler := api.NewDeviceHandlerFull(
		deviceService,
		tagService,
		deviceGroupService,
		dataCleanupService,
		interfaceRepo,
		collectorRepo,
		deviceRepo,
		pingTargetRepo,
		serverURL,
	)
	tagHandler := api.NewTagHandler(tagService)
	deviceGroupHandler := api.NewDeviceGroupHandler(deviceGroupService, deviceService)
	
	// 创建 Ping 目标处理器（带数据清理服务和自动重新部署功能）
	pingTargetHandler := api.NewPingTargetHandlerFull(
		pingTargetRepo,
		deviceRepo,
		interfaceRepo,
		collectorRepo,
		dataCleanupService,
		serverURL,
	)
	
	// 创建采集器处理器
	collectorHandler := api.NewCollectorHandler(
		collectorRepo,
		settingsRepo,
		deviceService,
		interfaceRepo,
		pingTargetRepo,
		dataCleanupService,
		serverURL,
	)

	// 创建代理管理相关
	proxyRepo := repository.NewProxyRepository(database.DB)
	proxyManager := proxy.NewManager(proxyRepo)
	proxyHandler := api.NewProxyHandler(proxyRepo, proxyManager)

	// 创建系统备份服务和处理器
	backupConfig := &backup.BackupConfig{
		BackupDir:    "/opt/nmp/backups",
		MaxBackups:   10,
		PostgresHost: cfg.Database.Host,
		PostgresPort: cfg.Database.Port,
		PostgresDB:   cfg.Database.Database,
		PostgresUser: cfg.Database.Username,
		PostgresPass: cfg.Database.Password,
		InfluxDBURL:  cfg.InfluxDB.URL,
		InfluxDBOrg:  cfg.InfluxDB.Org,
		InfluxDBToken: cfg.InfluxDB.Token,
		ConfigDir:    "./configs",
		PluginsDir:   cfg.Plugins.Directory,
	}
	backupService := backup.NewService(backupConfig, logger)
	backupHandler := api.NewSystemBackupHandler(backupService, logger)

	// 创建插件市场服务
	// 插件注册表从 GitHub 获取
	marketplaceConfig := &marketplace.MarketplaceConfig{
		RegistryURL: "https://raw.githubusercontent.com/Erion6677/nmp-plugins/main/registry.json",
		PluginsDir:  cfg.Plugins.Directory,
		CacheDir:    "/tmp/nmp-plugins-cache",
	}
	mp := marketplace.NewMarketplace(marketplaceConfig, logger)
	marketplaceHandler := api.NewMarketplaceHandler(mp, logger)

	// 创建设备权限检查器
	devicePermChecker := auth.NewDevicePermissionChecker(rbacService, permRepo, userRepo)

	// 创建路由器
	router := gin.New()

	// 添加中间件
	router.Use(RecoveryMiddleware(logger))
	router.Use(LoggerMiddleware(logger))
	router.Use(ErrorHandlerMiddleware(logger))
	router.Use(CORSMiddleware())
	router.Use(SecurityMiddleware())
	
	// 在生产环境中启用速率限制
	if cfg.Server.Mode == "release" {
		router.Use(RateLimitMiddleware())
	}
	
	// 添加超时中间件
	router.Use(TimeoutMiddleware(cfg.Server.ReadTimeout))

	server := &Server{
		config:         cfg,
		logger:         logger,
		router:         router,
		redisClient:    redisClient,
		influxdbClient: influxdbClient,
		healthChecker:  healthChecker,
		authService:    authService,
		authHandler:    authHandler,
		rbacService:    rbacService,
		rbacHandler:    rbacHandler,
		adminHandler:   adminHandler,
		startTime:      time.Now(), // 记录服务器启动时间
		
		// 设备权限检查器
		devicePermChecker: devicePermChecker,
		
		// 设备管理相关
		deviceService:      deviceService,
		tagService:         tagService,
		deviceGroupService: deviceGroupService,
		deviceHandler:      deviceHandler,
		tagHandler:         tagHandler,
		deviceGroupHandler: deviceGroupHandler,
		
		// 数据接收相关
		dataReceiverService: dataReceiverService,
		dataReceiverHandler: dataReceiverHandler,
		
		// 数据存储管理相关
		dataCompressionService: dataCompressionService,
		dataStorageHandler:     dataStorageHandler,
		
		// 数据查询相关
		dataQueryService: dataQueryService,
		dataQueryHandler: dataQueryHandler,
		
		// 连接测试相关
		connectionTestService: connectionTestService,
		connectionTestHandler: connectionTestHandler,
		
		// Ping 目标管理相关
		pingTargetHandler: pingTargetHandler,
		
		// 系统设置相关
		settingsRepo:    settingsRepo,
		settingsHandler: settingsHandler,
		
		// 采集器管理相关
		collectorHandler: collectorHandler,
		
		// 代理管理相关
		proxyHandler: proxyHandler,
		
		// 系统备份相关
		backupService: backupService,
		backupHandler: backupHandler,
		
		// 插件市场相关
		mp:                 mp,
		marketplaceHandler: marketplaceHandler,
	}

	// 设置路由
	server.setupRoutes()

	// 创建HTTP服务器
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  time.Second * 60,
	}

	return server, nil
}

// Router 返回Gin路由器
func (s *Server) Router() *gin.Engine {
	return s.router
}

// Start 启动HTTP服务器
func (s *Server) Start() error {
	s.logger.Info("Starting HTTP server",
		zap.String("address", s.httpServer.Addr),
		zap.String("mode", s.config.Server.Mode),
	)
	
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("Failed to start HTTP server", zap.Error(err))
		return err
	}
	
	return nil
}

// Shutdown 优雅关闭HTTP服务器
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Failed to shutdown HTTP server", zap.Error(err))
		return err
	}
	
	return nil
}

// Close 关闭服务器资源
func (s *Server) Close() error {
	s.logger.Info("Closing server resources")
	
	// 关闭InfluxDB连接
	if s.influxdbClient != nil {
		s.influxdbClient.Close()
	}
	
	// 关闭Redis连接
	if s.redisClient != nil {
		s.redisClient.Close()
	}
	
	// 关闭数据库连接
	if err := database.Close(); err != nil {
		s.logger.Error("Failed to close database connection", zap.Error(err))
		return err
	}
	
	return nil
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 根路径
	s.router.GET("/", s.index)
	
	// 健康检查
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/health/detailed", s.detailedHealthCheck)
	
	// 系统信息
	s.router.GET("/version", s.version)
	s.router.GET("/metrics", s.metrics)

	// 设备数据推送路由（不带 /v1 前缀，供 RouterOS 设备使用）
	// 这个路由不需要认证，设备直接推送数据
	pushApi := s.router.Group("/api/push")
	{
		pushApi.POST("/metrics", s.dataReceiverHandler.PushRouterOSMetrics)
		pushApi.POST("/bandwidth", s.dataReceiverHandler.PushBandwidthData)
		pushApi.POST("/ping", s.dataReceiverHandler.PushPingData)
	}

	// API路由组
	api := s.router.Group("/api/v1")
	{
		api.GET("/ping", s.ping)
		
		// 系统信息API
		api.GET("/system/info", s.systemInfo)
		
		// 注册认证路由
		s.authHandler.RegisterRoutes(api)
		
		// 注册RBAC路由
		s.rbacHandler.RegisterRoutes(api, s.authService)
		
		// 注册管理员API路由
		s.adminHandler.RegisterRoutes(api, s.authService)
		
		// 注册设备管理路由
		authenticated := api.Group("")
		authenticated.Use(auth.AuthMiddleware(s.authService))
		{
			// 使用带权限检查的设备路由注册
			readMiddleware := auth.DevicePermissionMiddleware(s.devicePermChecker, "read")
			updateMiddleware := auth.DevicePermissionMiddleware(s.devicePermChecker, "update")
			deleteMiddleware := auth.DevicePermissionMiddleware(s.devicePermChecker, "delete")
			s.deviceHandler.RegisterRoutesWithPermission(authenticated, readMiddleware, updateMiddleware, deleteMiddleware)
			
			s.tagHandler.RegisterRoutes(authenticated)
			s.deviceGroupHandler.RegisterRoutes(authenticated)
			s.connectionTestHandler.RegisterRoutes(authenticated) // 添加连接测试路由
			s.pingTargetHandler.RegisterRoutesWithPermission(authenticated, readMiddleware, updateMiddleware) // 添加 Ping 目标管理路由（带权限检查）
			s.settingsHandler.RegisterRoutes(authenticated)       // 添加系统设置路由
			s.collectorHandler.RegisterRoutesWithPermission(authenticated, readMiddleware, updateMiddleware) // 添加采集器管理路由（带权限检查）
			s.proxyHandler.RegisterRoutes(authenticated)          // 添加代理管理路由
			s.backupHandler.RegisterRoutes(authenticated)         // 添加系统备份路由
			s.marketplaceHandler.RegisterRoutes(authenticated)    // 添加插件市场路由
		}
		
		// 注册数据接收路由（不需要认证，供设备推送数据使用）
		s.dataReceiverHandler.RegisterRoutes(api)
		
		// 注册数据存储管理路由（需要认证）
		s.dataStorageHandler.RegisterRoutes(authenticated)
		
		// 注册数据查询路由（需要认证）
		s.dataQueryHandler.RegisterRoutes(authenticated)
		
		// 监控数据查询路由（占位符，后续实现）
		monitoring := api.Group("/monitoring")
		monitoring.Use(auth.AuthMiddleware(s.authService))
		{
			monitoring.GET("/realtime/:device_id", s.placeholder("GET /monitoring/realtime/:device_id"))
			monitoring.GET("/history/:device_id", s.placeholder("GET /monitoring/history/:device_id"))
		}
	}
}

// index 根路径处理器
func (s *Server) index(c *gin.Context) {
	c.JSON(200, gin.H{
		"name":    "NMP Platform",
		"version": "1.0.0",
		"message": "Network Monitoring Platform API",
		"docs":    "/api/v1/docs",
	})
}

// systemInfo 系统信息处理器
func (s *Server) systemInfo(c *gin.Context) {
	// 获取内存统计信息
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	// 计算内存使用率（基于 Go 运行时分配的内存）
	// Alloc: 当前分配的内存字节数
	// Sys: 从操作系统获取的总内存字节数
	memoryUsage := float64(0)
	if memStats.Sys > 0 {
		memoryUsage = float64(memStats.Alloc) / float64(memStats.Sys) * 100
	}
	
	// 获取 CPU 核心数作为参考（Go 运行时没有直接的 CPU 使用率 API）
	// 实际 CPU 使用率需要通过系统调用或第三方库获取
	numCPU := runtime.NumCPU()
	numGoroutine := runtime.NumGoroutine()
	
	// 计算运行时间
	uptime := time.Since(s.startTime)
	uptimeSeconds := int64(uptime.Seconds())
	
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"name":            "NMP Platform",
			"version":         "1.0.0",
			"description":     "网络监控平台",
			"build_time":      time.Now().Format("2006-01-02 15:04:05"),
			"go_version":      runtime.Version(),
			"platform":        runtime.GOOS + "/" + runtime.GOARCH,
			"uptime":          formatUptime(uptimeSeconds),
			"uptime_seconds":  uptimeSeconds,
			"status":          "running",
			"cpu_cores":       numCPU,
			"goroutines":      numGoroutine,
			"memory_alloc":    memStats.Alloc,
			"memory_sys":      memStats.Sys,
			"memory_usage":    memoryUsage,
		},
	})
}

// formatUptime 格式化运行时间为人类可读格式
func formatUptime(seconds int64) string {
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60
	
	if days > 0 {
		return fmt.Sprintf("%d天 %d小时 %d分钟", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d小时 %d分钟", hours, minutes)
	} else if minutes > 0 {
		return fmt.Sprintf("%d分钟 %d秒", minutes, secs)
	}
	return fmt.Sprintf("%d秒", secs)
}

// healthCheck 简单健康检查处理器
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
		"message": "NMP Platform is running",
		"timestamp": time.Now().Unix(),
	})
}

// detailedHealthCheck 详细健康检查处理器
func (s *Server) detailedHealthCheck(c *gin.Context) {
	ctx := c.Request.Context()
	overallStatus, results := s.healthChecker.GetOverallStatus(ctx)
	
	statusCode := 200
	if overallStatus == health.StatusUnhealthy {
		statusCode = 503
	} else if overallStatus == health.StatusDegraded {
		statusCode = 200 // 降级状态仍返回200，但在响应中标明
	}
	
	c.JSON(statusCode, gin.H{
		"status": overallStatus,
		"services": results,
		"timestamp": time.Now().Unix(),
	})
}

// version 版本信息处理器
func (s *Server) version(c *gin.Context) {
	c.JSON(200, gin.H{
		"version": "1.0.0",
		"build_time": "2024-01-01T00:00:00Z", // 实际应该从构建时注入
		"git_commit": "unknown", // 实际应该从构建时注入
		"go_version": "go1.21+",
	})
}

// metrics 系统指标处理器（简单实现）
func (s *Server) metrics(c *gin.Context) {
	c.JSON(200, gin.H{
		"uptime":             time.Since(s.startTime).String(),
		"uptime_seconds":     time.Since(s.startTime).Seconds(),
		"requests_total":     0, // TODO: 实际应该有计数器
		"active_connections": 0, // TODO: 实际应该有连接计数
	})
}

// ping 测试接口
func (s *Server) ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
		"timestamp": time.Now().Unix(),
	})
}

// placeholder 占位符处理器
func (s *Server) placeholder(endpoint string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": fmt.Sprintf("Endpoint %s - implementation pending", endpoint),
			"status": "placeholder",
		})
	}
}

// getLocalIP 获取本机的非回环 IP 地址
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}