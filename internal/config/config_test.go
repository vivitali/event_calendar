package config

import (
	"testing"
)

func TestLoad_DefaultsAndFallbacks(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "main-token")
	t.Setenv("TELEGRAM_CHAT_ID", "main-chat")
	// Poll envs unset — should fall back.
	cfg := Load()
	if cfg.BotToken != "main-token" || cfg.ChatID != "main-chat" {
		t.Fatalf("main creds wrong: %+v", cfg)
	}
	if cfg.PollBotToken != "main-token" || cfg.PollChatID != "main-chat" {
		t.Errorf("poll did not fall back to main: %+v", cfg)
	}
	if cfg.City != "Winnipeg" || cfg.Categories != "tech" || cfg.PeriodDays != 30 {
		t.Errorf("defaults wrong: %+v", cfg)
	}
	if cfg.TestMode {
		t.Errorf("TestMode should default to false")
	}
}

func TestLoad_OverridesAndExplicitPollCreds(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "main")
	t.Setenv("TELEGRAM_CHAT_ID", "mainchat")
	t.Setenv("TELEGRAM_POLL_BOT_TOKEN", "poll")
	t.Setenv("TELEGRAM_POLL_CHAT_ID", "pollchat")
	t.Setenv("CITY", "Toronto")
	t.Setenv("CATEGORIES", "ai")
	t.Setenv("PERIOD_DAYS", "14")
	t.Setenv("TEST_MODE", "true")

	cfg := Load()
	if cfg.PollBotToken != "poll" || cfg.PollChatID != "pollchat" {
		t.Errorf("explicit poll creds not honored: %+v", cfg)
	}
	if cfg.City != "Toronto" || cfg.Categories != "ai" || cfg.PeriodDays != 14 || !cfg.TestMode {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}

func TestLoad_BadPeriodDays(t *testing.T) {
	t.Setenv("PERIOD_DAYS", "not-a-number")
	cfg := Load()
	if cfg.PeriodDays != 30 {
		t.Errorf("expected fallback to 30 on bad PERIOD_DAYS, got %d", cfg.PeriodDays)
	}
}
