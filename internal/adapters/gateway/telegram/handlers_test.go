package telegram_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"aris/internal/adapters/gateway"
	"aris/internal/adapters/gateway/telegram"
	"aris/internal/core/domain"
	"aris/internal/core/services"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// MockBotAPI records sent messages for verification in tests.
type MockBotAPI struct {
	mu           sync.Mutex
	SentMessages []tgbotapi.Chattable
}

func (m *MockBotAPI) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentMessages = append(m.SentMessages, c)
	return tgbotapi.Message{MessageID: len(m.SentMessages)}, nil
}

func (m *MockBotAPI) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	ch := make(chan tgbotapi.Update)
	return ch
}

func (m *MockBotAPI) StopReceivingUpdates() {}

// MockGatewayEngine implements gateway.GatewayEngine for testing.
type MockGatewayEngine struct {
	GenerateFunc         func(ctx context.Context, prompt string, opts services.GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error)
	ExecuteSubagentFunc  func(ctx context.Context, subagent, prompt string) (string, error)
	PipelineGenerateFunc func(ctx context.Context, prompt string, opts services.PipelineOptions) (*services.PipelineResult, error)
	SearchMemoryFunc     func(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error)
	ListSubagentsFunc    func(ctx context.Context) ([]domain.SubagentDef, error)
	ListBackendsFunc     func() []string
	GetDefaultBackendFunc func() string
	StatusFunc           func(ctx context.Context) (gateway.GatewayStatus, error)
}

func (m *MockGatewayEngine) Generate(ctx context.Context, prompt string, opts services.GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error) {
	if m.GenerateFunc != nil {
		return m.GenerateFunc(ctx, prompt, opts)
	}
	return &domain.ImageSpec{EnhancedPrompt: prompt, AspectRatio: "1:1", Backend: "pollinations", Width: 1024, Height: 1024, Seed: 123},
		&domain.ImageResult{LocalPath: "/tmp/fake.jpg"}, nil
}

func (m *MockGatewayEngine) ExecuteSubagent(ctx context.Context, subagent, prompt string) (string, error) {
	if m.ExecuteSubagentFunc != nil {
		return m.ExecuteSubagentFunc(ctx, subagent, prompt)
	}
	return "Subagent response for: " + prompt, nil
}

func (m *MockGatewayEngine) PipelineGenerate(ctx context.Context, prompt string, opts services.PipelineOptions) (*services.PipelineResult, error) {
	if m.PipelineGenerateFunc != nil {
		return m.PipelineGenerateFunc(ctx, prompt, opts)
	}
	return &services.PipelineResult{
		DirectorConcept: "cinematic vision",
		PromptSmithSpec: &domain.ImageSpec{EnhancedPrompt: prompt, AspectRatio: "16:9", Width: 1344, Height: 768, Backend: "pollinations"},
		ImageResult:     &domain.ImageResult{LocalPath: "/tmp/fake.jpg"},
		CriticScore:     0.95,
		Duration:        2 * time.Second,
	}, nil
}

func (m *MockGatewayEngine) SearchMemory(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error) {
	if m.SearchMemoryFunc != nil {
		return m.SearchMemoryFunc(ctx, query, scope, limit)
	}
	return []domain.KnowledgeFact{
		{Topic: "style:cyberpunk", Concept: "lighting", Fact: "neon reflections and rain"},
	}, nil
}

func (m *MockGatewayEngine) ListSubagents(ctx context.Context) ([]domain.SubagentDef, error) {
	if m.ListSubagentsFunc != nil {
		return m.ListSubagentsFunc(ctx)
	}
	return []domain.SubagentDef{
		{Name: "director", DisplayName: "Art Director", Role: "Scene Composition"},
	}, nil
}

func (m *MockGatewayEngine) ListBackends() []string {
	if m.ListBackendsFunc != nil {
		return m.ListBackendsFunc()
	}
	return []string{"pollinations", "comfyui"}
}

func (m *MockGatewayEngine) GetDefaultBackend() string {
	if m.GetDefaultBackendFunc != nil {
		return m.GetDefaultBackendFunc()
	}
	return "pollinations"
}

func (m *MockGatewayEngine) Status(ctx context.Context) (gateway.GatewayStatus, error) {
	if m.StatusFunc != nil {
		return m.StatusFunc(ctx)
	}
	return gateway.GatewayStatus{
		Uptime:         10 * time.Minute,
		DefaultBackend: "pollinations",
		DefaultModel:   "flux",
		LLMProvider:    "passthrough",
		LLMModel:       "gpt-4o-mini",
	}, nil
}

func TestTelegramHandler_HelpAndInfoCommands(t *testing.T) {
	bot := &MockBotAPI{}
	engine := &MockGatewayEngine{}
	queue := gateway.NewJobQueue(1, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue.Start(ctx)
	defer queue.Stop()

	auth := telegram.NewAuthorizer([]int64{100}, []int64{200})
	handler := telegram.NewHandler(bot, engine, queue, auth, false)

	// 1. Test /help
	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 100},
			From: &tgbotapi.User{ID: 200},
			Text: "/help",
		},
	}

	handler.HandleUpdate(ctx, update)

	if len(bot.SentMessages) == 0 {
		t.Fatalf("expected response message for /help")
	}

	msg, ok := bot.SentMessages[0].(tgbotapi.MessageConfig)
	if !ok || !strings.Contains(msg.Text, "ARIS Telegram Gateway") {
		t.Errorf("unexpected /help response: %v", msg.Text)
	}

	// 2. Test /subagents
	bot.SentMessages = nil
	update.Message.Text = "/subagents"
	handler.HandleUpdate(ctx, update)

	if len(bot.SentMessages) == 0 {
		t.Fatalf("expected response for /subagents")
	}
	msg = bot.SentMessages[0].(tgbotapi.MessageConfig)
	if !strings.Contains(msg.Text, "@director") {
		t.Errorf("unexpected /subagents output: %s", msg.Text)
	}

	// 3. Test /backends
	bot.SentMessages = nil
	update.Message.Text = "/backends"
	handler.HandleUpdate(ctx, update)

	if len(bot.SentMessages) == 0 {
		t.Fatalf("expected response for /backends")
	}
	msg = bot.SentMessages[0].(tgbotapi.MessageConfig)
	if !strings.Contains(msg.Text, "pollinations") {
		t.Errorf("unexpected /backends output: %s", msg.Text)
	}

	// 4. Test /status
	bot.SentMessages = nil
	update.Message.Text = "/status"
	handler.HandleUpdate(ctx, update)

	if len(bot.SentMessages) == 0 {
		t.Fatalf("expected response for /status")
	}
	msg = bot.SentMessages[0].(tgbotapi.MessageConfig)
	if !strings.Contains(msg.Text, "Default Backend") {
		t.Errorf("unexpected /status output: %s", msg.Text)
	}

	// 5. Test /memory
	bot.SentMessages = nil
	update.Message.Text = "/memory cyberpunk"
	handler.HandleUpdate(ctx, update)

	if len(bot.SentMessages) == 0 {
		t.Fatalf("expected response for /memory")
	}
	msg = bot.SentMessages[0].(tgbotapi.MessageConfig)
	if !strings.Contains(msg.Text, "neon reflections") {
		t.Errorf("unexpected /memory output: %s", msg.Text)
	}
}

func TestTelegramHandler_ImageGenerationDelivery(t *testing.T) {
	// Create temporary dummy image file
	tmpDir := t.TempDir()
	imgFile := filepath.Join(tmpDir, "test.jpg")
	_ = os.WriteFile(imgFile, []byte("fake-jpeg-data"), 0644)

	bot := &MockBotAPI{}
	engine := &MockGatewayEngine{
		GenerateFunc: func(ctx context.Context, prompt string, opts services.GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error) {
			return &domain.ImageSpec{
				EnhancedPrompt: prompt,
				AspectRatio:    domain.RatioLandscape,
				Backend:        "pollinations",
				Model:          "flux",
				Width:          1344,
				Height:         768,
				Seed:           9999,
			}, &domain.ImageResult{
				LocalPath: imgFile,
			}, nil
		},
	}

	queue := gateway.NewJobQueue(1, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue.Start(ctx)
	defer queue.Stop()

	auth := telegram.NewAuthorizer([]int64{100}, []int64{200})
	handler := telegram.NewHandler(bot, engine, queue, auth, false)

	update := &tgbotapi.Update{
		Message: &tgbotapi.Message{
			Chat: &tgbotapi.Chat{ID: 100},
			From: &tgbotapi.User{ID: 200},
			Text: "/gen futuristic city --ratio 16:9",
		},
	}

	handler.HandleUpdate(ctx, update)

	// Wait for async queue job to complete
	time.Sleep(100 * time.Millisecond)

	bot.mu.Lock()
	messages := bot.SentMessages
	bot.mu.Unlock()

	var foundPhoto bool
	for _, m := range messages {
		if photo, ok := m.(tgbotapi.PhotoConfig); ok {
			foundPhoto = true
			if !strings.Contains(photo.Caption, "futuristic city") {
				t.Errorf("expected caption to contain prompt, got: %s", photo.Caption)
			}
			if !strings.Contains(photo.Caption, "16:9") {
				t.Errorf("expected caption to contain ratio, got: %s", photo.Caption)
			}
		}
	}

	if !foundPhoto {
		t.Fatalf("expected tgbotapi.PhotoConfig message to be sent")
	}
}
