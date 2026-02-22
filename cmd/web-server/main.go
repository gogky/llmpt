package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"llmpt/internal/api"
	"llmpt/internal/config"
	"llmpt/internal/database"
)

func main() {
	fmt.Println("🚀 Starting Web API Server...")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Database connected for Web API")

	// 创建 API 处理器
	handler := api.NewHandler(db, cfg)

	// 设置路由
	mux := http.NewServeMux()

	// 健康检查
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 注册 API 路由
	handler.RegisterRoutes(mux)

	// API Web Server 默认端口给 8080 (注意要与 Tracker Server 的端口区分开，通常可以用环境变量覆盖)
	port := 8080
	addr := fmt.Sprintf(":%d", port)

	server := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器（非阻塞）
	go func() {
		fmt.Printf("🎯 Web API Server listening on %s\n", addr)
		fmt.Printf("📡 API Endpoint Base: http://localhost%s/api/v1\n", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Web Server failed: %v", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\n🛑 Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	fmt.Println("✅ Web Server stopped gracefully")
}

// loggingMiddleware 记录所有请求
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// 由于 OPTIONS 请求会很多且影响控制台阅读，如果是 OPTIONS 请求可以静默或者降低日志级别
		if r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}

		log.Printf("[API] %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)

		duration := time.Since(start)
		log.Printf("[API] Request completed in %v", duration)
	})
}
