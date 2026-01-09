package config

import (
	"os"
	"strconv"
)

type Config struct {
	Server   ServerConfig
	Lark     LarkConfig
	Wecom    WecomConfig
	Telegram TelegramConfig
}

type ServerConfig struct {
	Host    string
	Port    int
	BaseURL string
}

type LarkConfig struct {
	AppID     string
	AppSecret string
}

type WecomConfig struct {
	WebhookURL string
}

type TelegramConfig struct {
	BotToken string
}

func Load() *Config {
	return &Config{
		Server: ServerConfig{
			Host:    getEnv("APP_SERVER_HOST", "0.0.0.0"),
			Port:    getEnvInt("APP_SERVER_PORT", 8000),
			BaseURL: getEnv("APP_SERVER_BASE_URL", "http://localhost:8000/"),
		},
		Lark: LarkConfig{
			AppID:     getEnv("APP_LARK_ID", ""),
			AppSecret: getEnv("APP_LARK_SECRET", ""),
		},
		Wecom: WecomConfig{
			WebhookURL: getEnv("APP_WECOM_WEBHOOK_URL", "https://qyapi.weixin.qq.com/cgi-bin/webhook/send"),
		},
		Telegram: TelegramConfig{
			BotToken: getEnv("APP_TELEGRAM_BOT_TOKEN", ""),
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
