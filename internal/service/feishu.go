package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"notify/internal/config"
)

const feishuBaseURL = "https://open.feishu.cn"

type FeishuService struct {
	appID     string
	appSecret string
	client    *http.Client
	token     string
	tokenExp  time.Time
	tokenMu   sync.RWMutex
}

func NewFeishuService(cfg config.FeishuConfig) *FeishuService {
	return &FeishuService{
		appID:     cfg.AppID,
		appSecret: cfg.AppSecret,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *FeishuService) Channel() Channel {
	return ChannelFeishu
}

func (s *FeishuService) BuildMessage(params MessageParams) any {
	return s.buildCardMessage(params)
}

func (s *FeishuService) SendMessage(target string, params MessageParams) (*SendResult, error) {
	message := s.buildCardMessage(params)
	return s.SendRawMessage(target, message)
}

func (s *FeishuService) SendRawMessage(target string, message any) (*SendResult, error) {
	slog.Info("Sending Feishu message", "target", target)

	token, err := s.getTenantAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get tenant access token: %w", err)
	}

	content, err := json.Marshal(message)
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
	}

	reqBody := map[string]any{
		"receive_id": target,
		"msg_type":   "interactive",
		"content":    string(content),
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", feishuBaseURL+"/open-apis/im/v1/messages?receive_id_type=chat_id", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Code != 0 {
		return nil, fmt.Errorf("feishu error: %d - %s", result.Code, result.Msg)
	}

	return &SendResult{Success: true}, nil
}

func (s *FeishuService) ListChats() ([]ChatItem, error) {
	token, err := s.getTenantAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get tenant access token: %w", err)
	}

	req, err := http.NewRequest("GET", feishuBaseURL+"/open-apis/im/v1/chats?page_size=100", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list chats: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

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
		return nil, fmt.Errorf("feishu error: %d - %s", result.Code, result.Msg)
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

func (s *FeishuService) getTenantAccessToken() (string, error) {
	s.tokenMu.RLock()
	if s.token != "" && time.Now().Before(s.tokenExp) {
		token := s.token
		s.tokenMu.RUnlock()
		return token, nil
	}
	s.tokenMu.RUnlock()

	s.tokenMu.Lock()
	defer s.tokenMu.Unlock()

	// double-check after acquiring write lock
	if s.token != "" && time.Now().Before(s.tokenExp) {
		return s.token, nil
	}

	reqBody := map[string]string{
		"app_id":     s.appID,
		"app_secret": s.appSecret,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	resp, err := s.client.Post(
		feishuBaseURL+"/open-apis/auth/v3/tenant_access_token/internal",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("get token failed: %d - %s", result.Code, result.Msg)
	}

	s.token = result.TenantAccessToken
	// Expire is in seconds, reduce by 60s buffer
	s.tokenExp = time.Now().Add(time.Duration(result.Expire-60) * time.Second)

	return result.TenantAccessToken, nil
}

func (s *FeishuService) buildCardMessage(params MessageParams) map[string]any {
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
