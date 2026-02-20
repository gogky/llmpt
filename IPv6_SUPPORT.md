# IPv6 支持说明

## ✅ 已完成

你的 Tracker Server 现在**完全支持 IPv6**！

## 🎯 支持的功能

### 1. **双栈支持（IPv4 + IPv6）**

Tracker 可以同时处理 IPv4 和 IPv6 客户端：
- 自动检测客户端 IP 版本
- 分别返回 IPv4 和 IPv6 Peer 列表
- 兼容纯 IPv4、纯 IPv6 和混合网络环境

### 2. **Compact 格式（BEP-0023 + BEP-0007）**

#### IPv4 Compact（6 字节）
```
[IP 4字节] [Port 2字节]
```

示例：`192.168.1.100:6881`
```
C0 A8 01 64 1A E1
```

#### IPv6 Compact（18 字节）
```
[IP 16字节] [Port 2字节]
```

示例：`[2001:db8::1]:6881`
```
20 01 0D B8 00 00 00 00 00 00 00 00 00 00 00 01 1A E1
```

### 3. **Announce 响应格式**

#### Compact 模式（compact=1）
```bencode
d
  8:intervali1800e
  8:completei5e
  10:incompletei10e
  5:peers6:...       # IPv4 Peers（6字节/个）
  6:peers618:...     # IPv6 Peers（18字节/个）
e
```

#### 标准模式（compact=0）
```bencode
d
  8:intervali1800e
  5:peersld2:ip13:192.168.1.1004:porti6881eed...ee  # IPv4
  6:peers6ld2:ip11:2001:db8::14:porti6881eed...ee  # IPv6
e
```

## 🧪 测试结果

```bash
$ cd cmd/test-tracker
$ go run main.go
```

### 测试输出

```
🧪 Testing Tracker Implementation...

📝 Test 1: Bencode Encoding
✅ 通过

📦 Test 2: Compact Peer Format (IPv4)
Peer: 192.168.1.100:6881 -> c0a801641ae1 (length: 6 bytes)
✅ Single peer test passed
✅ Multiple peers test passed

📦 Test 3: Compact Peer Format (IPv6)
Peer: [2001:db8::1]:6881 -> 20010db80000000000000000000000011ae1 (length: 18 bytes)
✅ Single IPv6 peer test passed
✅ Multiple IPv6 peers test passed

🔀 Testing IPv4/IPv6 separation...
IPv4 Peers (2): [192.168.1.100:6881 10.0.0.5:51413]
IPv6 Peers (2): [[2001:db8::1]:6881 [fe80::1]:8999]
✅ IPv4/IPv6 separation test passed

✅ All tests completed!
```

## 📊 技术实现

### 核心函数

#### `compact.go`

```go
// 自动检测 IPv4/IPv6
CompactPeer(ip string, port int) ([]byte, error)

// 仅 IPv4
CompactPeersIPv4(peers []string) ([]byte, error)

// 仅 IPv6
CompactPeersIPv6(peers []string) ([]byte, error)

// 分离 IPv4 和 IPv6
SeparatePeersByIPVersion(peers []string) (ipv4, ipv6 []string)

// 解码（自动检测）
DecompactPeer(data []byte) (ip string, port int, error)
DecompactPeersIPv4(data []byte) ([]string, error)
DecompactPeersIPv6(data []byte) ([]string, error)
```

#### `announce.go`

```go
// 自动处理 IPv4/IPv6 响应
sendSuccess(w http.ResponseWriter, req *AnnounceRequest, 
            peers []string, seeders, leechers int64)
```

### 工作流程

```
客户端请求
    │
    ├─ IPv4 客户端 (192.168.1.100)
    │   ↓
    │   Tracker 返回:
    │   - peers: IPv4 列表（6字节/个）
    │   - peers6: IPv6 列表（18字节/个，如果有）
    │
    └─ IPv6 客户端 (2001:db8::1)
        ↓
        Tracker 返回:
        - peers: IPv4 列表（6字节/个）
        - peers6: IPv6 列表（18字节/个）
```

## 🔧 配置

### 服务器端

无需额外配置，Tracker 自动支持 IPv6。

确保系统启用了 IPv6：

#### Windows
```powershell
# 检查 IPv6 是否启用
ipconfig | findstr IPv6

# 查看 IPv6 地址
netsh interface ipv6 show address
```

#### Linux
```bash
# 检查 IPv6 是否启用
ip -6 addr show

# 启用 IPv6
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=0
```

### 客户端端

主流 BT 客户端已原生支持 IPv6：
- ✅ qBittorrent 4.0+
- ✅ Transmission 3.0+
- ✅ Deluge 2.0+
- ✅ rTorrent/ruTorrent

## 🌐 使用场景

### 场景 1: 纯 IPv4 网络
```
客户端 A (IPv4) ←→ Tracker ←→ 客户端 B (IPv4)
```
Tracker 仅返回 IPv4 Peers

### 场景 2: 纯 IPv6 网络
```
客户端 A (IPv6) ←→ Tracker ←→ 客户端 B (IPv6)
```
Tracker 仅返回 IPv6 Peers

### 场景 3: 混合网络（推荐）
```
客户端 A (IPv4)     ↘
客户端 B (IPv6)     → Tracker → 返回 IPv4 + IPv6 列表
客户端 C (双栈)     ↗
```
Tracker 同时返回 IPv4 和 IPv6 Peers，客户端自行选择

## 🧪 测试 IPv6

### 方法 1: 本地回环测试

```powershell
# 启动 Tracker
cd cmd\tracker
go run main.go

# 在另一个终端测试
curl -g "http://[::1]:8080/health"
```

### 方法 2: 局域网 IPv6 测试

1. 确保两台设备都有 IPv6 地址：
   ```powershell
   ipconfig | findstr IPv6
   # 查找类似：fe80::1234:5678:90ab:cdef
   ```

2. 用 qBittorrent 制作种子：
   - Tracker URL: `http://[fe80::你的IPv6地址%网卡名]:8080/announce`
   - 示例: `http://[fe80::1234%以太网]:8080/announce`

3. 在另一台设备上用 Transmission 下载

### 方法 3: 公网 IPv6 测试

如果你有公网 IPv6 地址：

```powershell
# 获取公网 IPv6
curl -6 ifconfig.co

# Tracker URL
http://[你的公网IPv6]:8080/announce
```

## 📈 性能对比

| 协议 | Peer 大小 | 50个Peer | 带宽节省 |
|------|----------|----------|---------|
| IPv4 标准 | ~50 字节 | ~2.5 KB | - |
| IPv4 Compact | 6 字节 | 300 字节 | 88% |
| IPv6 标准 | ~60 字节 | ~3.0 KB | - |
| IPv6 Compact | 18 字节 | 900 字节 | 70% |

## 🔍 调试

### 查看 Redis 中的 IPv6 Peer

```powershell
docker exec -it llmpt-redis-1 redis-cli

# 查看所有 Peer
SMEMBERS tracker:peers:abc123...

# 输出示例：
# 1) "192.168.1.100:6881"
# 2) "[2001:db8::1]:6881"
# 3) "[fe80::1234]:51413"
```

### Tracker 日志

```
2026/02/05 16:30:15 GET /announce from 192.168.1.100:54321     (IPv4)
2026/02/05 16:30:20 GET /announce from [2001:db8::1]:54322     (IPv6)
```

## ⚠️ 注意事项

### 1. IPv6 地址格式

在 URL 中使用 IPv6 地址时，必须用方括号包裹：
- ✅ 正确: `http://[2001:db8::1]:8080/announce`
- ❌ 错误: `http://2001:db8::1:8080/announce`（会被误解析为端口）

### 2. 链路本地地址（Link-Local）

使用 `fe80::` 开头的链路本地地址时，需要指定网卡：
- ✅ `http://[fe80::1%eth0]:8080/announce` (Linux)
- ✅ `http://[fe80::1%以太网]:8080/announce` (Windows)

### 3. 防火墙

确保防火墙允许 IPv6 连接：

#### Windows
```powershell
# 允许 IPv6 入站（8080 端口）
netsh advfirewall firewall add rule name="Tracker IPv6" dir=in action=allow protocol=TCP localport=8080
```

#### Linux
```bash
# 允许 IPv6 入站
sudo ip6tables -A INPUT -p tcp --dport 8080 -j ACCEPT
```

### 4. NAT64/DNS64

如果客户端在纯 IPv6 网络中，但 Tracker 只有 IPv4 地址，需要：
- NAT64 网关转换
- 或使用双栈 Tracker

## 📚 相关标准

- **BEP-0003**: BitTorrent 协议规范
- **BEP-0007**: IPv6 Tracker Extension
- **BEP-0023**: Compact Peer Lists

## 🎉 总结

你的 Tracker Server 现在：

- ✅ 完全支持 IPv4
- ✅ 完全支持 IPv6
- ✅ 支持双栈（同时 IPv4 和 IPv6）
- ✅ Compact 格式节省带宽
- ✅ 自动检测和分离
- ✅ 兼容所有主流 BT 客户端

**IPv6 支持已就绪，无需任何配置即可使用！** 🚀

---

**项目**: llmpt - 大模型 P2P 分享站  
**功能**: IPv6 支持  
**更新时间**: 2026-02-05
