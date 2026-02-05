# Tracker Server 使用指南

## 📖 概述

Tracker Server 是 BitTorrent 协议的核心组件，负责协调 Peer 之间的连接。本实现遵循以下标准：

- **BEP-0003**: BitTorrent 协议规范
- **BEP-0023**: Compact Peer 列表（紧凑格式）

## 🏗️ 架构设计

### 核心组件

```
internal/tracker/
├── announce.go     # /announce 接口实现
├── bencode.go      # Bencode 编码/解码
└── compact.go      # Compact Peer 格式处理

cmd/tracker/
└── main.go         # Tracker Server 入口
```

### 数据流

```
BT 客户端
    │
    ├─> HTTP GET /announce?info_hash=...&peer_id=...
    │
    v
Tracker Server (announce.go)
    │
    ├─> 解析请求参数
    ├─> 更新 Redis (Peer 列表 + 统计)
    ├─> 获取其他 Peer
    └─> 返回 Bencode 响应
         ├─> Compact 模式 (6字节/Peer)
         └─> 标准模式 (字典列表)
```

## 🚀 快速启动

### 1. 启动数据库

```bash
docker-compose up -d
```

### 2. 设置环境变量

复制并编辑配置文件：

```bash
cp .env.example .env
```

确保配置了以下参数：

```env
# 服务器配置
SERVER_PORT=8080
TRACKER_URL=http://localhost:8080/announce

# MongoDB 配置
MONGODB_URI=mongodb://admin:admin123@localhost:27017
MONGODB_DATABASE=hf_p2p_v1

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
```

### 3. 启动 Tracker Server

```bash
cd cmd/tracker
go run main.go
```

输出示例：

```
🚀 Starting Tracker Server...
✅ Database connected
🎯 Tracker Server listening on :8080
📡 Announce endpoint: http://localhost:8080/announce
```

### 4. 运行测试

```bash
cd cmd/test-tracker
go run main.go
```

## 📡 API 接口

### `/announce` - Tracker 核心接口

**请求方法**: `GET`

**请求参数**:

| 参数 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `info_hash` | string | ✅ | 种子的 Info Hash (20 字节，URL 编码) |
| `peer_id` | string | ✅ | 客户端 ID (20 字节) |
| `port` | int | ✅ | 监听端口 (1-65535) |
| `uploaded` | int64 | ❌ | 已上传字节数 |
| `downloaded` | int64 | ❌ | 已下载字节数 |
| `left` | int64 | ❌ | 剩余字节数 (0 表示 Seeder) |
| `event` | string | ❌ | 事件类型: `started`, `completed`, `stopped` |
| `compact` | int | ❌ | `1` = 紧凑格式，`0` = 标准格式 |
| `numwant` | int | ❌ | 期望返回的 Peer 数量 (默认 50，最大 50) |
| `ip` | string | ❌ | 客户端 IP (可选，默认使用连接 IP) |

**响应格式**: Bencode 编码

**成功响应**:

```
d8:intervali1800e12:min intervali900e8:completei5e10:incompletei10e5:peers...e
```

解码后的结构：

```json
{
  "interval": 1800,          // 心跳间隔（秒）
  "min interval": 900,       // 最小心跳间隔（秒）
  "complete": 5,             // Seeders 数量
  "incomplete": 10,          // Leechers 数量
  "peers": "..."             // Peer 列表（格式取决于 compact 参数）
}
```

**Compact 模式 (compact=1)**:

`peers` 字段为二进制字符串，每 6 字节表示一个 Peer：

```
[IP1 (4字节)] [Port1 (2字节)] [IP2 (4字节)] [Port2 (2字节)] ...
```

示例：

```
192.168.1.100:6881 -> C0 A8 01 64 1A E1
```

**标准模式 (compact=0)**:

`peers` 字段为字典列表：

```
l
  d2:ip13:192.168.1.1004:porti6881ee
  d2:ip9:10.0.0.54:porti51413ee
e
```

**错误响应**:

```
d14:failure reason30:invalid request: missing portee
```

### `/health` - 健康检查

**请求方法**: `GET`

**响应**: `OK` (HTTP 200)

## 🔧 核心实现细节

### 1. Bencode 编码 (`bencode.go`)

Bencode 是 BitTorrent 协议使用的编码格式：

- **字符串**: `<长度>:<内容>` → `4:spam`
- **整数**: `i<数字>e` → `i42e`
- **列表**: `l<元素>e` → `l4:spam4:eggse`
- **字典**: `d<key><value>e` → `d3:key5:valuee` (键必须按字典序排序)

### 2. Compact Peer 格式 (`compact.go`)

紧凑格式显著减少带宽消耗：

- **标准格式**: ~50 字节/Peer (Bencode 字典)
- **Compact 格式**: 6 字节/Peer (二进制)
- **节省**: **88%**

编码示例：

```go
ip := "192.168.1.100"
port := 6881

// 转换为字节
compact := []byte{
    0xC0, 0xA8, 0x01, 0x64,  // IP: 192.168.1.100
    0x1A, 0xE1,              // Port: 6881 (大端序)
}
```

### 3. Peer 管理 (`announce.go`)

**Redis 数据结构**:

1. **Peer 列表** (Set):
   - Key: `tracker:peers:{info_hash}`
   - Value: `IP:Port`
   - TTL: 30 分钟

2. **统计信息** (Hash):
   - Key: `tracker:stats:{info_hash}`
   - Fields: `seeders`, `leechers`, `completed`

**事件处理**:

| Event | 动作 |
|-------|------|
| `started` | 添加 Peer 到 Redis |
| `completed` | 增加完成计数，更新为 Seeder |
| `stopped` | 从 Redis 移除 Peer |
| (无事件) | 心跳，更新 TTL |

## 🧪 测试

### 单元测试

```bash
cd cmd/test-tracker
go run main.go
```

测试内容：

1. ✅ Bencode 编码/解码
2. ✅ Compact Peer 格式转换
3. ✅ 单个 Peer 注册
4. ✅ 多个 Peer 互相发现

### 兼容性测试（Step 2.5）

按照 README.md 的建议，使用标准 BT 客户端验证：

1. **用 qBittorrent 制作种子**:
   - 创建一个测试文件
   - Tracker 填写: `http://localhost:8080/announce`
   - 勾选 "私有种子" (Private)

2. **用 Transmission 下载**:
   - 在另一台电脑或虚拟机上打开种子
   - 观察是否能发现 qBittorrent 并开始传输

3. **检查 Redis**:

```bash
redis-cli
> KEYS tracker:*
> SMEMBERS tracker:peers:<info_hash>
> HGETALL tracker:stats:<info_hash>
```

## 📊 监控与调试

### 查看日志

Tracker Server 会记录所有请求：

```
2026/02/05 10:30:15 GET /announce from 192.168.1.100:54321
2026/02/05 10:30:15 Request completed in 5.234ms
```

### 查看 Redis 数据

```bash
# 查看所有 Tracker 相关的 Key
redis-cli KEYS "tracker:*"

# 查看某个 info_hash 的 Peer 列表
redis-cli SMEMBERS "tracker:peers:abc123..."

# 查看统计信息
redis-cli HGETALL "tracker:stats:abc123..."

# 查看 Peer TTL
redis-cli TTL "tracker:peers:abc123..."
```

### 性能优化

- **连接池**: Redis 连接池大小 50，最小 10
- **TTL 自动清理**: Redis 自动删除过期 Peer
- **随机 Peer 选择**: 使用 `SRANDMEMBER` 实现负载均衡
- **限制返回数量**: 最多返回 50 个 Peer

## 🔐 安全考虑

### 当前实现

- ✅ 私有 Tracker (不支持 DHT)
- ✅ 自动过期机制 (30 分钟 TTL)
- ✅ IP 地址验证

### 待增强

- ⏳ 请求频率限制 (Rate Limiting)
- ⏳ IP 白名单/黑名单
- ⏳ Peer ID 验证
- ⏳ HTTPS 支持

## 📚 参考资料

- [BEP-0003: BitTorrent Protocol](https://www.bittorrent.org/beps/bep_0003.html)
- [BEP-0023: Tracker Returns Compact Peer Lists](https://www.bittorrent.org/beps/bep_0023.html)
- [Theory.org: How BitTorrent Works](http://www.theory.org/software/bittorrent/bittorrent-faq.html)

## 🐛 常见问题

### 1. Tracker 启动失败

**问题**: `Failed to connect to database`

**解决**:
```bash
# 检查数据库是否运行
docker-compose ps

# 重启数据库
docker-compose restart
```

### 2. 客户端无法连接

**问题**: qBittorrent 显示 "Tracker 不工作"

**检查清单**:
- [ ] Tracker URL 正确: `http://IP:8080/announce`
- [ ] 防火墙开放 8080 端口
- [ ] 客户端和 Tracker 在同一网络
- [ ] 查看 Tracker 日志是否收到请求

### 3. Peer 发现缓慢

**原因**: Redis 中 Peer 数量不足

**解决**:
- 增加做种客户端
- 减少 `numwant` 参数（让客户端频繁请求）
- 检查 Peer TTL 是否过短

## 🎯 下一步

- [ ] 实现 Web API (`/api/v1/publish`, `/api/v1/torrents`)
- [ ] 开发 CLI 客户端 (`model-cli share`, `model-cli download`)
- [ ] 添加前端界面 (Vue 3 + Element Plus)
- [ ] 性能测试 (10GB+ 文件传输)

---

**项目**: llmpt - 大模型 P2P 分享站  
**阶段**: Step 2 - Tracker Server ✅  
**更新**: 2026-02-05
