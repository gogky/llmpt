# LLM-PT 部署指南

## 1. 基础环境
- **Go**: 1.20+
- **Node.js**: 18+
- **MongoDB**: 6.0+
- **Redis**: 7.0+

推荐使用 Docker Compose 完整启动所有基础设施（数据库与 Nginx 代理）：
```bash
docker-compose up -d
```

## 2. 配置说明

项目根目录下有一个 `.env.example` 文件，请将其复制并重命名为 `.env`。

关于 `TRACKER_URL` 的配置：
- **本地开发**：保持默认的 `http://localhost/announce`（通过 Nginx 代理）。
- **生产部署**：请将 `TRACKER_URL` 修改为服务器的公网 IP 或真实域名。该地址会被写入 `.torrent` 种子文件中，以供客户端连接使用。

## 3. 启动说明


### Nginx 反向代理
由于 Tracker 和 Web API 分属不同的进程，本项目使用 Nginx 对外提供统一的 `80` 端口。
请求将会通过以下规则被正确路由：
- `/announce` -> Tracker (默认 8081)
- 其他路径 -> Web API (默认 8080)

可以通过 `docker-compose up -d` 直接启动自带正确配置的 Nginx 代理容器。

### Tracker 服务 (P2P 握手与追踪)
监听 BitTorrent 客户端的 `/announce` 请求。默认本地监听在 8081 端口。
```bash
# Windows
go run ./cmd/tracker/main.go

# Linux/macOS
go run ./cmd/tracker/main.go
```

### Web API 服务 (模型元数据与接口)
给前端面板和 CLI 上传元数据提供服务。默认监听在 8080 端口。
```bash
# Windows
go run ./cmd/web-server/main.go

# Linux/macOS
go run ./cmd/web-server/main.go
```

### 前端面板 (Vue 3 UI)
提供可视化操作界面，默认依赖统一入口 `80` 抓取数据。
如果你的统一网关运行在别的端口（例如 9000），请在启动前通过 `VITE_API_URL` 告知 Vite 代理地址。

```bash
cd frontend
npm install

# 默认启动 (对应 Web API 跑在 8080)
npm run dev

# (可选) 自定义后端服务地址启动
# Windows: $env:VITE_API_URL="http://127.0.0.1:9000"; npm run dev
# Linux/macOS: VITE_API_URL="http://127.0.0.1:9000" npm run dev
```
打开浏览器访问 `http://localhost:5173`。
