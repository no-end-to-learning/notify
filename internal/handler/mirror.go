package handler

import (
	"log/slog"

	"notify/internal/queue"
	"notify/internal/service"
)

var mirrorChat string

func InitMirror(chatID string) {
	mirrorChat = chatID
	if chatID != "" {
		slog.Info("Mirror enabled", "telegramChat", chatID)
	}
}

func mirrorToTelegram(params service.MessageParams) {
	if mirrorChat == "" {
		return
	}

	svc, err := service.GetService(service.ChannelTelegram)
	if err != nil {
		slog.Warn("Mirror: Telegram service not available", "error", err)
		return
	}

	// Build message and enqueue through queue for rate limiting
	message := svc.BuildMessage(params)
	queue.GetManager().Enqueue(service.ChannelTelegram, mirrorChat, message)
}

func mirrorGrafanaAlertToTelegram(alert service.GrafanaAlert) {
	if mirrorChat == "" {
		return
	}

	// Build Telegram format message and enqueue
	message := formatGrafanaAlertForTelegram(alert)

	queue.GetManager().Enqueue(service.ChannelTelegram, mirrorChat, message)
}
