# 数据库连接基础代码说明

## 📁 项目结构

```
llmpt/
├── internal/            # 项目内部代码（受 Go 编译器保护）
│   ├── config/          # 配置管理
│   │   └── config.go    # 配置加载和环境变量处理
│   ├── database/        # 数据库连接
│   │   ├── db.go        # 数据库管理器（统一入口）
│   │   ├── mongodb.go   # MongoDB 连接和操作
│   │   └── redis.go     # Redis 连接和操作
│   └── models/          # 数据模型
│       └── torrent.go   # Torrent 相关数据模型
├── cmd/
│   └── test-db/         # 数据库测试程序
│       └── main.go
├── .env.example         # 环境变量示例
└── docker-compose.yml   # 数据库容器配置
```

## 🚀 快速开始

### 1. 启动数据库服务

```bash
# 启动 MongoDB 和 Redis
docker-compose up -d

# 检查服务状态
docker-compose ps
```

### 2. 安装 Go 依赖

```bash
go mod tidy
```

### 3. 配置环境变量（可选）

```bash
# 复制环境变量示例文件
cp .env.example .env

# 修改 .env 文件中的配置（如果需要）
```

### 4. 运行数据库连接测试

```bash
go run cmd/test-db/main.go
```

## 📦 核心功能

### 配置管理 (`internal/config`)

- 支持环境变量配置
- 提供默认值
- 简化配置加载

```go
cfg, err := config.Load()
```

### MongoDB 操作 (`internal/database/mongodb.go`)

#### 主要功能：
- ✅ 自动连接和健康检查
- ✅ 连接池管理（最大 50，最小 10）
- ✅ 自动创建索引
- ✅ 支持 torrents 集合操作

#### 创建的索引：
1. `info_hash` - 唯一索引
2. `created_at` - 降序索引（用于排序）
3. `name` - 文本索引（用于搜索）

#### 使用示例：

```go
// 获取 torrents 集合
collection := db.MongoDB.TorrentsCollection()

// 插入数据
result, err := collection.InsertOne(ctx, torrent)

// 查询数据
var torrent models.Torrent
err := collection.FindOne(ctx, bson.M{"info_hash": hash}).Decode(&torrent)
```

### Redis 操作 (`internal/database/redis.go`)

#### Tracker Peer 管理：

```go
// 添加 Peer（自动设置 30 分钟 TTL）
err := db.Redis.AddPeer(ctx, infoHash, "192.168.1.100:6881")

// 获取指定数量的随机 Peer
peers, err := db.Redis.GetPeers(ctx, infoHash, 50)

// 移除 Peer
err := db.Redis.RemovePeer(ctx, infoHash, peer)

// 获取 Peer 数量
count, err := db.Redis.GetPeerCount(ctx, infoHash)
```

#### 统计信息管理：

```go
// 更新统计信息
err := db.Redis.UpdateStats(ctx, infoHash, seeders, leechers, completed)

// 获取统计信息
stats, err := db.Redis.GetStats(ctx, infoHash)

// 增加完成下载计数
err := db.Redis.IncrementCompleted(ctx, infoHash)
```

### 数据模型 (`internal/models`)

#### Torrent 模型

```go
type Torrent struct {
    ID          primitive.ObjectID  // MongoDB ID
    Name        string              // 模型名称
    InfoHash    string              // 种子唯一指纹（40 字符 hex）
    TotalSize   int64               // 总大小（字节）
    FileCount   int                 // 文件数量
    MagnetLink  string              // 磁力链接
    PieceLength int64               // 分片大小（字节）
    CreatedAt   time.Time           // 创建时间
}
```

#### AnnounceRequest 模型（用于 Tracker）

```go
type AnnounceRequest struct {
    InfoHash   string  // 种子 hash
    PeerID     string  // 客户端 ID
    Port       int     // 监听端口
    Uploaded   int64   // 已上传字节数
    Downloaded int64   // 已下载字节数
    Left       int64   // 剩余字节数
    Event      string  // 事件: started, completed, stopped
    Compact    int     // 是否使用紧凑模式
    NumWant    int     // 期望返回的 peer 数量
}
```

## 🔧 在您的代码中使用

### 完整示例：

```go
package main

import (
    "context"
    "log"
    
    "llmpt/internal/config"
    "llmpt/internal/database"
    "llmpt/internal/models"
)

func main() {
    // 1. 加载配置
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    
    // 2. 初始化数据库
    db, err := database.New(cfg)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // 3. 使用 MongoDB
    ctx := context.Background()
    collection := db.MongoDB.TorrentsCollection()
    
    // 插入种子
    torrent := &models.Torrent{
        Name:        "Llama-3-8B",
        InfoHash:    "abc123...",
        TotalSize:   15000000000,
        FileCount:   120,
        PieceLength: 8388608,
    }
    _, err = collection.InsertOne(ctx, torrent)
    
    // 4. 使用 Redis
    // 添加 Peer
    err = db.Redis.AddPeer(ctx, torrent.InfoHash, "192.168.1.100:6881")
    
    // 获取 Peer 列表（最多 50 个）
    peers, err := db.Redis.GetPeers(ctx, torrent.InfoHash, 50)
    
    // 更新统计
    err = db.Redis.UpdateStats(ctx, torrent.InfoHash, 10, 5, 100)
}
```

## 🔑 Redis Key 设计

按照设计文档的规范：

1. **Peer 列表**: `tracker:peers:{info_hash}`
   - Type: Set
   - Value: `{IP}:{Port}`
   - TTL: 30 分钟

2. **统计信息**: `tracker:stats:{info_hash}`
   - Type: Hash
   - Fields: `seeders`, `leechers`, `completed`
   - TTL: 1 小时

## 📊 数据库连接配置

### MongoDB
- **默认连接**: `mongodb://admin:admin123@localhost:27017`
- **数据库**: `hf_p2p_v1`
- **连接池**: 最大 50，最小 10
- **空闲超时**: 30 秒

### Redis
- **默认地址**: `localhost:6379`
- **连接池**: 最大 50，最小 10
- **超时设置**:
  - 拨号超时: 5 秒
  - 读超时: 3 秒
  - 写超时: 3 秒

## 🧪 测试

运行测试程序将执行以下操作：

1. ✅ 连接 MongoDB 和 Redis
2. ✅ 创建数据库索引
3. ✅ 测试 MongoDB CRUD 操作
4. ✅ 测试 Redis Peer 管理
5. ✅ 测试 Redis 统计信息
6. ✅ 清理测试数据

## 🛠️ 下一步开发

现在您可以基于这些基础代码开发：

1. **Web API** (`/api/v1/publish`, `/api/v1/torrents`)
2. **Tracker API** (`/announce`)
3. **CLI 客户端** (做种、下载功能)

## 📝 注意事项

- 确保 Docker 容器正在运行
- 默认端口不要被占用（27017, 6379）
- 生产环境请修改默认密码
- Redis 的 Peer TTL 设置为 30 分钟，符合 BT 协议的心跳间隔
