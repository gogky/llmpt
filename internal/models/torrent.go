package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Torrent MongoDB 中的 Torrent 模型
type Torrent struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`                 // 模型名称
	InfoHash    string             `bson:"info_hash" json:"info_hash"`       // 种子唯一指纹
	TotalSize   int64              `bson:"total_size" json:"total_size"`     // 总大小（字节）
	FileCount   int                `bson:"file_count" json:"file_count"`     // 文件数量
	MagnetLink  string             `bson:"magnet_link" json:"magnet_link"`   // 磁力链接
	PieceLength int64              `bson:"piece_length" json:"piece_length"` // 分片大小
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`     // 创建时间
}

// TorrentStats Tracker 统计信息（从 Redis 获取）
type TorrentStats struct {
	Seeders   int64 `json:"seeders"`   // 做种人数
	Leechers  int64 `json:"leechers"`  // 下载人数
	Completed int64 `json:"completed"` // 完成下载次数
}

// TorrentWithStats 带统计信息的 Torrent
type TorrentWithStats struct {
	Torrent
	Stats TorrentStats `json:"stats"`
}

// PeerInfo Peer 信息
type PeerInfo struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	PeerID     string `json:"peer_id,omitempty"`
	Uploaded   int64  `json:"uploaded,omitempty"`
	Downloaded int64  `json:"downloaded,omitempty"`
	Left       int64  `json:"left,omitempty"`
}

// AnnounceRequest Tracker announce 请求参数
type AnnounceRequest struct {
	InfoHash   string `json:"info_hash"`  // 种子 hash
	PeerID     string `json:"peer_id"`    // 客户端 ID
	Port       int    `json:"port"`       // 监听端口
	Uploaded   int64  `json:"uploaded"`   // 已上传字节数
	Downloaded int64  `json:"downloaded"` // 已下载字节数
	Left       int64  `json:"left"`       // 剩余字节数
	Event      string `json:"event"`      // 事件: started, completed, stopped
	Compact    int    `json:"compact"`    // 是否使用紧凑模式
	NumWant    int    `json:"numwant"`    // 期望返回的 peer 数量
}

// AnnounceResponse Tracker announce 响应
type AnnounceResponse struct {
	Interval    int64      `json:"interval"`     // 心跳间隔（秒）
	MinInterval int64      `json:"min_interval"` // 最小心跳间隔（秒）
	Complete    int64      `json:"complete"`     // Seeders 数量
	Incomplete  int64      `json:"incomplete"`   // Leechers 数量
	Peers       []PeerInfo `json:"peers"`        // Peer 列表
}
