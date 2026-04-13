package service

import (
	"fmt"
	"strings"

	"notify/internal/config"
)

type NotifyService interface {
	Channel() Channel
	SendMessage(target string, params MessageParams) (*SendResult, error)
	SendRawMessage(target string, message any) (*SendResult, error)
	BuildMessage(params MessageParams) any
}

type ChatLister interface {
	ListChats() ([]ChatItem, error)
}

var services map[Channel]NotifyService

func Init(cfg *config.Config) {
	services = map[Channel]NotifyService{
		ChannelFeishu:   NewFeishuService(cfg.Feishu),
		ChannelTelegram: NewTelegramService(cfg.Telegram),
	}
}

func GetService(channel Channel) (NotifyService, error) {
	svc, ok := services[channel]
	if !ok {
		return nil, fmt.Errorf("unknown channel: %s", channel)
	}
	return svc, nil
}

func ValidateChannel(s string) (Channel, error) {
	switch Channel(strings.ToLower(s)) {
	case ChannelFeishu, ChannelTelegram:
		return Channel(strings.ToLower(s)), nil
	default:
		return "", fmt.Errorf("invalid channel: %s", s)
	}
}
