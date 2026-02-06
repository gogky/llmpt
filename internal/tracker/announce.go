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

	// 构建 Peer 标识 (IP:Port)
	peer := fmt.Sprintf("%s:%d", clientIP, req.Port)

	// 处理不同事件
	switch req.Event {
	case "stopped":
		// 客户端停止，移除 Peer
		if err := h.db.Redis.RemovePeer(ctx, req.InfoHash, peer); err != nil {
			fmt.Printf("failed to remove peer: %v\n", err)
		}
		h.sendSuccess(w, req, []string{}, 0, 0)
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

	// 更新统计信息
	var seeders, leechers int64 = 0, 0

	// 判断是 Seeder 还是 Leecher
	if req.Left == 0 {
		seeders = 1
	} else {
		leechers = 1
	}

	// 获取当前统计信息并更新
	if currentStatsMap, err := h.db.Redis.GetStats(ctx, req.InfoHash); err == nil && len(currentStatsMap) > 0 {
		// 如果已有统计信息，则累加
		if s, ok := currentStatsMap["seeders"]; ok {
			if val, _ := strconv.ParseInt(s, 10, 64); val > 0 {
				seeders += val
			}
		}
		if l, ok := currentStatsMap["leechers"]; ok {
			if val, _ := strconv.ParseInt(l, 10, 64); val > 0 {
				leechers += val
			}
		}
	}

	if err := h.db.Redis.UpdateStats(ctx, req.InfoHash, seeders, leechers, 0); err != nil {
		fmt.Printf("failed to update stats: %v\n", err)
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
	filteredPeers := []string{}
	for _, p := range peers {
		if p != peer {
			filteredPeers = append(filteredPeers, p)
		}
	}

	// 限制返回数量
	if len(filteredPeers) > numWant {
		filteredPeers = filteredPeers[:numWant]
	}

	// 获取统计信息
	currentStatsMap, err := h.db.Redis.GetStats(ctx, req.InfoHash)
	var finalSeeders, finalLeechers int64 = 0, 0
	if err == nil && len(currentStatsMap) > 0 {
		if s, ok := currentStatsMap["seeders"]; ok {
			finalSeeders, _ = strconv.ParseInt(s, 10, 64)
		}
		if l, ok := currentStatsMap["leechers"]; ok {
			finalLeechers, _ = strconv.ParseInt(l, 10, 64)
		}
	}

	// 发送响应
	h.sendSuccess(w, req, filteredPeers, finalSeeders, finalLeechers)
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

	// 转换 info_hash 为 Hex 字符串（URL 编码的是原始字节）
	infoHashHex := hex.EncodeToString([]byte(infoHash))

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
