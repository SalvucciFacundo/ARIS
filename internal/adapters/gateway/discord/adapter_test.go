package discord_test

import (
	"context"
	"testing"
	"time"

	"aris/internal/adapters/gateway"
	"aris/internal/adapters/gateway/discord"
	"aris/internal/config"
)

func TestDiscordAdapter_Lifecycle(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		BotToken: "mock-discord-token",
	}

	session := &MockDiscordSession{}
	engine := &MockGatewayEngine{}
	queue := gateway.NewJobQueue(1, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue.Start(ctx)
	defer queue.Stop()

	adapter := discord.NewAdapter(cfg, engine, queue)
	adapter.SetSession(session)

	if adapter.Name() != "discord" {
		t.Errorf("expected adapter name 'discord', got %q", adapter.Name())
	}

	if err := adapter.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	time.Sleep(20 * time.Millisecond)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	if err := adapter.Stop(stopCtx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
