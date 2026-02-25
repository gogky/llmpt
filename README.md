# LLM-PT 部署指南

## 1. 基础环境
- **Go**: 1.20+
- **Node.js**: 18+
- **MongoDB**: 6.0+
- **Redis**: 7.0+

推荐使用 Docker 本地启动数据库组件：
```bash
docker run -d -p 27017:27017 --name mongo-llm mongo:latest
docker run -d -p 6379:6379 --name redis-llm redis:latest
```

## 2. 启动说明

### Tracker 服务 (P2P 握手与追踪)
监听 BitTorrent 客户端的 `/announce` 请求。
```bash
# Windows
$env:SERVER_PORT="6969"; go run ./cmd/tracker/main.go

# Linux/macOS
export SERVER_PORT=6969
go run ./cmd/tracker/main.go
```

### Web API 服务 (模型元数据与接口)
给前端面板和 CLI 上传元数据提供服务。默认从 8080 端口启动。
```bash
# Windows
$env:SERVER_PORT="8080"; go run ./cmd/web-server/main.go

# Linux/macOS
export SERVER_PORT=8080
go run ./cmd/web-server/main.go
```

### 前端面板 (Vue 3 UI)
提供可视化操作界面，依赖后端的 8080 端口提供数据。
```bash
cd frontend
npm install
npm run dev
```
打开浏览器访问 `http://localhost:5173`。
