package discord

import (
	"context"
	"fmt"
	"log"
	"sync"

	"aris/internal/adapters/gateway"
	"aris/internal/config"
	"github.com/bwmarrin/discordgo"
)

// Adapter implements the gateway.GatewayAdapter interface for Discord.
type Adapter struct {
	cfg     config.DiscordConfig
	engine  gateway.GatewayEngine
	queue   *gateway.JobQueue
	session Session
	handler *Handler
	cancel  context.CancelFunc
	mu      sync.Mutex
	running bool
}

// NewAdapter creates a new Discord Gateway Adapter.
func NewAdapter(cfg config.DiscordConfig, engine gateway.GatewayEngine, queue *gateway.JobQueue) *Adapter {
	return &Adapter{
		cfg:    cfg,
		engine: engine,
		queue:  queue,
	}
}

// SetSession allows injecting a custom or mock Session.
func (a *Adapter) SetSession(session Session) {
	a.session = session
}

// Name returns the adapter's platform identifier.
func (a *Adapter) Name() string {
	return "discord"
}

// Start establishes the Discord gateway connection and registers event handlers.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.running {
		return nil
	}

	if a.session == nil {
		if a.cfg.BotToken == "" {
			return fmt.Errorf("discord bot token is required")
		}
		dg, err := discordgo.New("Bot " + a.cfg.BotToken)
		if err != nil {
			return fmt.Errorf("init discord session: %w", err)
		}
		a.session = dg
	}

	auth := NewAuthorizer(a.cfg.AllowedChannelIDs, a.cfg.AllowedUserIDs)
	a.handler = NewHandler(a.session, a.engine, a.queue, auth)

	listenCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel

	a.session.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		a.handler.HandleMessage(listenCtx, m)
	})

	if err := a.session.Open(); err != nil {
		return fmt.Errorf("open discord session: %w", err)
	}

	a.running = true
	log.Printf("🤖 Discord Gateway adapter started successfully.")
	return nil
}

// Stop gracefully closes the Discord gateway session.
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.running {
		return nil
	}

	a.running = false
	if a.cancel != nil {
		a.cancel()
	}

	if a.session != nil {
		_ = a.session.Close()
	}

	log.Printf("🛑 Discord Gateway adapter stopped.")
	return nil
}
