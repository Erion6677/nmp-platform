package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nmp-platform/internal/config"
	"nmp-platform/internal/server"

	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer logger.Sync()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// 设置全局配置
	config.SetConfig(cfg)

	// 创建服务器
	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to create server", zap.Error(err))
	}

	// 启动服务器
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal("Failed to start HTTP server", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// 关闭服务器资源
	if err := srv.Close(); err != nil {
		logger.Error("Error closing server resources", zap.Error(err))
	}

	logger.Info("Server exited")
}