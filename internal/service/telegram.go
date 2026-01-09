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

func (s *TelegramService) SendMessage(to string, params MessageParams) (*SendResult, error) {
	text := s.buildMessage(params)
	return s.SendRawMessage(to, map[string]any{
		"text":       text,
		"parse_mode": "HTML",
	})
}

func (s *TelegramService) SendRawMessage(to string, message any) (*SendResult, error) {
	url := s.baseURL + "/sendMessage"

	payload := map[string]any{
		"chat_id": to,
	}
	if m, ok := message.(map[string]any); ok {
		for k, v := range m {
			payload[k] = v
		}
	}

	slog.Info("Sending Telegram message", "to", to, "message", payload)

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
		Success:   true,
		MessageID: fmt.Sprintf("%d", result.Result.MessageID),
	}, nil
}

func (s *TelegramService) buildMessage(params MessageParams) string {
	var parts []string

	if params.Title != "" {
		emoji := ""
		if params.Color != "" {
			emoji = ColorEmoji[params.Color]
		}
		title := escapeHTML(params.Title)
		parts = append(parts, fmt.Sprintf("<b>%s %s</b>", emoji, title))
	}

	if params.Content != "" {
		parts = append(parts, escapeHTML(params.Content))
	}

	if params.Image != "" {
		parts = append(parts, fmt.Sprintf(`<a href="%s">[Image]</a>`, escapeHTML(params.Image)))
	}

	if params.URL != "" {
		parts = append(parts, fmt.Sprintf(`<a href="%s">View Details</a>`, escapeHTML(params.URL)))
	}

	if params.Note != "" {
		parts = append(parts, fmt.Sprintf("<i>%s</i>", escapeHTML(params.Note)))
	}

	return strings.Join(parts, "\n\n")
}

func escapeHTML(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	return text
}
