package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"aris/internal/adapters/gateway"
	"aris/internal/adapters/gateway/discord"
	"aris/internal/adapters/gateway/telegram"
	"aris/internal/config"
	"aris/internal/core/domain"
	"aris/internal/core/services"
	"github.com/bwmarrin/discordgo"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type E2ETelegramBot struct {
	mu           sync.Mutex
	SentMessages []tgbotapi.Chattable
	updatesCh    chan tgbotapi.Update
}

func NewE2ETelegramBot() *E2ETelegramBot {
	return &E2ETelegramBot{
		updatesCh: make(chan tgbotapi.Update, 100),
	}
}

func (b *E2ETelegramBot) Send(c tgbotapi.Chattable) (tgbotapi.Message, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.SentMessages = append(b.SentMessages, c)
	return tgbotapi.Message{MessageID: len(b.SentMessages)}, nil
}

func (b *E2ETelegramBot) GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel {
	return b.updatesCh
}

func (b *E2ETelegramBot) StopReceivingUpdates() {
	close(b.updatesCh)
}

type E2EDiscordSession struct {
	mu          sync.Mutex
	SentTexts   []string
	SentComplex []*discordgo.MessageSend
	handlers    []func(*discordgo.Session, *discordgo.MessageCreate)
}

func NewE2EDiscordSession() *E2EDiscordSession {
	return &E2EDiscordSession{}
}

func (s *E2EDiscordSession) ChannelMessageSend(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SentTexts = append(s.SentTexts, content)
	return &discordgo.Message{ID: "msg-1", ChannelID: channelID, Content: content}, nil
}

func (s *E2EDiscordSession) ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SentComplex = append(s.SentComplex, data)
	return &discordgo.Message{ID: "msg-complex-1", ChannelID: channelID}, nil
}

func (s *E2EDiscordSession) ChannelTyping(channelID string, options ...discordgo.RequestOption) error {
	return nil
}

func (s *E2EDiscordSession) AddHandler(handler interface{}) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if fn, ok := handler.(func(*discordgo.Session, *discordgo.MessageCreate)); ok {
		s.handlers = append(s.handlers, fn)
	}
	return func() {}
}

func (s *E2EDiscordSession) EmitMessage(m *discordgo.MessageCreate) {
	s.mu.Lock()
	handlers := append([]func(*discordgo.Session, *discordgo.MessageCreate){}, s.handlers...)
	s.mu.Unlock()

	for _, h := range handlers {
		h(nil, m)
	}
}

func (s *E2EDiscordSession) Open() error  { return nil }
func (s *E2EDiscordSession) Close() error { return nil }

type E2EEngine struct {
	mu           sync.Mutex
	activeCount  int32
	maxActive    int32
	tmpImagePath string
}

func (e *E2EEngine) Generate(ctx context.Context, prompt string, opts services.GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error) {
	cur := atomic.AddInt32(&e.activeCount, 1)
	defer atomic.AddInt32(&e.activeCount, -1)

	for {
		max := atomic.LoadInt32(&e.maxActive)
		if cur > max {
			if atomic.CompareAndSwapInt32(&e.maxActive, max, cur) {
				break
			}
		} else {
			break
		}
	}

	time.Sleep(30 * time.Millisecond) // Simulate rendering duration

	return &domain.ImageSpec{
		EnhancedPrompt: prompt,
		AspectRatio:    domain.RatioLandscape,
		Backend:        "pollinations",
		Model:          "flux",
		Width:          1344,
		Height:         768,
		Seed:           4242,
	}, &domain.ImageResult{
		LocalPath: e.tmpImagePath,
	}, nil
}

func (e *E2EEngine) ExecuteSubagent(ctx context.Context, subagent, prompt string) (string, error) {
	return "Mock response from @" + subagent + " for " + prompt, nil
}

func (e *E2EEngine) PipelineGenerate(ctx context.Context, prompt string, opts services.PipelineOptions) (*services.PipelineResult, error) {
	return &services.PipelineResult{
		DirectorConcept: "E2E concept",
		PromptSmithSpec: &domain.ImageSpec{EnhancedPrompt: prompt, AspectRatio: "1:1", Backend: "pollinations"},
		ImageResult:     &domain.ImageResult{LocalPath: e.tmpImagePath},
		CriticScore:     0.90,
		Duration:        50 * time.Millisecond,
	}, nil
}

func (e *E2EEngine) SearchMemory(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error) {
	return []domain.KnowledgeFact{{Topic: "e2e", Concept: "test", Fact: "integration test fact"}}, nil
}

func (e *E2EEngine) ListSubagents(ctx context.Context) ([]domain.SubagentDef, error) {
	return []domain.SubagentDef{{Name: "director", DisplayName: "Director", Role: "Scene Composition"}}, nil
}

func (e *E2EEngine) ListBackends() []string     { return []string{"pollinations"} }
func (e *E2EEngine) GetDefaultBackend() string { return "pollinations" }
func (e *E2EEngine) Status(ctx context.Context) (gateway.GatewayStatus, error) {
	return gateway.GatewayStatus{
		Uptime:         time.Minute,
		DefaultBackend: "pollinations",
		DefaultModel:   "flux",
		LLMProvider:    "passthrough",
	}, nil
}

func TestGateway_E2E_FullRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	imgPath := filepath.Join(tmpDir, "e2e_render.jpg")
	_ = os.WriteFile(imgPath, []byte("fake-e2e-image-bytes"), 0644)

	tgBot := NewE2ETelegramBot()
	dcSession := NewE2EDiscordSession()
	engine := &E2EEngine{tmpImagePath: imgPath}

	concurrency := 2
	maxQueue := 5
	queue := gateway.NewJobQueue(concurrency, maxQueue)

	tgConfig := config.TelegramConfig{
		Enabled:        true,
		BotToken:       "mock-tg-token",
		AllowedChatIDs: []int64{1001},
		AllowedUserIDs: []int64{2001},
	}
	tgAdapter := telegram.NewAdapter(tgConfig, engine, queue)
	tgAdapter.SetBot(tgBot)

	dcConfig := config.DiscordConfig{
		Enabled:           true,
		BotToken:          "mock-dc-token",
		AllowedChannelIDs: []string{"chan-100"},
		AllowedUserIDs:    []string{"user-200"},
	}
	dcAdapter := discord.NewAdapter(dcConfig, engine, queue)
	dcAdapter.SetSession(dcSession)

	mux := gateway.NewMultiplexer([]gateway.GatewayAdapter{tgAdapter, dcAdapter}, queue)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mux.Start(ctx); err != nil {
		t.Fatalf("Mux Start failed: %v", err)
	}

	// 1. Send /gen request from Telegram
	tgBot.updatesCh <- tgbotapi.Update{
		UpdateID: 1,
		Message: &tgbotapi.Message{
			MessageID: 10,
			Chat:      &tgbotapi.Chat{ID: 1001},
			From:      &tgbotapi.User{ID: 2001},
			Text:      "/gen Cyberpunk rain alley --ratio 16:9",
		},
	}

	// 2. Send /gen request from Discord
	dcSession.EmitMessage(&discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-dc-1",
			ChannelID: "chan-100",
			Author:    &discordgo.User{ID: "user-200", Bot: false},
			Content:   "/gen Neon samurai in bamboo forest --ratio 16:9",
		},
	})

	// 3. Send informational /status and /subagents commands
	tgBot.updatesCh <- tgbotapi.Update{
		UpdateID: 2,
		Message: &tgbotapi.Message{
			MessageID: 11,
			Chat:      &tgbotapi.Chat{ID: 1001},
			From:      &tgbotapi.User{ID: 2001},
			Text:      "/status",
		},
	}

	dcSession.EmitMessage(&discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "msg-dc-2",
			ChannelID: "chan-100",
			Author:    &discordgo.User{ID: "user-200", Bot: false},
			Content:   "/subagents",
		},
	})

	// Wait for queue processing and deliveries
	time.Sleep(300 * time.Millisecond)

	// Verify Telegram deliveries
	tgBot.mu.Lock()
	tgSent := append([]tgbotapi.Chattable{}, tgBot.SentMessages...)
	tgBot.mu.Unlock()

	var tgPhotoDelivered bool
	var tgStatusReplied bool
	for _, m := range tgSent {
		if photo, ok := m.(tgbotapi.PhotoConfig); ok {
			if strings.Contains(photo.Caption, "Cyberpunk rain alley") {
				tgPhotoDelivered = true
			}
		}
		if msg, ok := m.(tgbotapi.MessageConfig); ok {
			if strings.Contains(msg.Text, "Engine Status") {
				tgStatusReplied = true
			}
		}
	}

	if !tgPhotoDelivered {
		t.Errorf("expected Telegram photo delivery for /gen")
	}
	if !tgStatusReplied {
		t.Errorf("expected Telegram text reply for /status")
	}

	// Verify Discord deliveries
	dcSession.mu.Lock()
	dcComplex := append([]*discordgo.MessageSend{}, dcSession.SentComplex...)
	dcTexts := append([]string{}, dcSession.SentTexts...)
	dcSession.mu.Unlock()

	var dcEmbedDelivered bool
	for _, m := range dcComplex {
		if m.Embed != nil && (strings.Contains(m.Embed.Title, "Neon samurai") || strings.Contains(m.Embed.Description, "Neon samurai")) {
			dcEmbedDelivered = true
		}
	}

	var dcSubagentsReplied bool
	for _, txt := range dcTexts {
		if strings.Contains(txt, "@director") {
			dcSubagentsReplied = true
		}
	}

	if !dcEmbedDelivered {
		t.Errorf("expected Discord embed and file attachment delivery for /gen")
	}
	if !dcSubagentsReplied {
		t.Errorf("expected Discord text reply for /subagents")
	}

	// Verify concurrency limit was respected
	if engine.maxActive > int32(concurrency) {
		t.Errorf("observed concurrency %d exceeded limit %d", engine.maxActive, concurrency)
	}

	// 4. Test graceful shutdown
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()

	if err := mux.Stop(stopCtx); err != nil {
		t.Fatalf("Mux Stop failed: %v", err)
	}
}
