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

// AddPeer 添加 Peer 到指定 info_hash 的有序集合中，同时维护活跃种子列表
func (r *Redis) AddPeer(ctx context.Context, infoHash, peer string, isSeeder bool, ttl time.Duration) error {
	seederKey := fmt.Sprintf("tracker:seeders:%s", infoHash)
	leecherKey := fmt.Sprintf("tracker:leechers:%s", infoHash)
	activeKey := "tracker:active_torrents"

	now := float64(time.Now().Unix())
	pipe := r.Client.Pipeline()

	if isSeeder {
		// 添加到做种者集合
		pipe.ZAdd(ctx, seederKey, redis.Z{Score: now, Member: peer})
		// 从下载者集合中移除（防止状态反转时两头占坑）
		pipe.ZRem(ctx, leecherKey, peer)
		pipe.Expire(ctx, seederKey, ttl)
	} else {
		// 添加到下载者集合
		pipe.ZAdd(ctx, leecherKey, redis.Z{Score: now, Member: peer})
		// 从做种者集合中移除
		pipe.ZRem(ctx, seederKey, peer)
		pipe.Expire(ctx, leecherKey, ttl)
	}

	// 记录活跃的种子，方便后续清理
	pipe.SAdd(ctx, activeKey, infoHash)

	_, err := pipe.Exec(ctx)
	return err
}

// GetPeersForRequest 智能获取 Peer 列表，按比例混合做种者和下载者
// 策略：做种者（Seeder）只拿下载者（Leecher）；下载者则混合拿 30% Seeders + 70% Leechers
func (r *Redis) GetPeersForRequest(ctx context.Context, infoHash string, maxPeers int64, isSeeder bool) ([]string, error) {
	seederKey := fmt.Sprintf("tracker:seeders:%s", infoHash)
	leecherKey := fmt.Sprintf("tracker:leechers:%s", infoHash)

	var peers []string

	if isSeeder {
		// 如果是做种者，全部返回 Leecher
		leechers, err := r.Client.Do(ctx, "ZRANDMEMBER", leecherKey, int(maxPeers)).StringSlice()
		if err != nil && err != redis.Nil {
			return nil, err
		}
		peers = leechers
	} else {
		// 如果是下载者，按 3:7 比例请求
		seederQuota := int(float64(maxPeers) * 0.3)
		if seederQuota < 1 && maxPeers > 0 {
			seederQuota = 1
		}
		leecherQuota := int(maxPeers) - seederQuota

		// 获取 Seeders
		seeders, err := r.Client.Do(ctx, "ZRANDMEMBER", seederKey, seederQuota).StringSlice()
		if err != nil && err != redis.Nil {
			return nil, err
		}

		// 获取 Leechers
		leechers, err := r.Client.Do(ctx, "ZRANDMEMBER", leecherKey, leecherQuota).StringSlice()
		if err != nil && err != redis.Nil {
			return nil, err
		}

		// 如果 Seeder 不够 30%，用 Leecher 补足
		if len(seeders) < seederQuota {
			shortfall := seederQuota - len(seeders)
			extraLeechers, _ := r.Client.Do(ctx, "ZRANDMEMBER", leecherKey, leecherQuota+shortfall).StringSlice()
			leechers = extraLeechers
		} else if len(leechers) < leecherQuota {
			// 如果 Leecher 不够 70%，用 Seeder 补足
			shortfall := leecherQuota - len(leechers)
			extraSeeders, _ := r.Client.Do(ctx, "ZRANDMEMBER", seederKey, seederQuota+shortfall).StringSlice()
			seeders = extraSeeders
		}

		peers = append(peers, seeders...)
		peers = append(peers, leechers...)

		// 确保不超标并去重（理论上 ZSet 不同角色不会重复，但如果补足逻辑导致重合，可以用 map 去重，这里为了性能直接返回）
		if len(peers) > int(maxPeers) {
			peers = peers[:maxPeers]
		}
	}

	return peers, nil
}

// RemovePeer 从种子中移除指定的 Peer（同时从两边移除以防万一）
func (r *Redis) RemovePeer(ctx context.Context, infoHash, peer string) error {
	seederKey := fmt.Sprintf("tracker:seeders:%s", infoHash)
	leecherKey := fmt.Sprintf("tracker:leechers:%s", infoHash)

	pipe := r.Client.Pipeline()
	pipe.ZRem(ctx, seederKey, peer)
	pipe.ZRem(ctx, leecherKey, peer)
	_, err := pipe.Exec(ctx)
	return err
}

// GetPeerCount 获取精准的做种者和下载者数量
func (r *Redis) GetPeerCount(ctx context.Context, infoHash string) (seeders, leechers int64, err error) {
	seederKey := fmt.Sprintf("tracker:seeders:%s", infoHash)
	leecherKey := fmt.Sprintf("tracker:leechers:%s", infoHash)

	// 使用 Pipeline 提高效率
	pipe := r.Client.Pipeline()
	sCmd := pipe.ZCard(ctx, seederKey)
	lCmd := pipe.ZCard(ctx, leecherKey)

	_, err = pipe.Exec(ctx)
	if err != nil {
		return 0, 0, err
	}

	return sCmd.Val(), lCmd.Val(), nil
}

// CleanExpiredPeers 清理全局所有的超时节点
func (r *Redis) CleanExpiredPeers(ctx context.Context, timeout time.Duration) error {
	activeKey := "tracker:active_torrents"

	// 获取所有活跃的 info_hash
	infoHashes, err := r.Client.SMembers(ctx, activeKey).Result()
	if err != nil {
		return err
	}

	// 死亡判定线：早于这个时间戳更新的心跳统统算死节点
	deathLine := float64(time.Now().Unix() - int64(timeout.Seconds()))
	deathLineStr := fmt.Sprintf("%f", deathLine)

	for _, infoHash := range infoHashes {
		seederKey := fmt.Sprintf("tracker:seeders:%s", infoHash)
		leecherKey := fmt.Sprintf("tracker:leechers:%s", infoHash)

		pipe := r.Client.Pipeline()
		// 1. 抹权超时节点 (-inf 到 deathLine)
		pipe.ZRemRangeByScore(ctx, seederKey, "-inf", deathLineStr)
		pipe.ZRemRangeByScore(ctx, leecherKey, "-inf", deathLineStr)

		// 2. 查询余量
		sCmd := pipe.ZCard(ctx, seederKey)
		lCmd := pipe.ZCard(ctx, leecherKey)

		_, _ = pipe.Exec(ctx)

		// 3. 如果变成空城，果断将其从活跃列表中除名
		if sCmd.Val() == 0 && lCmd.Val() == 0 {
			r.Client.SRem(ctx, activeKey, infoHash)
		}
	}
	return nil
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

// CheckRateLimit 检查指定 IP 的请求频率是否超过限制
// 返回 true 表示允许请求，返回 false 表示限流
func (r *Redis) CheckRateLimit(ctx context.Context, ip string, window time.Duration, limit int) (bool, error) {
	key := fmt.Sprintf("tracker:ratelimit:%s", ip)

	// 原子递增
	count, err := r.Client.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}

	// 如果是第一次请求，设置过期时间
	if count == 1 {
		r.Client.Expire(ctx, key, window)
	}

	// 判断是否超过限制
	if count > int64(limit) {
		return false, nil
	}

	return true, nil
}
