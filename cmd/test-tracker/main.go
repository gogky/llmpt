package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"llmpt/internal/tracker"
)

func main() {
	fmt.Println("🧪 Testing Tracker Implementation...")
	fmt.Println()

	// 测试 1: Bencode 编码
	fmt.Println("📝 Test 1: Bencode Encoding")
	testBencode()
	fmt.Println()

	// 测试 2: Compact Peer 格式
	fmt.Println("📦 Test 2: Compact Peer Format")
	testCompactPeer()
	fmt.Println()

	// 测试 3: Announce 请求（需要先启动 Tracker Server）
	fmt.Println("🌐 Test 3: Announce Request")
	fmt.Println("请先启动 Tracker Server: cd cmd/tracker && go run main.go")
	fmt.Println("然后运行测试: testAnnounce()")
	// testAnnounce()
	fmt.Println()

	fmt.Println("✅ All tests completed!")
}

// testBencode 测试 Bencode 编码
func testBencode() {
	// 测试字符串编码
	str := "spam"
	encoded := tracker.EncodeString(str)
	fmt.Printf("String: %s -> %s\n", str, string(encoded))
	
	// 测试整数编码
	num := int64(42)
	encoded = tracker.EncodeInt(num)
	fmt.Printf("Int: %d -> %s\n", num, string(encoded))
	
	// 测试字典编码
	dict := map[string][]byte{
		"interval": tracker.EncodeInt(1800),
		"complete": tracker.EncodeInt(5),
		"incomplete": tracker.EncodeInt(10),
	}
	encoded = tracker.EncodeDict(dict)
	fmt.Printf("Dict: %s\n", string(encoded))
}

// testCompactPeer 测试紧凑格式 Peer 编码
func testCompactPeer() {
	// 测试单个 Peer
	ip := "192.168.1.100"
	port := 6881
	compact, err := tracker.CompactPeer(ip, port)
	if err != nil {
		fmt.Printf("❌ CompactPeer failed: %v\n", err)
		return
	}
	fmt.Printf("Peer: %s:%d -> %s (length: %d bytes)\n", ip, port, hex.EncodeToString(compact), len(compact))
	
	// 解码验证
	decodedIP, decodedPort, err := tracker.DecompactPeer(compact)
	if err != nil {
		fmt.Printf("❌ DecompactPeer failed: %v\n", err)
		return
	}
	fmt.Printf("Decoded: %s:%d\n", decodedIP, decodedPort)
	
	if decodedIP != ip || decodedPort != port {
		fmt.Printf("❌ Mismatch! Expected %s:%d, got %s:%d\n", ip, port, decodedIP, decodedPort)
		return
	}
	
	fmt.Println("✅ Single peer test passed")
	
	// 测试多个 Peer
	peers := []string{
		"192.168.1.100:6881",
		"10.0.0.5:51413",
		"172.16.0.20:8999",
	}
	
	compactPeers, err := tracker.CompactPeers(peers)
	if err != nil {
		fmt.Printf("❌ CompactPeers failed: %v\n", err)
		return
	}
	
	fmt.Printf("Multiple Peers (%d): %s (length: %d bytes)\n", len(peers), hex.EncodeToString(compactPeers), len(compactPeers))
	
	// 解码验证
	decodedPeers, err := tracker.DecompactPeers(compactPeers)
	if err != nil {
		fmt.Printf("❌ DecompactPeers failed: %v\n", err)
		return
	}
	
	fmt.Printf("Decoded Peers: %v\n", decodedPeers)
	
	for i, peer := range peers {
		if decodedPeers[i] != peer {
			fmt.Printf("❌ Mismatch at index %d! Expected %s, got %s\n", i, peer, decodedPeers[i])
			return
		}
	}
	
	fmt.Println("✅ Multiple peers test passed")
}

// testAnnounce 测试 Announce 请求
func testAnnounce() {
	// 模拟一个 info_hash
	infoHashBytes := []byte("test_info_hash_12345")
	infoHash := string(infoHashBytes)
	
	// 模拟 Peer ID
	peerID := "test_peer_00000001"
	
	// 构建 Announce URL
	baseURL := "http://localhost:8080/announce"
	params := url.Values{}
	params.Add("info_hash", infoHash)
	params.Add("peer_id", peerID)
	params.Add("port", "6881")
	params.Add("uploaded", "0")
	params.Add("downloaded", "0")
	params.Add("left", "1000000")
	params.Add("compact", "1")
	params.Add("event", "started")
	
	announceURL := baseURL + "?" + params.Encode()
	
	fmt.Printf("🔗 Announce URL: %s\n", announceURL)
	
	// 发送请求
	resp, err := http.Get(announceURL)
	if err != nil {
		fmt.Printf("❌ Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Read response failed: %v\n", err)
		return
	}
	
	fmt.Printf("📥 Response (%d bytes):\n", len(body))
	fmt.Println(string(body))
	
	// 检查是否包含 "failure reason"
	if bytes.Contains(body, []byte("failure reason")) {
		fmt.Println("❌ Tracker returned an error")
		return
	}
	
	// 检查是否包含必需字段
	requiredFields := []string{"interval", "complete", "incomplete", "peers"}
	for _, field := range requiredFields {
		if !bytes.Contains(body, []byte(field)) {
			fmt.Printf("❌ Missing required field: %s\n", field)
			return
		}
	}
	
	fmt.Println("✅ Announce test passed")
	
	// 测试多个客户端
	fmt.Println("\n🔄 Testing multiple clients...")
	testMultiplePeers()
}

// testMultiplePeers 测试多个 Peer
func testMultiplePeers() {
	infoHashBytes := []byte("test_info_hash_12345")
	infoHash := string(infoHashBytes)
	
	// 模拟 3 个客户端
	for i := 1; i <= 3; i++ {
		peerID := fmt.Sprintf("test_peer_%08d", i)
		port := 6880 + i
		
		params := url.Values{}
		params.Add("info_hash", infoHash)
		params.Add("peer_id", peerID)
		params.Add("port", fmt.Sprintf("%d", port))
		params.Add("uploaded", "0")
		params.Add("downloaded", "0")
		params.Add("left", "1000000")
		params.Add("compact", "1")
		params.Add("event", "started")
		
		announceURL := "http://localhost:8080/announce?" + params.Encode()
		
		resp, err := http.Get(announceURL)
		if err != nil {
			fmt.Printf("❌ Peer %d request failed: %v\n", i, err)
			continue
		}
		
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		fmt.Printf("✅ Peer %d registered: %s:%d\n", i, peerID, port)
		
		// 解析响应中的 peers 数量
		bodyStr := string(body)
		if strings.Contains(bodyStr, "peers") {
			fmt.Printf("   Response: %d bytes\n", len(body))
		}
	}
	
	fmt.Println("✅ Multiple peers test completed")
}
