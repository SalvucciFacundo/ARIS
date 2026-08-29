package telegram

import (
	"context"
	"fmt"
	"log"
	"sync"

	"aris/internal/adapters/gateway"
	"aris/internal/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Adapter implements the gateway.GatewayAdapter interface for Telegram.
type Adapter struct {
	cfg     config.TelegramConfig
	engine  gateway.GatewayEngine
	queue   *gateway.JobQueue
	bot     BotAPI
	handler *Handler
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	mu      sync.Mutex
	running bool
}

// NewAdapter creates a new Telegram Gateway Adapter.
func NewAdapter(cfg config.TelegramConfig, engine gateway.GatewayEngine, queue *gateway.JobQueue) *Adapter {
	return &Adapter{
		cfg:    cfg,
		engine: engine,
		queue:  queue,
	}
}

// SetBot allows injecting a custom/mock BotAPI instance.
func (a *Adapter) SetBot(bot BotAPI) {
	a.bot = bot
}

// Name returns the identifier for this gateway adapter.
func (a *Adapter) Name() string {
	return "telegram"
}

// Start begins long-polling updates from Telegram.
func (a *Adapter) Start(ctx context.Context) error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return nil
	}

	if a.bot == nil {
		if a.cfg.BotToken == "" {
			a.mu.Unlock()
			return fmt.Errorf("telegram bot token is required")
		}
		botAPI, err := tgbotapi.NewBotAPI(a.cfg.BotToken)
		if err != nil {
			a.mu.Unlock()
			return fmt.Errorf("init telegram bot: %w", err)
		}
		a.bot = botAPI
	}

	auth := NewAuthorizer(a.cfg.AllowedChatIDs, a.cfg.AllowedUserIDs)
	a.handler = NewHandler(a.bot, a.engine, a.queue, auth, a.cfg.SendAsDocument)

	pollCtx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	a.running = true
	a.mu.Unlock()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30
	updates := a.bot.GetUpdatesChan(u)

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		log.Printf("🤖 Telegram Gateway adapter started successfully.")
		for {
			select {
			case <-pollCtx.Done():
				return
			case update, ok := <-updates:
				if !ok {
					return
				}
				a.handler.HandleUpdate(pollCtx, &update)
			}
		}
	}()

	return nil
}

// Stop halts the Telegram polling loop.
func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = false
	if a.cancel != nil {
		a.cancel()
	}
	if a.bot != nil {
		a.bot.StopReceivingUpdates()
	}
	a.mu.Unlock()

	a.wg.Wait()
	log.Printf("🛑 Telegram Gateway adapter stopped.")
	return nil
}
