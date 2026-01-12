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

func (s *TelegramService) BuildMessage(params MessageParams) any {
	text := s.buildMessage(params)
	return map[string]any{
		"text":       text,
		"parse_mode": "MarkdownV2",
	}
}

func (s *TelegramService) SendMessage(to string, params MessageParams) (*SendResult, error) {
	text := s.buildMessage(params)
	return s.SendRawMessage(to, map[string]any{
		"text":       text,
		"parse_mode": "MarkdownV2",
	})
}

func (s *TelegramService) SendRawMessage(to string, message any) (*SendResult, error) {
	url := s.baseURL + "/sendMessage"

	payload := map[string]any{
		"chat_id": to,
		"link_preview_options": map[string]bool{
			"is_disabled": true,
		},
	}
	if m, ok := message.(map[string]any); ok {
		for k, v := range m {
			payload[k] = v
		}
	}

	slog.Info("Sending Telegram message", "to", to, slog.Any("payload", payload))

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
		emoji := ""
		if params.Color != "" {
			emoji = ColorEmoji[params.Color]
		}
		title := EscapeMarkdown(params.Title)
		parts = append(parts, fmt.Sprintf("*%s %s*", emoji, title))
	}

	if params.Content != "" {
		parts = append(parts, EscapeMarkdown(params.Content))
	}

	if params.URL != "" {
		parts = append(parts, fmt.Sprintf("[View Details](%s)", params.URL))
	}

	if params.Note != "" {
		// Use blockquote for note
		noteLines := strings.Split(params.Note, "\n")
		var quotedLines []string
		for _, line := range noteLines {
			quotedLines = append(quotedLines, "> "+EscapeMarkdown(line))
		}
		parts = append(parts, strings.Join(quotedLines, "\n"))
	}

	return strings.Join(parts, "\n\n")
}

func EscapeMarkdown(text string) string {
	// Escape MarkdownV2 special characters
	// Characters that need escaping: _ * [ ] ( ) ~ ` > # + - = | { } . !
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}
