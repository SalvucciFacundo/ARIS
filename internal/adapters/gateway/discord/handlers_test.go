package discord_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"aris/internal/adapters/gateway"
	"aris/internal/adapters/gateway/discord"
	"aris/internal/core/domain"
	"aris/internal/core/services"
	"github.com/bwmarrin/discordgo"
)

// MockDiscordSession records sent messages and typing calls.
type MockDiscordSession struct {
	mu           sync.Mutex
	SentTexts    []string
	SentComplex  []*discordgo.MessageSend
	TypingCalls  []string
	handlers     []interface{}
}

func (m *MockDiscordSession) ChannelMessageSend(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentTexts = append(m.SentTexts, content)
	return &discordgo.Message{ID: "msg-1", ChannelID: channelID, Content: content}, nil
}

func (m *MockDiscordSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.SentComplex = append(m.SentComplex, data)
	return &discordgo.Message{ID: "msg-complex-1", ChannelID: channelID}, nil
}

func (m *MockDiscordSession) ChannelTyping(channelID string, options ...discordgo.RequestOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.TypingCalls = append(m.TypingCalls, channelID)
	return nil
}

func (m *MockDiscordSession) AddHandler(handler interface{}) func() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
	return func() {}
}

func (m *MockDiscordSession) Open() error  { return nil }
func (m *MockDiscordSession) Close() error { return nil }

// MockGatewayEngine implements gateway.GatewayEngine for Discord tests.
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

func TestDiscordHandler_Commands(t *testing.T) {
	session := &MockDiscordSession{}
	engine := &MockGatewayEngine{}
	queue := gateway.NewJobQueue(1, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue.Start(ctx)
	defer queue.Stop()

	auth := discord.NewAuthorizer([]string{"chan-1"}, []string{"user-1"})
	handler := discord.NewHandler(session, engine, queue, auth)

	// 1. Test /help
	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ChannelID: "chan-1",
			Author:    &discordgo.User{ID: "user-1", Bot: false},
			Content:   "/help",
		},
	}

	handler.HandleMessage(ctx, msg)

	if len(session.SentTexts) == 0 {
		t.Fatalf("expected response for /help")
	}
	if !strings.Contains(session.SentTexts[0], "ARIS Discord Gateway") {
		t.Errorf("unexpected help text: %s", session.SentTexts[0])
	}

	// 2. Test /subagents
	session.SentTexts = nil
	msg.Content = "/subagents"
	handler.HandleMessage(ctx, msg)

	if len(session.SentTexts) == 0 {
		t.Fatalf("expected response for /subagents")
	}
	if !strings.Contains(session.SentTexts[0], "@director") {
		t.Errorf("unexpected subagents text: %s", session.SentTexts[0])
	}

	// 3. Test /status
	session.SentTexts = nil
	msg.Content = "/status"
	handler.HandleMessage(ctx, msg)

	if len(session.SentTexts) == 0 {
		t.Fatalf("expected response for /status")
	}
	if !strings.Contains(session.SentTexts[0], "Default Backend") {
		t.Errorf("unexpected status text: %s", session.SentTexts[0])
	}
}

func TestDiscordHandler_ImageGenerationEmbedAndAttachment(t *testing.T) {
	tmpDir := t.TempDir()
	imgFile := filepath.Join(tmpDir, "output.jpg")
	_ = os.WriteFile(imgFile, []byte("fake-jpeg-data"), 0644)

	session := &MockDiscordSession{}
	engine := &MockGatewayEngine{
		GenerateFunc: func(ctx context.Context, prompt string, opts services.GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error) {
			return &domain.ImageSpec{
				EnhancedPrompt: prompt,
				AspectRatio:    domain.RatioLandscape,
				Backend:        "pollinations",
				Model:          "flux",
				Width:          1344,
				Height:         768,
				Seed:           12345,
			}, &domain.ImageResult{
				LocalPath: imgFile,
				Metadata: map[string]interface{}{
					"critic_score": 0.88,
				},
			}, nil
		},
	}

	queue := gateway.NewJobQueue(1, 5)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	queue.Start(ctx)
	defer queue.Stop()

	auth := discord.NewAuthorizer([]string{"chan-1"}, []string{"user-1"})
	handler := discord.NewHandler(session, engine, queue, auth)

	msg := &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ChannelID: "chan-1",
			Author:    &discordgo.User{ID: "user-1", Bot: false},
			Content:   "/gen anime cyberpunk warrior in rain --ratio 16:9",
		},
	}

	handler.HandleMessage(ctx, msg)

	// Wait for queue job to complete
	time.Sleep(100 * time.Millisecond)

	session.mu.Lock()
	complexMsgs := session.SentComplex
	session.mu.Unlock()

	if len(complexMsgs) == 0 {
		t.Fatalf("expected ChannelMessageSendComplex to be called with embed and file attachment")
	}

	sent := complexMsgs[0]
	if len(sent.Files) == 0 {
		t.Errorf("expected files in message send")
	}
	if sent.Embed == nil {
		t.Fatalf("expected MessageEmbed in message send")
	}

	if !strings.Contains(sent.Embed.Title, "anime cyberpunk warrior") && !strings.Contains(sent.Embed.Description, "anime cyberpunk warrior") {
		t.Errorf("expected embed to describe prompt, got title=%q desc=%q", sent.Embed.Title, sent.Embed.Description)
	}
}
