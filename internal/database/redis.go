package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis Redis 客户端包装
type Redis struct {
	Client *redis.Client
}

// RedisPoolOptions Redis 连接池配置
type RedisPoolOptions struct {
	PoolSize     int
	MinIdleConns int
}

// NewRedis 创建新的 Redis 连接
// poolOpts 为 nil 时使用默认连接池配置（50/10）
func NewRedis(addr, password string, db int, poolOpts *RedisPoolOptions) (*Redis, error) {
	poolSize := 50
	minIdleConns := 10
	if poolOpts != nil {
		if poolOpts.PoolSize > 0 {
			poolSize = poolOpts.PoolSize
		}
		if poolOpts.MinIdleConns > 0 {
			minIdleConns = poolOpts.MinIdleConns
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConns,
		MaxRetries:   3,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	fmt.Printf("✓ Successfully connected to Redis (addr: %s)\n", addr)

	return &Redis{
		Client: client,
	}, nil
}

// Close 关闭 Redis 连接
func (r *Redis) Close() error {
	if r.Client != nil {
		return r.Client.Close()
	}
	return nil
}

// Tracker Peer 相关方法

// AddPeer 添加 Peer 到指定 info_hash 的集合
// TTL 默认 30 分钟
func (r *Redis) AddPeer(ctx context.Context, infoHash, peer string) error {
	key := fmt.Sprintf("tracker:peers:%s", infoHash)

	// 添加到集合
	if err := r.Client.SAdd(ctx, key, peer).Err(); err != nil {
		return err
	}

	// 设置 TTL
	return r.Client.Expire(ctx, key, 30*time.Minute).Err()
}

// GetPeers 获取指定 info_hash 的所有 Peer
// maxPeers: 最多返回的 Peer 数量（0 表示全部）
func (r *Redis) GetPeers(ctx context.Context, infoHash string, maxPeers int64) ([]string, error) {
	key := fmt.Sprintf("tracker:peers:%s", infoHash)

	if maxPeers <= 0 {
		// 返回所有
		return r.Client.SMembers(ctx, key).Result()
	}

	// 随机返回指定数量
	return r.Client.SRandMemberN(ctx, key, maxPeers).Result()
}

// RemovePeer 移除指定的 Peer
func (r *Redis) RemovePeer(ctx context.Context, infoHash, peer string) error {
	key := fmt.Sprintf("tracker:peers:%s", infoHash)
	return r.Client.SRem(ctx, key, peer).Err()
}

// GetPeerCount 获取 Peer 数量
func (r *Redis) GetPeerCount(ctx context.Context, infoHash string) (int64, error) {
	key := fmt.Sprintf("tracker:peers:%s", infoHash)
	return r.Client.SCard(ctx, key).Result()
}

// UpdateStats 更新统计信息
func (r *Redis) UpdateStats(ctx context.Context, infoHash string, seeders, leechers, completed int64) error {
	key := fmt.Sprintf("tracker:stats:%s", infoHash)

	pipe := r.Client.Pipeline()
	pipe.HSet(ctx, key, "seeders", seeders)
	pipe.HSet(ctx, key, "leechers", leechers)
	pipe.HSet(ctx, key, "completed", completed)
	pipe.Expire(ctx, key, 1*time.Hour) // 统计信息保留 1 小时

	_, err := pipe.Exec(ctx)
	return err
}

// GetStats 获取统计信息
func (r *Redis) GetStats(ctx context.Context, infoHash string) (map[string]string, error) {
	key := fmt.Sprintf("tracker:stats:%s", infoHash)
	return r.Client.HGetAll(ctx, key).Result()
}

// IncrementCompleted 增加完成下载的计数
func (r *Redis) IncrementCompleted(ctx context.Context, infoHash string) error {
	key := fmt.Sprintf("tracker:stats:%s", infoHash)
	return r.Client.HIncrBy(ctx, key, "completed", 1).Err()
}
