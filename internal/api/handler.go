package api

import (
	"encoding/json"
	"log"
	"net/http"

	"llmpt/internal/config"
	"llmpt/internal/database"
)

// Handler Web API 处理器
type Handler struct {
	db  *database.DB
	cfg *config.Config
}

// NewHandler 创建 Web API 处理器
func NewHandler(db *database.DB, cfg *config.Config) *Handler {
	return &Handler{
		db:  db,
		cfg: cfg,
	}
}

// corsMiddleware 简单的 CORS 中间件，允许前端跨域
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// 放行 OPTIONS 预检请求
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// RegisterRoutes 注册所有的 API 路由
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/torrents", corsMiddleware(h.ListTorrents))
	mux.HandleFunc("POST /api/v1/publish", corsMiddleware(h.PublishTorrent))
}

// JSONRes 返回 JSON 响应
func JSONRes(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode json response: %v", err)
	}
}

// ErrorRes 返回错误 JSON 响应
func ErrorRes(w http.ResponseWriter, status int, message string) {
	JSONRes(w, status, map[string]string{"error": message})
}
