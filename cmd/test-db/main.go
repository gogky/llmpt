package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"llmpt/internal/config"
	"llmpt/internal/database"
	"llmpt/internal/models"
)

func main() {
	fmt.Println("=== 数据库连接测试 ===")
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化数据库连接
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	fmt.Println("\n=== 测试 MongoDB 操作 ===\n")
	testMongoDB(db)

	fmt.Println("\n=== 测试 Redis 操作 ===\n")
	testRedis(db)

	fmt.Println("\n✓ 所有测试完成!")
}

func testMongoDB(db *database.DB) {
	ctx := context.Background()

	// 创建测试种子
	testTorrent := &models.Torrent{
		Name:        "Test-Llama-3-8B",
		InfoHash:    "1234567890abcdef1234567890abcdef12345678",
		TotalSize:   15000000000, // 15GB
		FileCount:   120,
		MagnetLink:  "magnet:?xt=urn:btih:1234567890abcdef1234567890abcdef12345678",
		PieceLength: 8388608, // 8MB
		CreatedAt:   time.Now(),
	}

	// 插入种子
	collection := db.MongoDB.TorrentsCollection()
	result, err := collection.InsertOne(ctx, testTorrent)
	if err != nil {
		log.Printf("Failed to insert torrent: %v", err)
	} else {
		fmt.Printf("✓ Inserted test torrent with ID: %v\n", result.InsertedID)
	}

	// 查询种子
	var found models.Torrent
	err = collection.FindOne(ctx, map[string]interface{}{"info_hash": testTorrent.InfoHash}).Decode(&found)
	if err != nil {
		log.Printf("Failed to find torrent: %v", err)
	} else {
		fmt.Printf("✓ Found torrent: %s (Size: %.2f GB)\n", found.Name, float64(found.TotalSize)/1e9)
	}

	// 清理测试数据
	_, err = collection.DeleteOne(ctx, map[string]interface{}{"info_hash": testTorrent.InfoHash})
	if err != nil {
		log.Printf("Failed to delete test torrent: %v", err)
	} else {
		fmt.Println("✓ Cleaned up test data")
	}
}

func testRedis(db *database.DB) {
	ctx := context.Background()
	testInfoHash := "test1234567890abcdef1234567890abcdef12"

	// 添加 Peer
	peers := []string{
		"192.168.1.100:6881",
		"192.168.1.101:6881",
		"192.168.1.102:6881",
	}

	for _, peer := range peers {
		err := db.Redis.AddPeer(ctx, testInfoHash, peer, 30*time.Minute)
		if err != nil {
			log.Printf("Failed to add peer: %v", err)
		}
	}
	fmt.Printf("✓ Added %d test peers\n", len(peers))

	// 获取 Peer 列表
	foundPeers, err := db.Redis.GetPeers(ctx, testInfoHash, 0)
	if err != nil {
		log.Printf("Failed to get peers: %v", err)
	} else {
		fmt.Printf("✓ Found %d peers: %v\n", len(foundPeers), foundPeers)
	}

	// 获取 Peer 数量
	count, err := db.Redis.GetPeerCount(ctx, testInfoHash)
	if err != nil {
		log.Printf("Failed to get peer count: %v", err)
	} else {
		fmt.Printf("✓ Peer count: %d\n", count)
	}

	// 更新统计信息
	err = db.Redis.UpdateStats(ctx, testInfoHash, 10, 5, 100)
	if err != nil {
		log.Printf("Failed to update stats: %v", err)
	} else {
		fmt.Println("✓ Updated stats (seeders: 10, leechers: 5, completed: 100)")
	}

	// 获取统计信息
	stats, err := db.Redis.GetStats(ctx, testInfoHash)
	if err != nil {
		log.Printf("Failed to get stats: %v", err)
	} else {
		fmt.Printf("✓ Stats: %v\n", stats)
	}

	// 清理测试数据
	db.Redis.Client.Del(ctx, fmt.Sprintf("tracker:peers:%s", testInfoHash))
	db.Redis.Client.Del(ctx, fmt.Sprintf("tracker:stats:%s", testInfoHash))
	fmt.Println("✓ Cleaned up test data")
}
