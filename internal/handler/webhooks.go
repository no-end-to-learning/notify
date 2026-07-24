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

	alert, err := decodeGrafanaAlert(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return
	}

	slog.Info("Grafana alert received",
		"state", alert.State,
		"ruleName", alert.RuleName,
		"notificationType", alert.NotificationType,
		"matches", len(alert.Matches),
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

type grafanaWebhook struct {
	Receiver          string                `json:"receiver"`
	Status            string                `json:"status"`
	Title             string                `json:"title"`
	Message           string                `json:"message"`
	Alerts            []grafanaWebhookAlert `json:"alerts"`
	GroupLabels       map[string]string     `json:"groupLabels"`
	CommonLabels      map[string]string     `json:"commonLabels"`
	CommonAnnotations map[string]string     `json:"commonAnnotations"`
	TruncatedAlerts   int                   `json:"truncatedAlerts"`
}

type grafanaWebhookAlert struct {
	Status      string              `json:"status"`
	Labels      map[string]string   `json:"labels"`
	Annotations map[string]string   `json:"annotations"`
	Values      map[string]*float64 `json:"values"`
}

type grafanaNotificationType string

const (
	grafanaNotificationTypeAlert  grafanaNotificationType = "alert"
	grafanaNotificationTypeReport grafanaNotificationType = "report"
)

type grafanaNotification struct {
	State            string
	RuleName         string
	NotificationType grafanaNotificationType
	Message          string
	Matches          []grafanaMatch
	SortOrder        string
	SortAbs          bool
}

type grafanaMatch struct {
	Summary string
	SortKey string
}

func decodeGrafanaAlert(body []byte) (grafanaNotification, error) {
	var webhook grafanaWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return grafanaNotification{}, err
	}
	if webhook.Receiver == "" || webhook.Status == "" || webhook.Alerts == nil {
		return grafanaNotification{}, fmt.Errorf("unsupported Grafana webhook payload")
	}

	alert, err := normalizeGrafanaAlert(webhook)
	if err != nil {
		return grafanaNotification{}, err
	}
	if alert.RuleName == "" || (alert.State != "alerting" && alert.State != "ok") {
		return grafanaNotification{}, fmt.Errorf("invalid Grafana webhook payload")
	}
	return alert, nil
}

func normalizeGrafanaAlert(webhook grafanaWebhook) (grafanaNotification, error) {
	state := normalizeGrafanaState(webhook.Status)
	ruleName := firstNonEmpty(
		webhook.CommonLabels["alertname"],
		webhook.GroupLabels["alertname"],
		firstAlertLabel(webhook.Alerts, "alertname"),
		webhook.Title,
	)
	alert := grafanaNotification{
		State:    state,
		RuleName: ruleName,
		NotificationType: grafanaNotificationType(strings.ToLower(firstMeaningful(
			webhook.CommonAnnotations["notificationType"],
			firstAlertAnnotation(webhook.Alerts, "notificationType"),
		))),
		Message: firstNonEmpty(
			webhook.CommonAnnotations["description"],
			firstAlertAnnotation(webhook.Alerts, "description"),
		),
		SortOrder: strings.ToLower(firstMeaningful(
			webhook.CommonAnnotations["notificationSortOrder"],
			firstAlertAnnotation(webhook.Alerts, "notificationSortOrder"),
		)),
	}
	alert.SortAbs, _ = strconv.ParseBool(firstMeaningful(
		webhook.CommonAnnotations["notificationSortAbsolute"],
		firstAlertAnnotation(webhook.Alerts, "notificationSortAbsolute"),
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

			summary := firstMeaningful(item.Annotations["summary"])
			if summary == "" {
				return grafanaNotification{}, fmt.Errorf("Grafana alert is missing summary")
			}
			alert.Matches = append(alert.Matches, grafanaMatch{
				Summary: summary,
				SortKey: firstMeaningful(item.Annotations["notificationSortKey"]),
			})
		}
		sortGrafanaMatches(&alert)
	}

	if webhook.TruncatedAlerts > 0 {
		truncated := fmt.Sprintf("Grafana omitted %d alerts from this notification", webhook.TruncatedAlerts)
		if alert.Message == "" {
			alert.Message = truncated
		} else {
			alert.Message += "\n" + truncated
		}
	}

	return alert, nil
}

func sortGrafanaMatches(alert *grafanaNotification) {
	if len(alert.Matches) < 2 || alert.SortOrder == "" {
		return
	}

	descending := alert.SortOrder == "desc"
	textSort := false
	for _, item := range alert.Matches {
		if item.SortKey == "" {
			continue
		}
		if _, err := strconv.ParseFloat(item.SortKey, 64); err != nil {
			textSort = true
			break
		}
	}
	sort.SliceStable(alert.Matches, func(i, j int) bool {
		left := alert.Matches[i]
		right := alert.Matches[j]
		leftKey := left.SortKey
		rightKey := right.SortKey
		if textSort {
			if leftKey == "" {
				leftKey = grafanaMatchText(left)
			}
			if rightKey == "" {
				rightKey = grafanaMatchText(right)
			}
		}
		if leftKey == "" || rightKey == "" {
			if leftKey == rightKey {
				return grafanaMatchText(left) < grafanaMatchText(right)
			}
			return leftKey != ""
		}

		comparison := compareGrafanaSortKeys(leftKey, rightKey, alert.SortAbs)
		if comparison == 0 {
			return grafanaMatchText(left) < grafanaMatchText(right)
		}
		if descending {
			return comparison > 0
		}
		return comparison < 0
	})
}

func grafanaMatchText(match grafanaMatch) string {
	return match.Summary
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

func firstAlertLabel(alerts []grafanaWebhookAlert, key string) string {
	for _, alert := range alerts {
		if value := alert.Labels[key]; value != "" {
			return value
		}
	}
	return ""
}

func firstAlertAnnotation(alerts []grafanaWebhookAlert, key string) string {
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

func formatGrafanaAlert(channel service.Channel, alert grafanaNotification) any {
	switch channel {
	case service.ChannelTelegram:
		return formatGrafanaAlertForTelegram(alert)
	default:
		return formatGrafanaAlertForFeishu(alert)
	}
}

func formatGrafanaAlertForFeishu(alert grafanaNotification) map[string]any {
	elements := []any{}
	var template, title string

	switch {
	case alert.NotificationType == grafanaNotificationTypeReport:
		template = string(service.ColorBlue)
		title = alert.RuleName
	case alert.State == "alerting":
		template = string(service.ColorOrange)
		title = alert.RuleName
	case alert.State == "ok":
		template = string(service.ColorGreen)
		title = "✅ " + alert.RuleName
	default:
		template = string(service.ColorGrey)
		title = alert.RuleName
	}

	if len(alert.Matches) > 0 {
		var items []string
		for _, item := range alert.Matches {
			items = append(items, item.Summary)
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

func formatGrafanaAlertForTelegram(alert grafanaNotification) map[string]any {
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

	// Content: matches
	if len(alert.Matches) > 0 {
		var items []string
		for _, item := range alert.Matches {
			items = append(items, service.EscapeHTML(item.Summary))
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
