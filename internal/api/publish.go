package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// PublishRequest 发布模型的请求结构
type PublishRequest struct {
	Name        string `json:"name"`
	InfoHash    string `json:"info_hash"`
	TotalSize   int64  `json:"total_size"`
	FileCount   int    `json:"file_count"`
	MagnetLink  string `json:"magnet_link"`
	PieceLength int64  `json:"piece_length"`
}

// PublishTorrent 接收并发布新的模型元数据 (POST /api/v1/publish)
func (h *Handler) PublishTorrent(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		ErrorRes(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ErrorRes(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// 基础校验
	if req.InfoHash == "" || req.Name == "" {
		ErrorRes(w, http.StatusBadRequest, "info_hash and name are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	collection := h.db.MongoDB.TorrentsCollection()

	// 使用 Upsert 逻辑（存在则更新，不存在则插入）
	filter := bson.M{"info_hash": req.InfoHash}
	update := bson.M{
		"$set": bson.M{
			"name":         req.Name,
			"total_size":   req.TotalSize,
			"file_count":   req.FileCount,
			"magnet_link":  req.MagnetLink,
			"piece_length": req.PieceLength,
		},
		"$setOnInsert": bson.M{
			"_id":        primitive.NewObjectID(),
			"created_at": time.Now(),
		},
	}

	opts := options.Update().SetUpsert(true)

	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		log.Printf("Failed to upsert torrent: %v", err)
		ErrorRes(w, http.StatusInternalServerError, "failed to save torrent metadata")
		return
	}

	msg := "torrent metadata updated"
	if result.UpsertedCount > 0 {
		msg = "torrent metadata created"
	}

	JSONRes(w, http.StatusOK, map[string]string{
		"message":   msg,
		"info_hash": req.InfoHash,
	})
}
