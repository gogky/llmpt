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

	"llmpt/internal/config"
	"llmpt/internal/database"
	"llmpt/internal/tracker"
)

func main() {
	fmt.Println("🚀 Starting Tracker Server...")

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 连接数据库
	ctx := context.Background()
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Database connected")

	// 创建 Tracker 处理器
	handler := tracker.NewHandler(db)

	// 启动后台清理任务
	go handler.StartCleanup(ctx, 5*time.Minute)

	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/announce", handler.Announce)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 创建 HTTP 服务器
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器（非阻塞）
	go func() {
		fmt.Printf("🎯 Tracker Server listening on %s\n", addr)
		fmt.Println("📡 Announce endpoint: http://localhost" + addr + "/announce")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
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

	fmt.Println("✅ Server stopped gracefully")
}

// loggingMiddleware 记录所有请求
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 记录请求
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

		// 调用下一个处理器
		next.ServeHTTP(w, r)

		// 记录耗时
		duration := time.Since(start)
		log.Printf("Request completed in %v", duration)
	})
}
