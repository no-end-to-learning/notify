package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lmittmann/tint"

	"notify/internal/config"
	"notify/internal/handler"
	"notify/internal/queue"
	"notify/internal/service"
)

func main() {
	// Setup structured logging with color support
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      slog.LevelInfo,
		TimeFormat: time.TimeOnly,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.Load()

	// Initialize services
	service.Init(cfg)

	// Initialize queue
	queue.Init(cfg.Queue)

	// Setup routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("POST /api/messages", handler.SendMessage)
	mux.HandleFunc("POST /api/messages/raw", handler.SendRawMessage)
	mux.HandleFunc("GET /api/chats", handler.ListChats)
	mux.HandleFunc("POST /api/webhooks/grafana", handler.HandleGrafanaWebhook)

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		slog.Info("Shutting down server...")
		queue.GetManager().Shutdown()
		os.Exit(0)
	}()

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("Server listening", "url", cfg.Server.BaseURL)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
