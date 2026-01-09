# NMP Platform Makefile

.PHONY: help build run test clean deps dev migrate migrate-fresh seed

# 默认目标
help:
	@echo "Available commands:"
	@echo "  build         - 构建应用程序"
	@echo "  run           - 运行应用程序"
	@echo "  dev           - 开发模式运行"
	@echo "  test          - 运行测试"
	@echo "  clean         - 清理构建文件"
	@echo "  deps          - 安装依赖"
	@echo "  migrate       - 执行数据库迁移"
	@echo "  migrate-fresh - 重新创建数据库并初始化数据"
	@echo "  seed          - 初始化基础数据"

# 构建应用程序
build:
	@echo "Building NMP Platform..."
	go build -o bin/nmp-server cmd/server/main.go

# 运行应用程序
run: build
	@echo "Starting NMP Platform..."
	./bin/nmp-server

# 开发模式运行
dev:
	@echo "Starting NMP Platform in development mode..."
	go run cmd/server/main.go

# 运行测试
test:
	@echo "Running tests..."
	go test -v ./...

# 清理构建文件
clean:
	@echo "Cleaning build files..."
	rm -rf bin/

# 安装依赖
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# 数据库迁移
migrate:
	@echo "Running database migration..."
	go build -o bin/migrate cmd/migrate/main.go
	./bin/migrate

# 重新创建数据库
migrate-fresh:
	@echo "Recreating database with fresh data..."
	go build -o bin/migrate cmd/migrate/main.go
	./bin/migrate -drop -seed

# 初始化基础数据
seed:
	@echo "Seeding database with initial data..."
	go build -o bin/migrate cmd/migrate/main.go
	./bin/migrate -seed

# 格式化代码
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
lint:
	@echo "Running linter..."
	golangci-lint run

# 生成文档
docs:
	@echo "Generating documentation..."
	godoc -http=:6060