# 快速启动指南

## 🚀 Step 2: Tracker Server 快速启动

### 前置要求

- ✅ Go 1.21+
- ✅ Docker & Docker Compose
- ✅ Git

### 1️⃣ 启动数据库

```powershell
# 启动 MongoDB 和 Redis
docker-compose up -d

# 查看运行状态
docker-compose ps
```

预期输出：

```
NAME                IMAGE               STATUS
llmpt-mongodb-1     mongo:7            Up
llmpt-redis-1       redis:7-alpine     Up
```

### 2️⃣ 测试数据库连接

```powershell
cd cmd\test-db
go run main.go
```

预期输出：

```
✓ Successfully connected to MongoDB
✓ Successfully connected to Redis
🧪 测试 Peer 管理...
✅ 所有测试通过！
```

### 3️⃣ 测试 Tracker 功能

```powershell
cd ..\test-tracker
go run main.go
```

预期输出：

```
🧪 Testing Tracker Implementation...

📝 Test 1: Bencode Encoding
String: spam -> 4:spam
Int: 42 -> i42e
✅ 通过

📦 Test 2: Compact Peer Format
✅ Single peer test passed
✅ Multiple peers test passed

✅ All tests completed!
```

### 4️⃣ 启动 Tracker Server

```powershell
cd ..\tracker
go run main.go
```

预期输出：

```
🚀 Starting Tracker Server...
✅ Database connected
🎯 Tracker Server listening on :8080
📡 Announce endpoint: http://localhost:8080/announce
```

### 5️⃣ 测试 Announce 接口

在另一个终端中运行：

```powershell
# 测试健康检查
curl http://localhost:8080/health

# 模拟 Announce 请求
curl "http://localhost:8080/announce?info_hash=test123&peer_id=peer001&port=6881&uploaded=0&downloaded=0&left=1000000&compact=1"
```

## 📊 查看 Redis 数据

```powershell
# 连接到 Redis
docker exec -it llmpt-redis-1 redis-cli

# 查看所有 Tracker 相关的 Key
KEYS tracker:*

# 查看 Peer 列表（替换 <info_hash>）
SMEMBERS tracker:peers:<info_hash>

# 查看统计信息
HGETALL tracker:stats:<info_hash>
```

## 🛠️ 使用 Makefile（可选）

如果你安装了 `make`（Windows 可以使用 Chocolatey 安装）：

```powershell
# 安装 make（如果没有）
choco install make

# 启动数据库
make db-up

# 测试数据库
make test-db

# 测试 Tracker
make test-tracker

# 启动 Tracker Server
make tracker

# 查看所有可用命令
make help
```

## 🐛 常见问题

### 问题 1: 数据库连接失败

```
Failed to connect to database
```

**解决方法**:

```powershell
# 检查 Docker 是否运行
docker ps

# 重启数据库
docker-compose restart

# 查看日志
docker-compose logs
```

### 问题 2: 端口被占用

```
bind: address already in use
```

**解决方法**:

1. 修改 `.env` 文件中的 `SERVER_PORT`
2. 或者停止占用 8080 端口的程序

```powershell
# 查看端口占用
netstat -ano | findstr :8080

# 杀死进程（替换 <PID>）
taskkill /PID <PID> /F
```

### 问题 3: Go 依赖下载慢

**解决方法**:

```powershell
# 设置 Go 代理（中国大陆）
$env:GOPROXY = "https://goproxy.cn,direct"

# 下载依赖
go mod download
```

## 📚 下一步

- 阅读 [TRACKER_GUIDE.md](./TRACKER_GUIDE.md) 了解详细实现
- 阅读 [STEP2_COMPLETION.md](./STEP2_COMPLETION.md) 查看完成总结
- 进行 **Step 2.5: 协议兼容性验证**（使用 qBittorrent + Transmission）

## 🎯 Step 2.5: 兼容性测试

### 1. 用 qBittorrent 制作种子

1. 打开 qBittorrent
2. 工具 → Torrent Creator
3. 选择文件/文件夹
4. Tracker URLs: `http://你的IP:8080/announce`
5. 勾选 "私有种子"
6. 创建并开始做种

### 2. 用 Transmission 下载

1. 在另一台电脑上安装 Transmission
2. 打开刚才的 `.torrent` 文件
3. 观察是否能发现 qBittorrent 并开始传输

### 3. 验证 Tracker

```powershell
# 查看 Tracker 日志
# 应该能看到两个客户端的请求

# 查看 Redis
docker exec -it llmpt-redis-1 redis-cli
> KEYS tracker:*
> SMEMBERS tracker:peers:<info_hash>
```

---

**项目**: llmpt - 大模型 P2P 分享站  
**当前阶段**: Step 2 ✅  
**更新时间**: 2026-02-05
