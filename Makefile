.PHONY: help db-up db-down db-logs test-db tracker test-tracker clean

help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

db-up: ## 启动数据库（MongoDB + Redis）
	docker-compose up -d
	@echo "✅ 数据库已启动"
	@echo "MongoDB: localhost:27017"
	@echo "Redis: localhost:6379"

db-down: ## 停止数据库
	docker-compose down
	@echo "✅ 数据库已停止"

db-logs: ## 查看数据库日志
	docker-compose logs -f

test-db: ## 测试数据库连接
	@echo "🧪 测试数据库连接..."
	cd cmd/test-db && go run main.go

tracker: ## 启动 Tracker Server
	@echo "🚀 启动 Tracker Server..."
	cd cmd/tracker && go run main.go

test-tracker: ## 测试 Tracker 功能
	@echo "🧪 测试 Tracker..."
	cd cmd/test-tracker && go run main.go

clean: ## 清理临时文件
	go clean
	rm -f cmd/test-db/test-db
	rm -f cmd/tracker/tracker
	rm -f cmd/test-tracker/test-tracker

build-tracker: ## 编译 Tracker Server
	@echo "🔨 编译 Tracker Server..."
	cd cmd/tracker && go build -o tracker main.go
	@echo "✅ 编译完成: cmd/tracker/tracker"

build-all: ## 编译所有程序
	@echo "🔨 编译所有程序..."
	cd cmd/test-db && go build -o test-db main.go
	cd cmd/tracker && go build -o tracker main.go
	cd cmd/test-tracker && go build -o test-tracker main.go
	@echo "✅ 编译完成"

redis-cli: ## 连接到 Redis CLI
	docker exec -it llmpt-redis-1 redis-cli

mongo-cli: ## 连接到 MongoDB CLI
	docker exec -it llmpt-mongodb-1 mongosh -u admin -p admin123 --authenticationDatabase admin

deps: ## 下载依赖
	go mod download
	go mod tidy

fmt: ## 格式化代码
	go fmt ./...

vet: ## 代码检查
	go vet ./...

lint: fmt vet ## 代码格式化和检查

run: db-up tracker ## 启动完整环境（数据库 + Tracker）
