package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"notify/internal/queue"
	"notify/internal/service"
)

func HandleGrafanaWebhook(w http.ResponseWriter, r *http.Request) {
	channelStr := r.URL.Query().Get("channel")
	target := r.URL.Query().Get("target")

	if channelStr == "" || target == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel and target query params are required")
		return
	}

	channel, err := service.ValidateChannel(channelStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	var alert service.GrafanaAlert
	if err := json.NewDecoder(r.Body).Decode(&alert); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	slog.Info("Grafana alert received", slog.Any("alert", alert))

	_, err = service.GetService(channel)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	message := formatGrafanaAlert(channel, alert)
	taskID := queue.GetManager().Enqueue(channel, target, message)

	// Mirror Lark Grafana alerts to Telegram
	if channel == service.ChannelLark {
		mirrorGrafanaAlertToTelegram(alert)
	}

	writeJSON(w, http.StatusOK, &service.SendResult{
		TaskID:  taskID,
		Success: true,
	})
}

func formatGrafanaAlert(channel service.Channel, alert service.GrafanaAlert) any {
	switch channel {
	case service.ChannelTelegram:
		return formatGrafanaAlertForTelegram(alert)
	default:
		return formatGrafanaAlertForLark(alert)
	}
}

func formatGrafanaAlertForLark(alert service.GrafanaAlert) map[string]any {
	var elements []any
	var template, title string

	switch alert.State {
	case "alerting":
		template = "Orange"
		title = alert.RuleName
	case "ok":
		template = "Green"
		title = "✅ " + alert.RuleName
	default:
		template = "Grey"
		title = alert.RuleName
	}

	if len(alert.EvalMatches) > 0 {
		var items []string
		for _, item := range alert.EvalMatches {
			items = append(items, fmt.Sprintf("%s: %v", item.Metric, item.Value))
		}
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": strings.Join(items, "\n"),
		})
	}

	if alert.Message != "" {
		if len(elements) > 0 {
			elements = append(elements, map[string]any{"tag": "hr"})
		}
		elements = append(elements, map[string]any{
			"tag":      "note",
			"elements": []any{map[string]any{"tag": "plain_text", "content": alert.Message}},
		})
	}

	if len(elements) == 0 {
		elements = append(elements, map[string]any{
			"tag":      "note",
			"elements": []any{map[string]any{"tag": "plain_text", "content": time.Now().String()}},
		})
	}

	return map[string]any{
		"config": map[string]any{"wide_screen_mode": true},
		"header": map[string]any{
			"title":    map[string]any{"tag": "plain_text", "content": title},
			"template": template,
		},
		"elements": elements,
	}
}

func formatGrafanaAlertForTelegram(alert service.GrafanaAlert) map[string]any {
	stateEmoji := map[string]string{
		"alerting": "⚠️",
		"ok":       "✅",
	}
	emoji := stateEmoji[alert.State]
	if emoji == "" {
		emoji = "📢"
	}

	var parts []string
	// Title: Bold
	parts = append(parts, fmt.Sprintf("<b>%s %s</b>", emoji, service.EscapeHTML(alert.RuleName)))

	// Content: EvalMatches
	if len(alert.EvalMatches) > 0 {
		var items []string
		for _, item := range alert.EvalMatches {
			items = append(items, fmt.Sprintf("%s: %v", service.EscapeHTML(item.Metric), item.Value))
		}
		parts = append(parts, strings.Join(items, "\n"))
	}

	// Note: Message (Italic)
	if alert.Message != "" {
		parts = append(parts, "<i>"+service.EscapeHTML(alert.Message)+"</i>")
	}

	if len(parts) == 1 {
		parts = append(parts, service.EscapeHTML(time.Now().String()))
	}

	return map[string]any{
		"text":       strings.Join(parts, "\n\n"),
		"parse_mode": "HTML",
	}
}
