# V1.1 原型系统设计文档：大模型 P2P 分享站 (Model-PT)

## 1. 系统概述 (Overview)

本阶段目标是搭建一个**专为大文件（LLM 权重）优化**的私有 P2P 追踪服务器和配套客户端。

* **核心差异化：** 相比普通 BT 软件，本系统针对 10GB-100GB 级别的文件夹传输进行了参数调优（如加大分片大小），以降低元数据体积和 CPU 占用。
* **核心组件：**
1. **Tracker Server:** 兼容 BitTorrent BEP-0003/BEP-0023 标准的追踪服务器。
2. **Web Server:** 负责展示模型列表、接收 `.torrent` 元数据上传。
3. **Client CLI:** 负责文件分片（Chunking）、哈希校验、断点续传。



---

## 2. 数据库设计 (Database Schema)

### 2.1 MongoDB (存储模型元数据)

**库名：** `hf_p2p_v1`
**集合：** `torrents`

| 字段名 | 类型 | 说明 |
| --- | --- | --- |
| `_id` | ObjectId | 唯一主键 |
| `name` | String | 模型名称 (支持目录名，如 "Llama-3-8B/") |
| `info_hash` | String | **核心字段**，种子唯一指纹 (Hex) |
| `total_size` | Int64 | 文件/文件夹总大小 (Bytes) |
| `file_count` | Int | 包含的文件数量 |
| `magnet_link` | String | 磁力链接 |
| `piece_length` | Int | 分片大小 (Bytes)，用于统计分析 |
| `created_at` | Timestamp | 发布时间 |

### 2.2 Redis (存储 P2P 节点列表)

**Key 设计：**

* **存储 Peer 列表：** `tracker:peers:{info_hash}`
* **Type:** Set (集合)
* **Value:** 紧凑格式或字符串 `{IP}:{Port}`
* **TTL:** 30分钟 (心跳过期)


* **存储统计信息：** `tracker:stats:{info_hash}`
* **Type:** Hash
* **Fields:** `seeders`, `leechers`, `completed`



---

## 3. 接口设计 (API Design)

### 3.1 Web API (RESTful - 给前端用)

* **POST** `/api/v1/publish`
* **功能：** 发布新模型。
* **逻辑变化：** 接收客户端解析好的元数据，不再接收实体文件。


* **GET** `/api/v1/torrents`
* **功能：** 获取模型列表（带实时做种人数）。



### 3.2 Tracker API (BitTorrent 标准核心)

* **GET** `/announce`
* **标准兼容：** 必须支持 **Compact Response (BEP-0023)**。
* **输入参数：** `info_hash`, `peer_id`, `port`, `uploaded`, `downloaded`, `left`, `event`, `compact=1`。
* **逻辑：**
1. **记录：** 更新 Redis 中该 Peer 的心跳时间。
2. **清洗：** 移除已过期的 Peer（利用 Redis TTL 或惰性删除）。
3. **响应：** 从 Redis 随机取出 50 个 Peer。
* *关键点：* 如果请求带 `compact=1`，必须返回 **二进制流**（每个 Peer 占 6 字节：4字节 IP + 2字节 Port），而非 Bencoded 字典列表。这能极大减少带宽。







---

## 4. 客户端 (Go CLI) 详细设计

**核心依赖：**

* BT 协议库：`github.com/anacrolix/torrent`
* 进度条 UI：`github.com/schollz/progressbar/v3` 或 `bubbletea`

### 4.1 目录结构

```text
cmd/
  model-cli/
    main.go
pkg/
  p2p/
    client.go  (封装 Client, AddTorrent)
    create.go  (封装元数据生成逻辑)

```

### 4.2 核心命令逻辑

**命令 A：做种 (Share) - 针对大文件优化**
`./model-cli share --path ./Llama-3-70B --tracker http://your-server/announce`

1. **参数配置：**
* **Piece Size (分片大小)：** 强制设为 **4MB, 8MB 或 16MB**（普通 BT 默认为 256KB）。
* *理由：* 100GB 文件若用 256KB 分片，种子文件会超过 10MB，解析极慢。


* **Private Flag：** 设置 `Private: true`。禁止客户端通过 DHT 寻找节点，强制只走你的 Tracker。


2. **生成元数据：** 遍历目录下所有文件，计算 Hash，生成 InfoHash。
3. **上报：** 调用 Web API `/publish` 注册资源。
4. **启动做种：** 监听 TCP 端口，保持进程运行。

**命令 B：下载 (Download) - 增强稳定性**
`./model-cli download --magnet "magnet:?..." --out ./models`

1. **断点续传 (Resume)：**
* 启动前检查目标目录是否存在部分文件。
* 如有，进行 **Hash Check**（校验已下载分片），仅下载缺失部分。


2. **预分配磁盘空间：** 防止下载到 99% 磁盘空间不足。
3. **交互体验：** 使用进度条库显示：`[=====>......] 45% (2.5 MB/s) | Peers: 5`。

---

## 5. 前端设计 (Vue 3 + Element Plus)

保持 V1.0 的极简设计。

* **列表页优化：** 增加显示 "文件结构" 或 "包含文件数"，让用户知道下载的是一个文件夹还是单文件。

---

## 6. 开发步骤清单 (Revised Roadmap)

新增了 **Step 2.5** 用于协议验证，这是避免“闭门造车”的关键。

**Step 1: 基础设施 (1天)**

* 环境准备 (Go, Mongo, Redis)。

**Step 2: Tracker Server 实现 (3天)**

* 实现 `/announce` 接口。
* **重点：** 实现 Compact (Binary) 模式的 Peer 列表返回。

**Step 2.5: 协议兼容性验证 (关键一步)**

* *不写代码，仅测试。*
* 用 **qBittorrent** 制作一个种子，Tracker 填你的服务器地址。
* 用 **Transmission** (或另一台电脑的 qBittorrent) 下载。
* **验证：** 你的 Tracker 能否正确让这两个标准软件互相发现并传输？如果能，说明 Tracker 达标。

**Step 3: CLI 客户端开发 (4-5天)**

* 实现 `CreateTorrent` 逻辑（大分片、多文件支持）。
* 实现 `Download` 逻辑（集成进度条）。
* *自测：* CLI 做种 <-> qBittorrent 下载（确保你的 CLI 生成的种子是标准的）。

**Step 4: Web API & Frontend (3天)**

* 完成元数据上报和展示。

**Step 5: 联调与部署 (2天)**

* 真实局域网/公网环境测试大文件 (10GB+) 传输稳定性。

