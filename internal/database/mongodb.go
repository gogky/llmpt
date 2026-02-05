package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoDB MongoDB 客户端包装
type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

// MongoPoolOptions MongoDB 连接池配置
type MongoPoolOptions struct {
	MaxPoolSize     uint64
	MinPoolSize     uint64
	MaxConnIdleTime time.Duration
}

// DefaultMongoPoolOptions 返回默认连接池配置
func DefaultMongoPoolOptions() MongoPoolOptions {
	return MongoPoolOptions{
		MaxPoolSize:     50,
		MinPoolSize:     10,
		MaxConnIdleTime: 30 * time.Second,
	}
}

// NewMongoDB 创建新的 MongoDB 连接
// poolOpts 为 nil 时使用默认连接池配置
func NewMongoDB(uri, database string, poolOpts *MongoPoolOptions) (*MongoDB, error) {
	opts := DefaultMongoPoolOptions()
	if poolOpts != nil {
		if poolOpts.MaxPoolSize > 0 {
			opts.MaxPoolSize = poolOpts.MaxPoolSize
		}
		if poolOpts.MinPoolSize > 0 {
			opts.MinPoolSize = poolOpts.MinPoolSize
		}
		if poolOpts.MaxConnIdleTime > 0 {
			opts.MaxConnIdleTime = poolOpts.MaxConnIdleTime
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 设置客户端选项
	clientOptions := options.Client().
		ApplyURI(uri).
		SetMaxPoolSize(opts.MaxPoolSize).
		SetMinPoolSize(opts.MinPoolSize).
		SetMaxConnIdleTime(opts.MaxConnIdleTime)

	// 连接到 MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// 检查连接
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(database)

	fmt.Printf("✓ Successfully connected to MongoDB (database: %s)\n", database)

	return &MongoDB{
		Client:   client,
		Database: db,
	}, nil
}

// Close 关闭 MongoDB 连接
func (m *MongoDB) Close(ctx context.Context) error {
	if m.Client != nil {
		return m.Client.Disconnect(ctx)
	}
	return nil
}

// GetCollection 获取集合
func (m *MongoDB) GetCollection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}

// TorrentsCollection 获取 torrents 集合
func (m *MongoDB) TorrentsCollection() *mongo.Collection {
	return m.GetCollection("torrents")
}

// CreateIndexes 创建索引
func (m *MongoDB) CreateIndexes(ctx context.Context) error {
	torrents := m.TorrentsCollection()

	// 创建 info_hash 唯一索引
	infoHashIndex := mongo.IndexModel{
		Keys:    map[string]interface{}{"info_hash": 1},
		Options: options.Index().SetUnique(true),
	}

	// 创建 created_at 索引（用于排序）
	createdAtIndex := mongo.IndexModel{
		Keys: map[string]interface{}{"created_at": -1},
	}

	// 创建 name 文本索引（用于搜索）
	nameIndex := mongo.IndexModel{
		Keys: map[string]interface{}{"name": "text"},
	}

	indexes := []mongo.IndexModel{infoHashIndex, createdAtIndex, nameIndex}

	_, err := torrents.Indexes().CreateMany(ctx, indexes)
	if err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	fmt.Println("✓ MongoDB indexes created successfully")
	return nil
}
