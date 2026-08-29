package config_test

import (
	"os"
	"testing"

	"aris/internal/config"
)

func TestDefaultGatewayConfig(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.Gateway.Concurrency != 1 {
		t.Errorf("expected Gateway.Concurrency=1, got %d", cfg.Gateway.Concurrency)
	}
	if cfg.Gateway.MaxQueue != 10 {
		t.Errorf("expected Gateway.MaxQueue=10, got %d", cfg.Gateway.MaxQueue)
	}
	if cfg.Gateway.Telegram.Enabled != false {
		t.Errorf("expected Gateway.Telegram.Enabled=false, got %v", cfg.Gateway.Telegram.Enabled)
	}
	if cfg.Gateway.Telegram.SendAsDocument != false {
		t.Errorf("expected Gateway.Telegram.SendAsDocument=false, got %v", cfg.Gateway.Telegram.SendAsDocument)
	}
	if cfg.Gateway.Discord.Enabled != false {
		t.Errorf("expected Gateway.Discord.Enabled=false, got %v", cfg.Gateway.Discord.Enabled)
	}
	if len(cfg.Gateway.Telegram.AllowedChatIDs) != 0 {
		t.Errorf("expected empty Telegram.AllowedChatIDs, got %v", cfg.Gateway.Telegram.AllowedChatIDs)
	}
	if len(cfg.Gateway.Telegram.AllowedUserIDs) != 0 {
		t.Errorf("expected empty Telegram.AllowedUserIDs, got %v", cfg.Gateway.Telegram.AllowedUserIDs)
	}
	if len(cfg.Gateway.Discord.AllowedChannelIDs) != 0 {
		t.Errorf("expected empty Discord.AllowedChannelIDs, got %v", cfg.Gateway.Discord.AllowedChannelIDs)
	}
	if len(cfg.Gateway.Discord.AllowedUserIDs) != 0 {
		t.Errorf("expected empty Discord.AllowedUserIDs, got %v", cfg.Gateway.Discord.AllowedUserIDs)
	}
}

func TestGatewayConfigEnvironmentOverrides(t *testing.T) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "tg-secret-123")
	os.Setenv("DISCORD_BOT_TOKEN", "dc-secret-456")
	os.Setenv("TELEGRAM_ALLOWED_CHAT_IDS", "1001,1002")
	os.Setenv("TELEGRAM_ALLOWED_USER_IDS", "9001,9002")
	os.Setenv("DISCORD_ALLOWED_CHANNEL_IDS", "2001,2002")
	os.Setenv("DISCORD_ALLOWED_USER_IDS", "8001,8002")
	os.Setenv("ARIS_GATEWAY_CONCURRENCY", "2")
	os.Setenv("ARIS_GATEWAY_MAX_QUEUE", "25")
	os.Setenv("ARIS_TELEGRAM_SEND_DOCUMENT", "true")

	defer func() {
		os.Unsetenv("TELEGRAM_BOT_TOKEN")
		os.Unsetenv("DISCORD_BOT_TOKEN")
		os.Unsetenv("TELEGRAM_ALLOWED_CHAT_IDS")
		os.Unsetenv("TELEGRAM_ALLOWED_USER_IDS")
		os.Unsetenv("DISCORD_ALLOWED_CHANNEL_IDS")
		os.Unsetenv("DISCORD_ALLOWED_USER_IDS")
		os.Unsetenv("ARIS_GATEWAY_CONCURRENCY")
		os.Unsetenv("ARIS_GATEWAY_MAX_QUEUE")
		os.Unsetenv("ARIS_TELEGRAM_SEND_DOCUMENT")
	}()

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Gateway.Telegram.BotToken != "tg-secret-123" {
		t.Errorf("expected Telegram.BotToken='tg-secret-123', got %q", cfg.Gateway.Telegram.BotToken)
	}
	if !cfg.Gateway.Telegram.Enabled {
		t.Errorf("expected Telegram.Enabled=true when token set, got false")
	}
	if cfg.Gateway.Discord.BotToken != "dc-secret-456" {
		t.Errorf("expected Discord.BotToken='dc-secret-456', got %q", cfg.Gateway.Discord.BotToken)
	}
	if !cfg.Gateway.Discord.Enabled {
		t.Errorf("expected Discord.Enabled=true when token set, got false")
	}
	if len(cfg.Gateway.Telegram.AllowedChatIDs) != 2 || cfg.Gateway.Telegram.AllowedChatIDs[0] != 1001 || cfg.Gateway.Telegram.AllowedChatIDs[1] != 1002 {
		t.Errorf("unexpected Telegram.AllowedChatIDs: %v", cfg.Gateway.Telegram.AllowedChatIDs)
	}
	if len(cfg.Gateway.Telegram.AllowedUserIDs) != 2 || cfg.Gateway.Telegram.AllowedUserIDs[0] != 9001 || cfg.Gateway.Telegram.AllowedUserIDs[1] != 9002 {
		t.Errorf("unexpected Telegram.AllowedUserIDs: %v", cfg.Gateway.Telegram.AllowedUserIDs)
	}
	if len(cfg.Gateway.Discord.AllowedChannelIDs) != 2 || cfg.Gateway.Discord.AllowedChannelIDs[0] != "2001" || cfg.Gateway.Discord.AllowedChannelIDs[1] != "2002" {
		t.Errorf("unexpected Discord.AllowedChannelIDs: %v", cfg.Gateway.Discord.AllowedChannelIDs)
	}
	if len(cfg.Gateway.Discord.AllowedUserIDs) != 2 || cfg.Gateway.Discord.AllowedUserIDs[0] != "8001" || cfg.Gateway.Discord.AllowedUserIDs[1] != "8002" {
		t.Errorf("unexpected Discord.AllowedUserIDs: %v", cfg.Gateway.Discord.AllowedUserIDs)
	}
	if cfg.Gateway.Concurrency != 2 {
		t.Errorf("expected Concurrency=2, got %d", cfg.Gateway.Concurrency)
	}
	if cfg.Gateway.MaxQueue != 25 {
		t.Errorf("expected MaxQueue=25, got %d", cfg.Gateway.MaxQueue)
	}
	if !cfg.Gateway.Telegram.SendAsDocument {
		t.Errorf("expected Telegram.SendAsDocument=true, got false")
	}
}
