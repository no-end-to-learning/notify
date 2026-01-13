package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Lark     LarkConfig
	Telegram TelegramConfig
	Queue    QueueConfig
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

type TelegramConfig struct {
	BotToken string
}

type QueueConfig struct {
	RatePerSecond float64
	MaxRetries    int
	RetryDelay    time.Duration
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
		Telegram: TelegramConfig{
			BotToken: getEnv("APP_TELEGRAM_BOT_TOKEN", ""),
		},
		Queue: QueueConfig{
			RatePerSecond: getEnvFloat("QUEUE_RATE_LIMIT", 1.0),
			MaxRetries:    getEnvInt("QUEUE_MAX_RETRIES", 3),
			RetryDelay:    getEnvDuration("QUEUE_RETRY_DELAY", time.Second),
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
