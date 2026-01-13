package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"notify/internal/config"
)

type TelegramService struct {
	botToken string
	baseURL  string
	client   *http.Client
}

func NewTelegramService(cfg config.TelegramConfig) *TelegramService {
	return &TelegramService{
		botToken: cfg.BotToken,
		baseURL:  fmt.Sprintf("https://api.telegram.org/bot%s", cfg.BotToken),
		client:   &http.Client{},
	}
}

func (s *TelegramService) Channel() Channel {
	return ChannelTelegram
}

func (s *TelegramService) BuildMessage(params MessageParams) any {
	text := s.buildMessage(params)
	return map[string]any{
		"text":       text,
		"parse_mode": "HTML",
	}
}

func (s *TelegramService) SendMessage(target string, params MessageParams) (*SendResult, error) {
	text := s.buildMessage(params)
	return s.SendRawMessage(target, map[string]any{
		"text":       text,
		"parse_mode": "HTML",
	})
}

func (s *TelegramService) SendRawMessage(target string, message any) (*SendResult, error) {
	url := s.baseURL + "/sendMessage"

	chatID := target
	var threadID int
	if idx := strings.LastIndex(target, ":"); idx != -1 {
		chatID = target[:idx]
		if tid, err := strconv.Atoi(target[idx+1:]); err == nil {
			threadID = tid
		}
	}

	payload := map[string]any{
		"chat_id": chatID,
		"link_preview_options": map[string]bool{
			"is_disabled": true,
		},
	}

	if threadID != 0 {
		payload["message_thread_id"] = threadID
	}

	if m, ok := message.(map[string]any); ok {
		for k, v := range m {
			payload[k] = v
		}
	}

	slog.Info("Sending Telegram message", "target", target, slog.Any("payload", payload))

	body, _ := json.Marshal(payload)
	resp, err := s.client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK          bool   `json:"ok"`
		ErrorCode   int    `json:"error_code"`
		Description string `json:"description"`
		Result      struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("telegram error: %d - %s", result.ErrorCode, result.Description)
	}

	return &SendResult{
		Success: true,
	}, nil
}

func (s *TelegramService) buildMessage(params MessageParams) string {
	var parts []string

	if params.Title != "" {
		title := EscapeHTML(params.Title)
		parts = append(parts, fmt.Sprintf("<b>%s</b>", title))
	}

	if params.Content != "" {
		parts = append(parts, EscapeHTML(params.Content))
	}

	if params.URL != "" {
		parts = append(parts, fmt.Sprintf("<a href=\"%s\">View Details</a>", params.URL))
	}

	if params.Note != "" {
		// Use italic for note
		parts = append(parts, "<i>"+EscapeHTML(params.Note)+"</i>")
	}

	return strings.Join(parts, "\n\n")
}

func EscapeHTML(text string) string {
	// Escape HTML special characters
	replacer := strings.NewReplacer(
		"<", "&lt;",
		">", "&gt;",
		"&", "&amp;",
	)
	return replacer.Replace(text)
}
