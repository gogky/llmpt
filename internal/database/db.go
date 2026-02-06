package database

import (
	"context"
	"fmt"
	"llmpt/internal/config"
)

// DB 数据库管理器
type DB struct {
	MongoDB *MongoDB
	Redis   *Redis
}

// New 创建新的数据库连接管理器
func New(cfg *config.Config) (*DB, error) {
	// 连接 MongoDB
	poolOpts := &MongoPoolOptions{
		MaxPoolSize:     cfg.MongoDB.MaxPoolSize,
		MinPoolSize:     cfg.MongoDB.MinPoolSize,
		MaxConnIdleTime: cfg.MongoDB.MaxConnIdleTime,
	}
	mongodb, err := NewMongoDB(cfg.GetMongoURI(), cfg.MongoDB.Database, poolOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MongoDB: %w", err)
	}

	// 连接 Redis
	redisPoolOpts := &RedisPoolOptions{
		PoolSize:     int(cfg.Redis.PoolSize),
		MinIdleConns: int(cfg.Redis.MinIdleConns),
	}
	redisClient, err := NewRedis(cfg.GetRedisAddr(), cfg.Redis.Password, cfg.Redis.DB, redisPoolOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis: %w", err)
	}

	db := &DB{
		MongoDB: mongodb,
		Redis:   redisClient,
	}

	// 创建索引
	ctx := context.Background()
	if err := mongodb.CreateIndexes(ctx); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return db, nil
}

// Close 关闭所有数据库连接
func (db *DB) Close() error {
	ctx := context.Background()

	// 关闭 MongoDB
	if err := db.MongoDB.Close(ctx); err != nil {
		return fmt.Errorf("failed to close MongoDB: %w", err)
	}

	// 关闭 Redis
	if err := db.Redis.Close(); err != nil {
		return fmt.Errorf("failed to close Redis: %w", err)
	}

	fmt.Println("✓ All database connections closed")
	return nil
}
