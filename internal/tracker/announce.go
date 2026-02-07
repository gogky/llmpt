package tracker

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"llmpt/internal/database"
	"llmpt/internal/models"
)

// Handler Tracker HTTP 处理器
type Handler struct {
	db *database.DB
}

// NewHandler 创建 Tracker 处理器
func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// Announce 处理 /announce 请求（BEP-0003 核心接口）
// GET /announce?info_hash=...&peer_id=...&port=...&uploaded=...&downloaded=...&left=...&event=...&compact=1
func (h *Handler) Announce(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 解析请求参数
	req, err := parseAnnounceRequest(r)
	if err != nil {
		h.sendError(w, fmt.Sprintf("invalid request: %v", err))
		return
	}

	// 获取客户端 IP
	clientIP := getClientIP(r)

	// 构建 Peer 标识
	// 必须使用 net.JoinHostPort，它会自动给 IPv6 地址加方括号
	// IPv4: "192.168.1.100:6881"
	// IPv6: "[2402:4e00:1820:400:...]:6881"
	peer := net.JoinHostPort(clientIP, strconv.Itoa(req.Port))

	fmt.Printf("[announce] peer_id=%s ip=%s peer=%s event=%s left=%d\n",
		req.PeerID, clientIP, peer, req.Event, req.Left)

	// 处理不同事件
	switch req.Event {
	case "stopped":
		// 客户端停止，移除 Peer
		if err := h.db.Redis.RemovePeer(ctx, req.InfoHash, peer); err != nil {
			fmt.Printf("failed to remove peer: %v\n", err)
		}
		// 重新计算统计
		seeders, leechers := h.countStats(ctx, req.InfoHash)
		h.sendSuccess(w, req, []string{}, seeders, leechers)
		return

	case "completed":
		// 下载完成，增加完成计数
		if err := h.db.Redis.IncrementCompleted(ctx, req.InfoHash); err != nil {
			fmt.Printf("failed to increment completed: %v\n", err)
		}
	}

	// 更新 Peer 信息（自动设置 30 分钟 TTL）
	if err := h.db.Redis.AddPeer(ctx, req.InfoHash, peer); err != nil {
		h.sendError(w, fmt.Sprintf("failed to add peer: %v", err))
		return
	}

	// 获取其他 Peer（排除自己）
	numWant := req.NumWant
	if numWant == 0 || numWant > 50 {
		numWant = 50 // 默认返回 50 个
	}

	peers, err := h.db.Redis.GetPeers(ctx, req.InfoHash, int64(numWant+1)) // 多取 1 个，用于排除自己
	if err != nil {
		h.sendError(w, fmt.Sprintf("failed to get peers: %v", err))
		return
	}

	// 排除当前客户端自己
	filteredPeers := make([]string, 0, len(peers))
	for _, p := range peers {
		if p != peer {
			filteredPeers = append(filteredPeers, p)
		}
	}

	// 限制返回数量
	if len(filteredPeers) > numWant {
		filteredPeers = filteredPeers[:numWant]
	}

	// 从 Redis 直接计算统计信息（Peer 集合中的实际数量）
	seeders, leechers := h.countStats(ctx, req.InfoHash)

	fmt.Printf("[announce] info_hash=%s total_peers=%d returned=%d seeders=%d leechers=%d\n",
		req.InfoHash[:16]+"...", len(peers), len(filteredPeers), seeders, leechers)

	// 发送响应
	h.sendSuccess(w, req, filteredPeers, seeders, leechers)
}

// parseAnnounceRequest 解析 Announce 请求参数
func parseAnnounceRequest(r *http.Request) (*models.AnnounceRequest, error) {
	query := r.URL.Query()

	// 必需参数
	infoHash := query.Get("info_hash")
	peerID := query.Get("peer_id")
	portStr := query.Get("port")

	if infoHash == "" || peerID == "" || portStr == "" {
		return nil, fmt.Errorf("missing required parameters")
	}

	// 解析端口
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port: %s", portStr)
	}

	// 转换 info_hash 为 Hex 字符串
	// 真实 BT 客户端发送 20 字节原始二进制（URL 编码），需要转为 hex
	// curl 测试可能直接发送 40 字符 hex 字符串，不需要再转
	infoHashHex := normalizeInfoHash(infoHash)

	req := &models.AnnounceRequest{
		InfoHash:   infoHashHex,
		PeerID:     peerID,
		Port:       port,
		Uploaded:   parseInt64(query.Get("uploaded")),
		Downloaded: parseInt64(query.Get("downloaded")),
		Left:       parseInt64(query.Get("left")),
		Event:      query.Get("event"),
		Compact:    parseInt(query.Get("compact")),
		NumWant:    parseInt(query.Get("numwant")),
	}

	return req, nil
}

// getClientIP 获取客户端真实 IP
func getClientIP(r *http.Request) string {
	// 优先使用查询参数中的 IP（客户端可能在 NAT 后）
	if ip := r.URL.Query().Get("ip"); ip != "" {
		return ip
	}

	// 检查 X-Forwarded-For 头（代理）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// 检查 X-Real-IP 头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// 使用连接的远程地址
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// sendSuccess 发送成功响应（支持 IPv4 和 IPv6，BEP-0007）
func (h *Handler) sendSuccess(w http.ResponseWriter, req *models.AnnounceRequest, peers []string, seeders, leechers int64) {
	// 构建响应字典
	response := make(map[string][]byte)

	// 设置心跳间隔（秒）
	response["interval"] = EncodeInt(1800)       // 30 分钟
	response["min interval"] = EncodeInt(900)    // 15 分钟
	response["complete"] = EncodeInt(seeders)    // Seeders
	response["incomplete"] = EncodeInt(leechers) // Leechers

	// 分离 IPv4 和 IPv6 Peer（BEP-0007）
	ipv4Peers, ipv6Peers := SeparatePeersByIPVersion(peers)

	// 处理 Peer 列表
	if req.Compact == 1 {
		// Compact 模式：返回二进制格式（BEP-0023 + BEP-0007）

		// IPv4 Peers
		if len(ipv4Peers) > 0 {
			compactPeers, err := CompactPeersIPv4(ipv4Peers)
			if err != nil {
				h.sendError(w, fmt.Sprintf("failed to compact IPv4 peers: %v", err))
				return
			}
			response["peers"] = EncodeBytes(compactPeers)
		} else {
			// 即使没有 IPv4 peer，也返回空字符串
			response["peers"] = EncodeBytes([]byte{})
		}

		// IPv6 Peers（BEP-0007）
		if len(ipv6Peers) > 0 {
			compactPeers6, err := CompactPeersIPv6(ipv6Peers)
			if err != nil {
				h.sendError(w, fmt.Sprintf("failed to compact IPv6 peers: %v", err))
				return
			}
			response["peers6"] = EncodeBytes(compactPeers6)
		}
	} else {
		// 标准模式：返回字典列表
		// IPv4 Peers
		ipv4PeerList := make([][]byte, 0, len(ipv4Peers))
		for _, peer := range ipv4Peers {
			host, portStr, err := net.SplitHostPort(peer)
			if err != nil {
				continue
			}
			port, _ := strconv.Atoi(portStr)

			peerDict := make(map[string][]byte)
			peerDict["ip"] = EncodeString(host)
			peerDict["port"] = EncodeInt(int64(port))
			peerDict["peer id"] = EncodeString("") // 可选

			ipv4PeerList = append(ipv4PeerList, EncodeDict(peerDict))
		}
		response["peers"] = EncodeList(ipv4PeerList)

		// IPv6 Peers（BEP-0007）
		if len(ipv6Peers) > 0 {
			ipv6PeerList := make([][]byte, 0, len(ipv6Peers))
			for _, peer := range ipv6Peers {
				host, portStr, err := net.SplitHostPort(peer)
				if err != nil {
					continue
				}
				port, _ := strconv.Atoi(portStr)

				peerDict := make(map[string][]byte)
				peerDict["ip"] = EncodeString(host)
				peerDict["port"] = EncodeInt(int64(port))
				peerDict["peer id"] = EncodeString("") // 可选

				ipv6PeerList = append(ipv6PeerList, EncodeDict(peerDict))
			}
			response["peers6"] = EncodeList(ipv6PeerList)
		}
	}

	// 编码为 Bencode
	data := EncodeDict(response)

	// 发送响应
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// sendError 发送错误响应
func (h *Handler) sendError(w http.ResponseWriter, reason string) {
	response := make(map[string][]byte)
	response["failure reason"] = EncodeString(reason)

	data := EncodeDict(response)

	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(http.StatusOK) // Tracker 错误仍返回 200
	w.Write(data)
}

// normalizeInfoHash 统一处理 info_hash 参数
// 真实 BT 客户端: 发送 20 字节原始二进制 → 需要 hex 编码为 40 字符
// curl 测试: 可能直接发送 40 字符 hex 字符串 → 直接使用
func normalizeInfoHash(raw string) string {
	// 如果长度是 40 且全是合法 hex 字符，说明已经是 hex 格式
	if len(raw) == 40 && isHexString(raw) {
		return strings.ToLower(raw)
	}

	// 否则是原始二进制（20 字节），转为 hex
	return hex.EncodeToString([]byte(raw))
}

// isHexString 检查字符串是否全部由合法 hex 字符组成
func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// parseInt 解析整数参数
func parseInt(s string) int {
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
	return n
}

// parseInt64 解析 int64 参数
func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}

// countStats 从 Redis 直接计算统计信息
// 使用 Peer 集合的实际成员数量，不再手动累加，避免数字膨胀
func (h *Handler) countStats(ctx context.Context, infoHash string) (seeders, leechers int64) {
	// 直接从 Redis Set 获取当前 Peer 总数
	totalPeers, err := h.db.Redis.GetPeerCount(ctx, infoHash)
	if err != nil {
		return 0, 0
	}

	// 简化处理：所有在线 Peer 都算 seeders
	// 因为当前 Redis Set 中只存了 "IP:Port"，没有区分 seeder/leecher
	// 真实的区分需要在 Redis 中额外存储 left 值（后续优化）
	seeders = totalPeers
	leechers = 0
	return
}

// StartCleanup 启动定期清理过期 Peer 的任务
// 虽然 Redis TTL 会自动删除，但这个任务可以用于更新统计信息
func (h *Handler) StartCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// TODO: 可以在这里实现统计信息的重新计算
			fmt.Println("Cleanup task running...")
		}
	}
}
