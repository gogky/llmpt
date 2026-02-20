package tracker

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
)

// CompactPeer 将 Peer 信息编码为紧凑格式（BEP-0023）
// 自动检测 IPv4 或 IPv6
// IPv4: 6 字节 (4字节 IP + 2字节 Port)
// IPv6: 18 字节 (16字节 IP + 2字节 Port)
func CompactPeer(ip string, port int) ([]byte, error) {
	// 解析 IP 地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	// 尝试转换为 IPv4
	if ipv4 := parsedIP.To4(); ipv4 != nil {
		return compactPeerIPv4(ipv4, port), nil
	}

	// 否则作为 IPv6 处理
	if ipv6 := parsedIP.To16(); ipv6 != nil {
		return compactPeerIPv6(ipv6, port), nil
	}

	return nil, fmt.Errorf("unsupported IP format: %s", ip)
}

// compactPeerIPv4 编码 IPv4 Peer（6 字节）
// 格式: 4字节 IP + 2字节 Port (大端序)
// 例如: 192.168.1.100:6881 -> [0xC0, 0xA8, 0x01, 0x64, 0x1A, 0xE1]
func compactPeerIPv4(ip net.IP, port int) []byte {
	buf := make([]byte, 6)
	copy(buf[:4], ip)                                  // 前 4 字节: IP
	binary.BigEndian.PutUint16(buf[4:6], uint16(port)) // 后 2 字节: Port
	return buf
}

// compactPeerIPv6 编码 IPv6 Peer（18 字节，BEP-0007）
// 格式: 16字节 IP + 2字节 Port (大端序)
// 例如: [2001:db8::1]:6881 -> 16字节IPv6 + 2字节端口
func compactPeerIPv6(ip net.IP, port int) []byte {
	buf := make([]byte, 18)
	copy(buf[:16], ip)                                   // 前 16 字节: IPv6
	binary.BigEndian.PutUint16(buf[16:18], uint16(port)) // 后 2 字节: Port
	return buf
}

// CompactPeers 将多个 Peer 编码为紧凑格式
// 输入: peers 格式为 "IP:Port" 的字符串数组
// 输出: 所有 Peer 的紧凑表示拼接在一起
// 注意: 混合 IPv4 和 IPv6 时，应该分别返回（peers 和 peers6）
func CompactPeers(peers []string) ([]byte, error) {
	buf := make([]byte, 0, len(peers)*6) // 预分配空间（假设大多数是 IPv4）

	for _, peer := range peers {
		// 解析 "IP:Port"
		host, portStr, err := net.SplitHostPort(peer)
		if err != nil {
			return nil, fmt.Errorf("invalid peer format: %s", peer)
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("invalid port in peer %s: %w", peer, err)
		}

		// 编码单个 Peer
		compactPeer, err := CompactPeer(host, port)
		if err != nil {
			return nil, fmt.Errorf("failed to compact peer %s: %w", peer, err)
		}

		buf = append(buf, compactPeer...)
	}

	return buf, nil
}

// CompactPeersIPv4 仅编码 IPv4 Peer（用于分离返回）
func CompactPeersIPv4(peers []string) ([]byte, error) {
	buf := make([]byte, 0, len(peers)*6)

	for _, peer := range peers {
		host, portStr, err := net.SplitHostPort(peer)
		if err != nil {
			continue // 跳过无效格式
		}

		parsedIP := net.ParseIP(host)
		if parsedIP == nil || parsedIP.To4() == nil {
			continue // 跳过非 IPv4
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		compactPeer := compactPeerIPv4(parsedIP.To4(), port)
		buf = append(buf, compactPeer...)
	}

	return buf, nil
}

// CompactPeersIPv6 仅编码 IPv6 Peer（用于分离返回，BEP-0007）
func CompactPeersIPv6(peers []string) ([]byte, error) {
	buf := make([]byte, 0, len(peers)*18)

	for _, peer := range peers {
		host, portStr, err := net.SplitHostPort(peer)
		if err != nil {
			continue // 跳过无效格式
		}

		parsedIP := net.ParseIP(host)
		if parsedIP == nil || parsedIP.To4() != nil {
			continue // 跳过 IPv4
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}

		compactPeer := compactPeerIPv6(parsedIP.To16(), port)
		buf = append(buf, compactPeer...)
	}

	return buf, nil
}

// DecompactPeer 解码单个紧凑格式的 Peer（自动检测 IPv4 或 IPv6）
func DecompactPeer(data []byte) (string, int, error) {
	switch len(data) {
	case 6:
		// IPv4
		ip := net.IP(data[:4]).String()
		port := int(binary.BigEndian.Uint16(data[4:6]))
		return ip, port, nil

	case 18:
		// IPv6
		ip := net.IP(data[:16]).String()
		port := int(binary.BigEndian.Uint16(data[16:18]))
		return ip, port, nil

	default:
		return "", 0, fmt.Errorf("invalid compact peer length: %d (expected 6 for IPv4 or 18 for IPv6)", len(data))
	}
}

// DecompactPeers 解码多个紧凑格式的 Peer（出于安全考虑，强烈建议直接调用 v4 或 v6 专有解析函数）
func DecompactPeers(data []byte) ([]string, error) {
	if len(data) == 0 {
		return []string{}, nil
	}

	peers := make([]string, 0)
	offset := 0

	for offset < len(data) {
		remaining := len(data) - offset

		var peerSize int
		// 优先严格判断 IPv4（因为 BT 中绝大多数混合探测都是要求 IPv4）
		// 如果必须混合探测，且长度刚好既能是 v4 也能是 v6 (如 18 字节)：为了避免把 v6 当成 3个 v4，
		// 这里最标准的做法是在外部由具体的业务决定。但在这里我们通过强行预判做隔离。
		// 由于该函数本身就不推荐在严谨场合下使用，如果有这种模棱两可的情况，返回错误要求业务明确。
		if remaining >= 18 && remaining%18 == 0 && remaining%6 == 0 {
			// 如果数据长度既是 18 的倍数，也是 6 的倍数，这在理论上无法 100% 安全推断
			// 绝大多数情况下如果走这里的猜测函数，其实都是 v4 (BEP-23)
			// 但安全起见，我们只能按 v4 试探解析。
			peerSize = 6
		} else if remaining%6 == 0 {
			peerSize = 6
		} else if remaining%18 == 0 {
			peerSize = 18
		} else {
			return nil, fmt.Errorf("invalid compact peers data length: %d (cannot safely infer IPv4 or IPv6), offset: %d", len(data), offset)
		}

		ip, port, err := DecompactPeer(data[offset : offset+peerSize])
		if err != nil {
			return nil, fmt.Errorf("failed to decompact peer at offset %d: %w", offset, err)
		}

		peers = append(peers, net.JoinHostPort(ip, strconv.Itoa(port)))
		offset += peerSize
	}

	return peers, nil
}

// DecompactPeersIPv4 解码纯 IPv4 Peer 列表
func DecompactPeersIPv4(data []byte) ([]string, error) {
	if len(data)%6 != 0 {
		return nil, fmt.Errorf("invalid IPv4 compact peers length: %d", len(data))
	}

	numPeers := len(data) / 6
	peers := make([]string, 0, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * 6
		ip := net.IP(data[offset : offset+4]).String()
		port := int(binary.BigEndian.Uint16(data[offset+4 : offset+6]))
		peers = append(peers, net.JoinHostPort(ip, strconv.Itoa(port)))
	}

	return peers, nil
}

// DecompactPeersIPv6 解码纯 IPv6 Peer 列表（BEP-0007）
func DecompactPeersIPv6(data []byte) ([]string, error) {
	if len(data)%18 != 0 {
		return nil, fmt.Errorf("invalid IPv6 compact peers length: %d", len(data))
	}

	numPeers := len(data) / 18
	peers := make([]string, 0, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * 18
		ip := net.IP(data[offset : offset+16]).String()
		port := int(binary.BigEndian.Uint16(data[offset+16 : offset+18]))
		peers = append(peers, net.JoinHostPort(ip, strconv.Itoa(port)))
	}

	return peers, nil
}

// IsIPv6 判断 IP 地址字符串是否为 IPv6
func IsIPv6(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	return ip != nil && ip.To4() == nil
}

// SeparatePeersByIPVersion 将 Peer 列表分离为 IPv4 和 IPv6
func SeparatePeersByIPVersion(peers []string) (ipv4Peers, ipv6Peers []string) {
	for _, peer := range peers {
		host, _, err := net.SplitHostPort(peer)
		if err != nil {
			continue
		}

		if IsIPv6(host) {
			ipv6Peers = append(ipv6Peers, peer)
		} else {
			ipv4Peers = append(ipv4Peers, peer)
		}
	}
	return
}
