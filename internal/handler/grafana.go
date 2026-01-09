package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"notify/internal/service"
)

func GrafanaWebhook(w http.ResponseWriter, r *http.Request) {
	channelStr := r.URL.Query().Get("channel")
	to := r.URL.Query().Get("to")

	if channelStr == "" || to == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "channel and to query params are required")
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

	slog.Info("Grafana alert received", "alert", alert)

	svc, err := service.GetService(channel)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	message := buildGrafanaMessage(channel, alert)
	result, err := svc.SendRawMessage(to, message)
	if err != nil {
		writeError(w, http.StatusBadGateway, "SERVICE_ERROR", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func buildGrafanaMessage(channel service.Channel, alert service.GrafanaAlert) any {
	switch channel {
	case service.ChannelWecom:
		return buildWecomGrafanaMessage(alert)
	case service.ChannelTelegram:
		return buildTelegramGrafanaMessage(alert)
	default:
		return buildLarkGrafanaMessage(alert)
	}
}

func buildWecomGrafanaMessage(alert service.GrafanaAlert) map[string]any {
	stateEmoji := map[string]string{
		"alerting": "⚠️",
		"ok":       "✅",
	}
	emoji := stateEmoji[alert.State]
	if emoji == "" {
		emoji = "📢"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("### %s %s", emoji, alert.RuleName))

	if len(alert.EvalMatches) > 0 {
		parts = append(parts, `<font color="comment">──────────</font>`)
		var items []string
		for _, item := range alert.EvalMatches {
			items = append(items, fmt.Sprintf("%s: %v", item.Metric, item.Value))
		}
		parts = append(parts, strings.Join(items, "\n"))
	}

	if alert.Message != "" {
		parts = append(parts, `<font color="comment">──────────</font>`)
		lines := strings.Split(alert.Message, "\n")
		var coloredLines []string
		for _, line := range lines {
			trimmed := strings.TrimPrefix(line, "- ")
			coloredLines = append(coloredLines, fmt.Sprintf(`<font color="comment">%s</font>`, trimmed))
		}
		parts = append(parts, strings.Join(coloredLines, "\n"))
	}

	if len(parts) == 1 {
		parts = append(parts, fmt.Sprintf("> %s", time.Now().String()))
	}

	return map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"content": strings.Join(parts, "\n"),
		},
	}
}

func buildLarkGrafanaMessage(alert service.GrafanaAlert) map[string]any {
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

func buildTelegramGrafanaMessage(alert service.GrafanaAlert) map[string]any {
	stateEmoji := map[string]string{
		"alerting": "⚠️",
		"ok":       "✅",
	}
	emoji := stateEmoji[alert.State]
	if emoji == "" {
		emoji = "📢"
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("<b>%s %s</b>", emoji, escapeHTML(alert.RuleName)))

	if len(alert.EvalMatches) > 0 {
		var items []string
		for _, item := range alert.EvalMatches {
			items = append(items, fmt.Sprintf("%s: %v", escapeHTML(item.Metric), item.Value))
		}
		parts = append(parts, strings.Join(items, "\n"))
	}

	if alert.Message != "" {
		parts = append(parts, fmt.Sprintf("<i>%s</i>", escapeHTML(alert.Message)))
	}

	if len(parts) == 1 {
		parts = append(parts, time.Now().String())
	}

	return map[string]any{
		"text":       strings.Join(parts, "\n\n"),
		"parse_mode": "HTML",
	}
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}
