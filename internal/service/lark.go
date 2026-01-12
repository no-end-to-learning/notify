package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"notify/internal/config"
)

type LarkService struct {
	appID     string
	appSecret string
	client    *http.Client
}

func NewLarkService(cfg config.LarkConfig) *LarkService {
	return &LarkService{
		appID:     cfg.AppID,
		appSecret: cfg.AppSecret,
		client:    &http.Client{},
	}
}

func (s *LarkService) Channel() Channel {
	return ChannelLark
}

func (s *LarkService) BuildMessage(params MessageParams) any {
	return s.buildCardMessage(params)
}

func (s *LarkService) SendMessage(to string, params MessageParams) (*SendResult, error) {
	message := s.buildCardMessage(params)
	return s.SendRawMessage(to, message)
}

func (s *LarkService) SendRawMessage(to string, message any) (*SendResult, error) {
	slog.Info("Sending Lark message", "to", to, slog.Any("payload", message))

	token, err := s.getTenantAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get tenant access token: %w", err)
	}

	content, _ := json.Marshal(message)
	reqBody := map[string]any{
		"receive_id": to,
		"msg_type":   "interactive",
		"content":    string(content),
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			MessageID string `json:"message_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("lark error: %d - %s", result.Code, result.Msg)
	}

	return &SendResult{
		Success: true,
	}, nil
}


func (s *LarkService) ListChats() ([]ChatItem, error) {
	token, err := s.getTenantAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get tenant access token: %w", err)
	}

	req, _ := http.NewRequest("GET", "https://open.feishu.cn/open-apis/im/v1/chats?page_size=100", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Items []struct {
				ChatID      string `json:"chat_id"`
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("lark error: %d - %s", result.Code, result.Msg)
	}

	chats := make([]ChatItem, len(result.Data.Items))
	for i, item := range result.Data.Items {
		chats[i] = ChatItem{
			ChatID:      item.ChatID,
			Name:        item.Name,
			Description: item.Description,
		}
	}
	return chats, nil
}

func (s *LarkService) getTenantAccessToken() (string, error) {
	reqBody := map[string]string{
		"app_id":     s.appID,
		"app_secret": s.appSecret,
	}
	body, _ := json.Marshal(reqBody)

	resp, err := s.client.Post(
		"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("get token failed: %d - %s", result.Code, result.Msg)
	}

	return result.TenantAccessToken, nil
}

func (s *LarkService) buildCardMessage(params MessageParams) map[string]any {
	message := map[string]any{
		"config":   map[string]any{"wide_screen_mode": true},
		"elements": []any{},
	}

	if params.URL != "" {
		message["card_link"] = map[string]any{"url": params.URL}
	}

	if params.Title != "" {
		color := params.Color
		if color == "" {
			color = ColorBlue
		}
		message["header"] = map[string]any{
			"title":    map[string]any{"tag": "plain_text", "content": params.Title},
			"template": string(color),
		}
	}

	elements := []any{}

	if params.Content != "" {
		elements = append(elements, map[string]any{
			"tag":     "markdown",
			"content": params.Content,
		})
	}

	if params.Note != "" {
		if params.Content != "" || params.URL != "" {
			elements = append(elements, map[string]any{"tag": "hr"})
		}
		elements = append(elements, map[string]any{
			"tag":      "note",
			"elements": []any{map[string]any{"tag": "plain_text", "content": params.Note}},
		})
	}

	message["elements"] = elements
	return message
}
