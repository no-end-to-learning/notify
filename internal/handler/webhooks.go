package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"sort"
	"strconv"
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

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Failed to read request body")
		return
	}

	alert, payloadFormat, err := decodeGrafanaAlert(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	slog.Info("Grafana alert received",
		"format", payloadFormat,
		"state", alert.State,
		"ruleName", alert.RuleName,
		"matches", len(alert.EvalMatches),
		"bodyBytes", len(body),
	)

	_, err = service.GetService(channel)
	if err != nil {
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}

	message := formatGrafanaAlert(channel, alert)
	queue.GetManager().Enqueue(channel, target, message)

	writeJSON(w, http.StatusOK, &service.SendResult{
		Success: true,
	})
}

type grafanaUnifiedWebhook struct {
	Receiver          string                `json:"receiver"`
	Status            string                `json:"status"`
	State             string                `json:"state"`
	Title             string                `json:"title"`
	Message           string                `json:"message"`
	Alerts            []grafanaUnifiedAlert `json:"alerts"`
	GroupLabels       map[string]string     `json:"groupLabels"`
	CommonLabels      map[string]string     `json:"commonLabels"`
	CommonAnnotations map[string]string     `json:"commonAnnotations"`
	TruncatedAlerts   int                   `json:"truncatedAlerts"`
}

type grafanaUnifiedAlert struct {
	Status      string              `json:"status"`
	Labels      map[string]string   `json:"labels"`
	Annotations map[string]string   `json:"annotations"`
	Values      map[string]*float64 `json:"values"`
}

func decodeGrafanaAlert(body []byte) (service.GrafanaAlert, string, error) {
	var legacy service.GrafanaAlert
	if err := json.Unmarshal(body, &legacy); err != nil {
		return service.GrafanaAlert{}, "", err
	}

	var unified grafanaUnifiedWebhook
	if err := json.Unmarshal(body, &unified); err != nil {
		return service.GrafanaAlert{}, "", err
	}

	if isUnifiedGrafanaWebhook(unified) {
		return normalizeUnifiedGrafanaAlert(unified), "unified", nil
	}
	if legacy.RuleName == "" && legacy.State == "" && legacy.EvalMatches == nil {
		return service.GrafanaAlert{}, "", fmt.Errorf("unsupported Grafana webhook payload")
	}

	return legacy, "legacy", nil
}

func isUnifiedGrafanaWebhook(webhook grafanaUnifiedWebhook) bool {
	return webhook.Receiver != "" || webhook.Status != "" || len(webhook.Alerts) > 0
}

func normalizeUnifiedGrafanaAlert(webhook grafanaUnifiedWebhook) service.GrafanaAlert {
	state := normalizeGrafanaState(webhook.Status)
	if state == "" {
		state = normalizeGrafanaState(webhook.State)
	}

	ruleName := firstNonEmpty(
		webhook.CommonLabels["alertname"],
		webhook.GroupLabels["alertname"],
		firstAlertLabel(webhook.Alerts, "alertname"),
		webhook.Title,
	)
	message := firstNonEmpty(
		webhook.CommonAnnotations["lark_message"],
		webhook.CommonAnnotations["message"],
		firstAlertAnnotation(webhook.Alerts, "lark_message"),
		firstAlertAnnotation(webhook.Alerts, "message"),
	)

	alert := service.GrafanaAlert{
		State:    state,
		RuleName: ruleName,
		Message:  message,
		SortOrder: strings.ToLower(firstMeaningful(
			webhook.CommonAnnotations["notify_sort_order"],
			firstAlertAnnotation(webhook.Alerts, "notify_sort_order"),
		)),
	}
	alert.SortAbs, _ = strconv.ParseBool(firstMeaningful(
		webhook.CommonAnnotations["notify_sort_abs"],
		firstAlertAnnotation(webhook.Alerts, "notify_sort_abs"),
	))
	if state == "alerting" {
		for _, item := range webhook.Alerts {
			itemState := normalizeGrafanaState(item.Status)
			if itemState != "" && itemState != "alerting" {
				continue
			}

			if errorMessage := firstMeaningful(
				item.Annotations["Error"],
				item.Annotations["error"],
			); errorMessage != "" {
				source := firstMeaningful(item.Labels["rulename"], item.Labels["alertname"])
				if source != "" && source != ruleName {
					errorMessage = source + ": " + errorMessage
				}
				alert.Message = appendMessage(alert.Message, errorMessage)
				continue
			}

			value, ok := unifiedAlertValue(item)
			if !ok {
				continue
			}
			alert.EvalMatches = append(alert.EvalMatches, service.EvalMatch{
				Metric:  unifiedAlertMetric(item),
				Value:   value,
				SortKey: firstMeaningful(item.Annotations["notify_sort_key"]),
			})
		}
		sortGrafanaEvalMatches(&alert)
	}

	if webhook.TruncatedAlerts > 0 {
		truncated := fmt.Sprintf("Grafana omitted %d alerts from this notification", webhook.TruncatedAlerts)
		if alert.Message == "" {
			alert.Message = truncated
		} else {
			alert.Message += "\n" + truncated
		}
	}

	return alert
}

func sortGrafanaEvalMatches(alert *service.GrafanaAlert) {
	if len(alert.EvalMatches) < 2 || alert.SortOrder == "" {
		return
	}

	descending := alert.SortOrder == "desc"
	sort.SliceStable(alert.EvalMatches, func(i, j int) bool {
		left := alert.EvalMatches[i]
		right := alert.EvalMatches[j]
		if left.SortKey == "" || right.SortKey == "" {
			if left.SortKey == right.SortKey {
				return left.Metric < right.Metric
			}
			return left.SortKey != ""
		}

		comparison := compareGrafanaSortKeys(left.SortKey, right.SortKey, alert.SortAbs)
		if comparison == 0 {
			return left.Metric < right.Metric
		}
		if descending {
			return comparison > 0
		}
		return comparison < 0
	})
}

func compareGrafanaSortKeys(left, right string, absolute bool) int {
	leftNumber, leftErr := strconv.ParseFloat(left, 64)
	rightNumber, rightErr := strconv.ParseFloat(right, 64)
	if leftErr == nil && rightErr == nil {
		if absolute {
			leftNumber = math.Abs(leftNumber)
			rightNumber = math.Abs(rightNumber)
		}
		switch {
		case leftNumber < rightNumber:
			return -1
		case leftNumber > rightNumber:
			return 1
		default:
			return 0
		}
	}

	return strings.Compare(strings.ToLower(left), strings.ToLower(right))
}

func normalizeGrafanaState(state string) string {
	switch strings.ToLower(state) {
	case "firing", "alerting":
		return "alerting"
	case "resolved", "ok":
		return "ok"
	default:
		return strings.ToLower(state)
	}
}

func unifiedAlertMetric(alert grafanaUnifiedAlert) string {
	if metric := firstMeaningful(
		alert.Annotations["lark_metric"],
		alert.Annotations["metric"],
		alert.Labels["metric"],
		alert.Labels["rulename"],
	); metric != "" {
		return metric
	}

	var labels []string
	for key, value := range alert.Labels {
		if value == "" || isInternalGrafanaLabel(key) {
			continue
		}
		labels = append(labels, fmt.Sprintf("%s=%s", key, value))
	}
	sort.Strings(labels)
	if len(labels) > 0 {
		return strings.Join(labels, ", ")
	}

	return firstMeaningful(alert.Labels["alertname"], "alert")
}

func unifiedAlertValue(alert grafanaUnifiedAlert) (float64, bool) {
	if value := firstMeaningful(alert.Annotations["lark_value"], alert.Annotations["value"]); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed, true
		}
	}

	if value, ok := alert.Values["A"]; ok && value != nil {
		return *value, true
	}
	keys := make([]string, 0, len(alert.Values))
	for key := range alert.Values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if alert.Values[key] != nil {
			return *alert.Values[key], true
		}
	}

	return 0, false
}

func appendMessage(message, addition string) string {
	if message == "" {
		return addition
	}
	if addition == "" || strings.Contains(message, addition) {
		return message
	}
	return message + "\n" + addition
}

func firstMeaningful(values ...string) string {
	for _, value := range values {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "", "[no value]", "<no value>", "no value":
			continue
		default:
			return value
		}
	}
	return ""
}

func isInternalGrafanaLabel(label string) bool {
	switch label {
	case "alertname", "grafana_folder", "grafana_rule_uid", "rule_uid", "rulename":
		return true
	default:
		return strings.HasPrefix(label, "__")
	}
}

func firstAlertLabel(alerts []grafanaUnifiedAlert, key string) string {
	for _, alert := range alerts {
		if value := alert.Labels[key]; value != "" {
			return value
		}
	}
	return ""
}

func firstAlertAnnotation(alerts []grafanaUnifiedAlert, key string) string {
	for _, alert := range alerts {
		if value := alert.Annotations[key]; value != "" {
			return value
		}
	}
	return ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func formatGrafanaAlert(channel service.Channel, alert service.GrafanaAlert) any {
	switch channel {
	case service.ChannelTelegram:
		return formatGrafanaAlertForTelegram(alert)
	default:
		return formatGrafanaAlertForFeishu(alert)
	}
}

func formatGrafanaAlertForFeishu(alert service.GrafanaAlert) map[string]any {
	elements := []any{}
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

	if len(elements) == 0 {
		elements = append(elements, map[string]any{
			"tag":      "note",
			"elements": []any{map[string]any{"tag": "plain_text", "content": time.Now().UTC().Format("2006-01-02 15:04:05 UTC")}},
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
