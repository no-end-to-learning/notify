package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Feishu   FeishuConfig
	Telegram TelegramConfig
	Queue    QueueConfig
}

type ServerConfig struct {
	Host    string
	Port    int
	BaseURL string
}

type FeishuConfig struct {
	AppID     string
	AppSecret string
}

type TelegramConfig struct {
	BotToken string
}

type QueueConfig struct {
	RatePerSecond float64
	MaxAttempts   int
	RetryDelay    time.Duration
	BufferSize    int
	IdleTimeout   time.Duration
}

func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host:    getEnv("APP_SERVER_HOST", "0.0.0.0"),
			Port:    getEnvInt("APP_SERVER_PORT", 8000),
			BaseURL: getEnv("APP_SERVER_BASE_URL", "http://localhost:8000/"),
		},
		Feishu: FeishuConfig{
			AppID:     getEnv("APP_FEISHU_ID", ""),
			AppSecret: getEnv("APP_FEISHU_SECRET", ""),
		},
		Telegram: TelegramConfig{
			BotToken: getEnv("APP_TELEGRAM_BOT_TOKEN", ""),
		},
		Queue: QueueConfig{
			RatePerSecond: getEnvFloat("QUEUE_RATE_LIMIT", 1.0),
			MaxAttempts:   getEnvInt("QUEUE_MAX_ATTEMPTS", 3),
			RetryDelay:    getEnvDuration("QUEUE_RETRY_DELAY", time.Second),
			BufferSize:    getEnvInt("QUEUE_BUFFER_SIZE", 1000),
			IdleTimeout:   getEnvDuration("QUEUE_IDLE_TIMEOUT", 5*time.Minute),
		},
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	feishuPartial := (c.Feishu.AppID == "") != (c.Feishu.AppSecret == "")
	if feishuPartial {
		return fmt.Errorf("feishu: APP_FEISHU_ID and APP_FEISHU_SECRET must both be set")
	}

	if c.Feishu.AppID == "" && c.Telegram.BotToken == "" {
		return fmt.Errorf("at least one service must be configured (feishu or telegram)")
	}

	return nil
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

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
			return floatVal
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if durationVal, err := time.ParseDuration(value); err == nil {
			return durationVal
		}
	}
	return defaultValue
}
