package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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
	level := logLevelFromEnv()
	logger := slog.New(tint.NewHandler(os.Stdout, &tint.Options{
		Level:      level,
		TimeFormat: time.TimeOnly,
	}))
	slog.SetDefault(logger)

	// Load and validate configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("Configuration error", "error", err)
		os.Exit(1)
	}

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

		done := make(chan struct{})
		go func() {
			queue.GetManager().Shutdown()
			close(done)
		}()

		select {
		case <-done:
			slog.Info("Graceful shutdown complete")
		case <-time.After(30 * time.Second):
			slog.Warn("Shutdown timed out, forcing exit")
		}
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

func logLevelFromEnv() slog.Level {
	switch strings.ToLower(os.Getenv("APP_LOG_LEVEL")) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
