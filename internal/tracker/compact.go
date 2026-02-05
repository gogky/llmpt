package tracker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// CompactPeer 将 Peer 信息编码为紧凑格式（BEP-0023）
// 格式: 4字节 IP + 2字节 Port (大端序)
// 例如: 192.168.1.100:6881 -> [0xC0, 0xA8, 0x01, 0x64, 0x1A, 0xE1]
func CompactPeer(ip string, port int) ([]byte, error) {
	// 解析 IP 地址
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	// 转换为 IPv4（4 字节）
	ipv4 := parsedIP.To4()
	if ipv4 == nil {
		return nil, fmt.Errorf("not an IPv4 address: %s", ip)
	}

	// 构建紧凑格式
	buf := make([]byte, 6)
	copy(buf[:4], ipv4)                                // 前 4 字节: IP
	binary.BigEndian.PutUint16(buf[4:6], uint16(port)) // 后 2 字节: Port

	return buf, nil
}

// CompactPeers 将多个 Peer 编码为紧凑格式
// 输入: peers 格式为 "IP:Port" 的字符串数组
// 输出: 所有 Peer 的紧凑表示拼接在一起
func CompactPeers(peers []string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	buf.Grow(len(peers) * 6) // 预分配空间

	for _, peer := range peers {
		// 解析 "IP:Port"
		parts := strings.Split(peer, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid peer format: %s", peer)
		}

		ip := parts[0]
		port, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid port in peer %s: %w", peer, err)
		}

		// 编码单个 Peer
		compactPeer, err := CompactPeer(ip, port)
		if err != nil {
			return nil, fmt.Errorf("failed to compact peer %s: %w", peer, err)
		}

		buf.Write(compactPeer)
	}

	return buf.Bytes(), nil
}

// DecompactPeer 解码单个紧凑格式的 Peer
func DecompactPeer(data []byte) (string, int, error) {
	if len(data) != 6 {
		return "", 0, fmt.Errorf("invalid compact peer length: %d", len(data))
	}

	ip := net.IP(data[:4]).String()
	port := int(binary.BigEndian.Uint16(data[4:6]))

	return ip, port, nil
}

// DecompactPeers 解码多个紧凑格式的 Peer
func DecompactPeers(data []byte) ([]string, error) {
	if len(data)%6 != 0 {
		return nil, fmt.Errorf("invalid compact peers length: %d", len(data))
	}

	numPeers := len(data) / 6
	peers := make([]string, 0, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * 6
		ip, port, err := DecompactPeer(data[offset : offset+6])
		if err != nil {
			return nil, fmt.Errorf("failed to decompact peer at index %d: %w", i, err)
		}
		peers = append(peers, fmt.Sprintf("%s:%d", ip, port))
	}

	return peers, nil
}
