// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all runtime settings. Poll fields default to the main bot/chat
// when their dedicated env vars are not set.
type Config struct {
	BotToken     string
	ChatID       string
	PollBotToken string
	PollChatID   string
	City         string
	Categories   string
	PeriodDays   int
	TestMode     bool
}

// Load reads environment variables and applies defaults.
func Load() Config {
	cfg := Config{
		BotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		ChatID:     os.Getenv("TELEGRAM_CHAT_ID"),
		City:       envOr("CITY", "Winnipeg"),
		Categories: envOr("CATEGORIES", "tech"),
		PeriodDays: envInt("PERIOD_DAYS", 30),
		TestMode:   os.Getenv("TEST_MODE") == "true",
	}
	cfg.PollBotToken = envOr("TELEGRAM_POLL_BOT_TOKEN", cfg.BotToken)
	cfg.PollChatID = envOr("TELEGRAM_POLL_CHAT_ID", cfg.ChatID)
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
