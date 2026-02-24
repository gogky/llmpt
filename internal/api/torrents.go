package api

import (
	"log"
	"net/http"
	"strconv"

	"llmpt/internal/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ListTorrents 获取模型列表 (GET /api/v1/torrents)
func (h *Handler) ListTorrents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	collection := h.db.MongoDB.TorrentsCollection()

	filter := bson.M{}
	repoID := r.URL.Query().Get("repo_id")
	if repoID != "" {
		filter["repo_id"] = repoID
	}

	// 1. 从 MongoDB 提取列表
	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf("Failed to fetch torrents from db: %v", err)
		ErrorRes(w, http.StatusInternalServerError, "Failed to fetch torrents")
		return
	}
	defer cursor.Close(ctx)

	var torrents []models.Torrent
	if err = cursor.All(ctx, &torrents); err != nil {
		log.Printf("Failed to decode torrents: %v", err)
		ErrorRes(w, http.StatusInternalServerError, "Failed to decode torrents")
		return
	}

	// 2. 对于每个 Torrent，从 Redis 拿最新统计数据
	var results []models.TorrentWithStats
	for _, t := range torrents {
		// 这里通过 GetPeerCount 获取最新当前值
		seeders, leechers, err := h.db.Redis.GetPeerCount(ctx, t.InfoHash)
		if err != nil {
			log.Printf("Failed to get peer count for %s: %v", t.InfoHash, err)
			continue
		}

		// 获取总下载完成数
		statsStrMap, _ := h.db.Redis.GetStats(ctx, t.InfoHash)
		var completed int64
		if val, ok := statsStrMap["completed"]; ok && val != "" {
			parsed, err := strconv.ParseInt(val, 10, 64)
			if err == nil {
				completed = parsed
			}
		}

		stats := models.TorrentStats{
			Seeders:   seeders,
			Leechers:  leechers,
			Completed: completed,
		}

		results = append(results, models.TorrentWithStats{
			Torrent: t,
			Stats:   stats,
		})
	}

	// 如果为空，返回空列表而非 nil
	if results == nil {
		results = []models.TorrentWithStats{}
	}

	JSONRes(w, http.StatusOK, map[string]interface{}{
		"total": len(results),
		"data":  results,
	})
}
