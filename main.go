package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"notify/internal/config"
	"notify/internal/handler"
	"notify/internal/service"
)

func main() {
	// Setup structured logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg := config.Load()

	// Initialize services
	service.Init(cfg)

	// Setup routes
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("POST /api/messages", handler.SendMessage)
	mux.HandleFunc("POST /api/messages/raw", handler.SendRawMessage)
	mux.HandleFunc("GET /api/chats", handler.ListChats)
	mux.HandleFunc("POST /api/webhooks/grafana", handler.HandleGrafanaWebhook)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("Server listening", "url", cfg.Server.BaseURL)

	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
