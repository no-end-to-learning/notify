package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"notify/internal/config"
)

type WecomService struct {
	webhookURL string
	client     *http.Client
}

func NewWecomService(cfg config.WecomConfig) *WecomService {
	return &WecomService{
		webhookURL: cfg.WebhookURL,
		client:     &http.Client{},
	}
}

func (s *WecomService) Channel() Channel {
	return ChannelWecom
}

func (s *WecomService) SendMessage(to string, params MessageParams) (*SendResult, error) {
	message := s.buildMessage(params)
	return s.SendRawMessage(to, message)
}

func (s *WecomService) SendRawMessage(to string, message any) (*SendResult, error) {
	url := fmt.Sprintf("%s?key=%s", s.webhookURL, to)
	slog.Info("Sending WeCom message", "to", to, "message", message)

	body, _ := json.Marshal(message)
	resp, err := s.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("wecom error: %d - %s", result.ErrCode, result.ErrMsg)
	}

	return &SendResult{Success: true}, nil
}

func (s *WecomService) buildMessage(params MessageParams) map[string]any {
	if params.Image != "" || params.URL != "" {
		return s.buildNewsMessage(params)
	}
	return s.buildMarkdownMessage(params)
}

func (s *WecomService) buildMarkdownMessage(params MessageParams) map[string]any {
	var parts []string

	if params.Title != "" {
		emoji := ""
		if params.Color != "" {
			emoji = ColorEmoji[params.Color]
		}
		parts = append(parts, fmt.Sprintf("### %s %s", emoji, params.Title))
	}

	if params.Content != "" {
		parts = append(parts, params.Content)
	}

	if params.Note != "" {
		parts = append(parts, fmt.Sprintf("> %s", params.Note))
	}

	return map[string]any{
		"msgtype": "markdown",
		"markdown": map[string]any{
			"content": strings.Join(parts, "\n\n"),
		},
	}
}

func (s *WecomService) buildNewsMessage(params MessageParams) map[string]any {
	title := params.Title
	if title == "" {
		title = "Notification"
	}

	var descParts []string
	if params.Content != "" {
		descParts = append(descParts, params.Content)
	}
	if params.Note != "" {
		descParts = append(descParts, params.Note)
	}

	article := map[string]any{
		"title": title,
	}
	if len(descParts) > 0 {
		article["description"] = strings.Join(descParts, "\n\n")
	}
	if params.URL != "" {
		article["url"] = params.URL
	}
	if params.Image != "" {
		article["picurl"] = params.Image
	}

	return map[string]any{
		"msgtype": "news",
		"news": map[string]any{
			"articles": []any{article},
		},
	}
}
