# 项目结构说明

## 📁 完整目录树

```
llmpt/
├── cmd/                      # 可执行程序入口
│   ├── test-db/
│   │   └── main.go          # 数据库连接测试程序
│   ├── tracker/             # ✨ Step 2 新增
│   │   └── main.go          # Tracker Server 入口
│   └── test-tracker/        # ✨ Step 2 新增
│       └── main.go          # Tracker 测试程序
│
├── internal/                 # 项目内部代码（Go 编译器保护）
│   ├── config/
│   │   └── config.go        # 配置管理（环境变量、默认值）
│   ├── database/
│   │   ├── db.go            # 数据库管理器（统一入口）
│   │   ├── mongodb.go       # MongoDB 连接、操作、索引管理
│   │   └── redis.go         # Redis 连接、Peer 管理、统计
│   ├── models/
│   │   └── torrent.go       # 数据模型（Torrent、Peer、Announce）
│   └── tracker/             # ✨ Step 2 新增
│       ├── announce.go      # /announce 接口实现
│       ├── bencode.go       # Bencode 编码/解码
│       └── compact.go       # Compact Peer 格式处理
│
├── .env.example              # 环境变量配置示例
├── .gitignore                # Git 忽略规则
├── docker-compose.yml        # 数据库容器配置（MongoDB + Redis）
├── go.mod                    # Go 模块依赖
├── go.sum                    # 依赖校验和
├── Makefile                  # ✨ Step 2 新增 - 构建和运行脚本
├── DATABASE_SETUP.md         # 数据库使用文档
├── TRACKER_GUIDE.md          # ✨ Step 2 新增 - Tracker 使用指南
├── STEP2_COMPLETION.md       # ✨ Step 2 新增 - 完成总结
├── REFACTORING.md            # 重构说明文档
├── PROJECT_STRUCTURE.md      # 本文件
└── README.md                 # 项目设计文档

```

## 📦 模块说明

### `cmd/` - 命令行工具
存放可执行程序的入口文件。

- **`test-db/`**: 数据库连接测试工具
  - 验证 MongoDB 和 Redis 连接
  - 测试 CRUD 操作
  - 测试 Peer 管理和统计功能

- **`tracker/`**: Tracker Server（✅ Step 2 完成）
  - BitTorrent Tracker 服务器
  - 处理 /announce 请求
  - 管理 Peer 列表和统计信息

- **`test-tracker/`**: Tracker 测试工具
  - 测试 Bencode 编码
  - 测试 Compact Peer 格式
  - 测试 Announce 请求

### `internal/` - 内部代码（核心业务逻辑）

#### `internal/config/`
**职责**: 应用配置管理

- 从环境变量加载配置
- 提供默认值
- 配置结构定义（MongoDB、Redis、Server）

**主要函数**:
- `Load()`: 加载配置
- `GetMongoURI()`: 获取 MongoDB 连接字符串
- `GetRedisAddr()`: 获取 Redis 地址

#### `internal/database/`
**职责**: 数据库连接和操作

**`db.go`** - 数据库管理器
- 统一的数据库初始化入口
- 管理 MongoDB 和 Redis 连接生命周期
- 自动创建索引

**`mongodb.go`** - MongoDB 操作
- 连接池管理（最大 50，最小 10）
- 自动健康检查
- 创建索引：
  - `info_hash` (唯一索引)
  - `created_at` (降序索引)
  - `name` (文本搜索索引)
- `TorrentsCollection()`: 获取 torrents 集合

**`redis.go`** - Redis 操作
- 连接池管理
- **Peer 管理**:
  - `AddPeer()`: 添加 Peer（自动 30 分钟 TTL）
  - `GetPeers()`: 随机获取指定数量的 Peer
  - `RemovePeer()`: 移除 Peer
  - `GetPeerCount()`: 获取 Peer 数量
- **统计管理**:
  - `UpdateStats()`: 更新统计信息
  - `GetStats()`: 获取统计信息
  - `IncrementCompleted()`: 增加完成计数

#### `internal/models/`
**职责**: 数据模型定义

- `Torrent`: 种子信息（MongoDB 模型）
- `TorrentStats`: 统计信息（Redis 数据）
- `PeerInfo`: Peer 信息
- `AnnounceRequest`: Tracker 请求参数
- `AnnounceResponse`: Tracker 响应数据

#### `internal/tracker/` ✅ Step 2 完成
**职责**: Tracker Server 核心实现

**`announce.go`** - Announce 接口
- 处理 `/announce` HTTP 请求
- 解析请求参数（info_hash, peer_id, port 等）
- 管理 Peer 注册、心跳、移除
- 更新统计信息（Seeders/Leechers）
- 返回 Bencode 响应（支持 Compact 和标准模式）

**`bencode.go`** - Bencode 编码
- `EncodeString()`: 字符串编码
- `EncodeInt()`: 整数编码
- `EncodeList()`: 列表编码
- `EncodeDict()`: 字典编码（键自动排序）
- `DecodeString()`: 字符串解码

**`compact.go`** - Compact Peer 格式（BEP-0023）
- `CompactPeer()`: 单个 Peer 编码（6 字节）
- `CompactPeers()`: 批量 Peer 编码
- `DecompactPeer()`: 单个 Peer 解码
- `DecompactPeers()`: 批量 Peer 解码

## 🔗 依赖关系

```
cmd/test-db/
    ├── internal/config      (配置加载)
    ├── internal/database    (数据库操作)
    └── internal/models      (数据模型)

internal/database/
    ├── internal/config      (获取连接配置)
    ├── go.mongodb.org/mongo-driver
    └── github.com/redis/go-redis/v9
```

## 🎯 设计原则

### 1. **单一职责**
- `config`: 只负责配置管理
- `database`: 只负责数据库操作
- `models`: 只定义数据结构

### 2. **依赖注入**
```go
// 通过参数传递依赖，便于测试
func New(cfg *config.Config) (*DB, error)
```

### 3. **封装隔离**
- 使用 `internal/` 防止外部依赖
- 数据库实现细节不暴露给外部

### 4. **错误处理**
```go
// 统一的错误包装格式
return nil, fmt.Errorf("failed to connect: %w", err)
```

## 📈 未来扩展

按照 README.md 的设计，后续需要添加：

### `internal/api/` - Web API（Step 4）
```
internal/api/
├── handler.go          # HTTP 路由处理
├── publish.go          # POST /api/v1/publish
└── torrents.go         # GET /api/v1/torrents
```

### `internal/tracker/` - Tracker 服务 ✅ 已完成（Step 2）
```
internal/tracker/
├── announce.go         # ✅ GET /announce
├── bencode.go          # ✅ Bencode 编码/解码
└── compact.go          # ✅ Compact 模式实现（BEP-0023）
```

### `cmd/model-cli/` - CLI 客户端
```
cmd/model-cli/
├── main.go             # CLI 入口
├── share.go            # 做种命令
└── download.go         # 下载命令
```

### `pkg/p2p/` - BT 协议封装（可选）
```
pkg/p2p/
├── client.go           # BT 客户端封装
└── create.go           # 种子创建
```

> 注意：`pkg/` 仅用于真正通用的、可被外部导入的库代码

## 🧪 测试策略

### 单元测试（计划中）
```
internal/config/config_test.go
internal/database/mongodb_test.go
internal/database/redis_test.go
```

### 集成测试
- ✅ `cmd/test-db/main.go` - 已完成

### 性能测试（计划中）
```
internal/database/benchmark_test.go
```

## 📝 编码规范

### Import 顺序
```go
import (
    // 1. 标准库
    "context"
    "fmt"
    
    // 2. 第三方库
    "github.com/redis/go-redis/v9"
    
    // 3. 本项目内部包
    "llmpt/internal/config"
)
```

### 命名规范
- **包名**: 小写单数（`config`, `database`）
- **文件名**: 小写下划线（`mongodb.go`, `peer_manager.go`）
- **导出函数**: 大驼峰（`NewMongoDB`, `AddPeer`）
- **私有函数**: 小驼峰（`getEnv`, `validateConfig`）

### 注释规范
```go
// AddPeer 添加 Peer 到指定 info_hash 的集合
// TTL 默认 30 分钟
func (r *Redis) AddPeer(ctx context.Context, infoHash, peer string) error
```

## 🔍 快速查找

| 功能 | 文件位置 |
|------|----------|
| 加载配置 | `internal/config/config.go` |
| MongoDB 连接 | `internal/database/mongodb.go` |
| Redis 连接 | `internal/database/redis.go` |
| 数据库初始化 | `internal/database/db.go` |
| 数据模型 | `internal/models/torrent.go` |
| Tracker Server | `cmd/tracker/main.go` |
| Announce 接口 | `internal/tracker/announce.go` |
| Bencode 编码 | `internal/tracker/bencode.go` |
| Compact 格式 | `internal/tracker/compact.go` |
| 数据库测试 | `cmd/test-db/main.go` |
| Tracker 测试 | `cmd/test-tracker/main.go` |
| 环境变量配置 | `.env.example` |
| 数据库容器 | `docker-compose.yml` |
| 构建脚本 | `Makefile` |

## 📚 相关文档

- **[README.md](./README.md)**: 系统设计文档
- **[DATABASE_SETUP.md](./DATABASE_SETUP.md)**: 数据库使用指南
- **[TRACKER_GUIDE.md](./TRACKER_GUIDE.md)**: Tracker Server 使用指南 ✨
- **[STEP2_COMPLETION.md](./STEP2_COMPLETION.md)**: Step 2 完成总结 ✨
- **[REFACTORING.md](./REFACTORING.md)**: 代码重构说明
- **[.env.example](./.env.example)**: 配置示例
- **[Makefile](./Makefile)**: 构建和运行脚本 ✨

## 📊 开发进度

- ✅ **Step 1**: 基础设施（MongoDB + Redis）
- ✅ **Step 2**: Tracker Server 实现
- ⏳ **Step 2.5**: 协议兼容性验证
- ⏳ **Step 3**: CLI 客户端开发
- ⏳ **Step 4**: Web API & Frontend
- ⏳ **Step 5**: 联调与部署

---

**项目名称**: llmpt - 大模型 P2P 分享站  
**当前版本**: V1.1  
**当前阶段**: Step 2 完成 ✅  
**更新日期**: 2026-02-05
