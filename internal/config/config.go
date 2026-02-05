package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config 应用配置
type Config struct {
	MongoDB MongoDBConfig
	Redis   RedisConfig
	Server  ServerConfig
}

// MongoDBConfig MongoDB 配置
type MongoDBConfig struct {
	URI              string
	Database         string
	Username         string
	Password         string
	MaxPoolSize      uint64        // 连接池最大连接数，默认 50
	MinPoolSize      uint64        // 连接池最小连接数，默认 10
	MaxConnIdleTime  time.Duration // 连接最大空闲时间，默认 30s
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host         string
	Port         string
	Password     string
	DB           int
	PoolSize     uint64 // 连接池最大连接数，默认 50
	MinIdleConns uint64 // 连接池最小空闲连接数，默认 10
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port        string
	TrackerURL  string
	Environment string
}

// Load 加载配置（从环境变量）
func Load() (*Config, error) {
	config := &Config{
		MongoDB: MongoDBConfig{
			URI:             getEnv("MONGODB_URI", "mongodb://admin:admin123@localhost:27017"),
			Database:        getEnv("MONGODB_DATABASE", "hf_p2p_v1"),
			Username:        getEnv("MONGODB_USERNAME", "admin"),
			Password:        getEnv("MONGODB_PASSWORD", "admin123"),
			MaxPoolSize:     getEnvUint64("MONGODB_MAX_POOL_SIZE", 50),
			MinPoolSize:     getEnvUint64("MONGODB_MIN_POOL_SIZE", 10),
			MaxConnIdleTime: getEnvDuration("MONGODB_MAX_CONN_IDLE_TIME", 30*time.Second),
		},
		Redis: RedisConfig{
			Host:         getEnv("REDIS_HOST", "localhost"),
			Port:         getEnv("REDIS_PORT", "6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           0,
			PoolSize:     getEnvUint64("REDIS_POOL_SIZE", 50),
			MinIdleConns: getEnvUint64("REDIS_MIN_IDLE_CONNS", 10),
		},
		Server: ServerConfig{
			Port:        getEnv("SERVER_PORT", "8080"),
			TrackerURL:  getEnv("TRACKER_URL", "http://localhost:8080/announce"),
			Environment: getEnv("ENVIRONMENT", "development"),
		},
	}

	return config, nil
}

// GetMongoURI 获取 MongoDB 连接字符串
func (c *Config) GetMongoURI() string {
	if c.MongoDB.URI != "" {
		return c.MongoDB.URI
	}
	return fmt.Sprintf("mongodb://%s:%s@localhost:27017",
		c.MongoDB.Username, c.MongoDB.Password)
}

// GetRedisAddr 获取 Redis 地址
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Redis.Host, c.Redis.Port)
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getEnvUint64 获取环境变量并解析为 uint64
func getEnvUint64(key string, defaultValue uint64) uint64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	v, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return defaultValue
	}
	return v
}

// getEnvDuration 获取环境变量并解析为 time.Duration（如 "30s", "1m"）
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return defaultValue
	}
	return d
}
