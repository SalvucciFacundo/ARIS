package telegram_test

import (
	"context"
	"testing"
	"time"

	"aris/internal/adapters/gateway"
	"aris/internal/adapters/gateway/telegram"
	"aris/internal/config"
)

func TestTelegramAdapter_Lifecycle(t *testing.T) {
	cfg := config.TelegramConfig{
		Enabled:  true,
		BotToken: "mock-token",
	}

	mockBot := &MockBotAPI{}
	engine := &MockGatewayEngine{}
	queue := gateway.NewJobQueue(1, 2)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue.Start(ctx)
	defer queue.Stop()

	adapter := telegram.NewAdapter(cfg, engine, queue)
	adapter.SetBot(mockBot)

	if adapter.Name() != "telegram" {
		t.Errorf("expected name 'telegram', got %q", adapter.Name())
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
