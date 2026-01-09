package service

import (
	"fmt"

	"notify/internal/config"
)

type NotifyService interface {
	Channel() Channel
	SendMessage(to string, params MessageParams) (*SendResult, error)
	SendRawMessage(to string, message any) (*SendResult, error)
}

type ChatLister interface {
	ListChats() ([]ChatItem, error)
}

type ImageUploader interface {
	UploadImage(imageURL string) (string, error)
}

var services map[Channel]NotifyService

func Init(cfg *config.Config) {
	services = map[Channel]NotifyService{
		ChannelLark:     NewLarkService(cfg.Lark),
		ChannelWecom:    NewWecomService(cfg.Wecom),
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
	switch Channel(s) {
	case ChannelLark, ChannelWecom, ChannelTelegram:
		return Channel(s), nil
	default:
		return "", fmt.Errorf("invalid channel: %s", s)
	}
}
