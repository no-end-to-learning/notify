package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"notify/internal/queue"
	"notify/internal/service"
)

func HandleGrafanaWebhook(w http.ResponseWriter, r *http.Request) {
	channelStr := r.URL.Query().Get("channel")
	target := r.URL.Query().Get("target")

	// Compatibility: use to if target is empty
	if target == "" {
		target = r.URL.Query().Get("to")
	}

	if channelStr == "" || target == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel and target (or to) query params are required")
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
		template = string(service.ColorOrange)
		title = alert.RuleName
	case "ok":
		template = string(service.ColorGreen)
		title = "✅ " + alert.RuleName
	default:
		template = string(service.ColorGrey)
		title = alert.RuleName
	}

	if len(alert.EvalMatches) > 0 {
		var items []string
		for _, item := range alert.EvalMatches {
			val := strconv.FormatFloat(item.Value, 'f', -1, 64)
			items = append(items, fmt.Sprintf("%s: %s", item.Metric, val))
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
			val := strconv.FormatFloat(item.Value, 'f', -1, 64)
			items = append(items, fmt.Sprintf("%s: %s", service.EscapeHTML(item.Metric), val))
		}
		parts = append(parts, strings.Join(items, "\n"))
	}

	// Note: Message (Italic)
	if alert.Message != "" {
		parts = append(parts, "<i>"+service.EscapeHTML(alert.Message)+"</i>")
	}

	return map[string]any{
		"text":       strings.Join(parts, "\n\n"),
		"parse_mode": "HTML",
	}
}
