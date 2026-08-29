package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aris/internal/adapters/gateway"
	"aris/internal/core/domain"
	"aris/internal/core/services"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// BotAPI abstracts Telegram Bot API client operations.
type BotAPI interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	GetUpdatesChan(config tgbotapi.UpdateConfig) tgbotapi.UpdatesChannel
	StopReceivingUpdates()
}

// Handler coordinates Telegram update processing, auth checking, and job queuing.
type Handler struct {
	bot            BotAPI
	engine         gateway.GatewayEngine
	queue          *gateway.JobQueue
	auth           *Authorizer
	sendAsDocument bool
}

// NewHandler creates a new Telegram message handler.
func NewHandler(
	bot BotAPI,
	engine gateway.GatewayEngine,
	queue *gateway.JobQueue,
	auth *Authorizer,
	sendAsDocument bool,
) *Handler {
	return &Handler{
		bot:            bot,
		engine:         engine,
		queue:          queue,
		auth:           auth,
		sendAsDocument: sendAsDocument,
	}
}

// HandleUpdate handles an incoming update.
func (h *Handler) HandleUpdate(ctx context.Context, update *tgbotapi.Update) {
	if update == nil || update.Message == nil {
		return
	}

	chatID := update.Message.Chat.ID
	msgID := update.Message.MessageID

	if !h.auth.IsAuthorized(update) {
		reply := tgbotapi.NewMessage(chatID, "⛔ Access Denied: You are not authorized to use this ARIS Gateway.")
		reply.ReplyToMessageID = msgID
		_, _ = h.bot.Send(reply)
		return
	}

	parsed := gateway.ParseMessage(update.Message.Text)

	switch parsed.Type {
	case gateway.CmdHelp:
		h.sendReply(chatID, msgID, FormatHelp())
	case gateway.CmdSubagents:
		subagents, err := h.engine.ListSubagents(ctx)
		if err != nil {
			h.sendReply(chatID, msgID, fmt.Sprintf("❌ Error listing subagents: %v", err))
			return
		}
		h.sendReply(chatID, msgID, FormatSubagents(subagents))
	case gateway.CmdBackends:
		backends := h.engine.ListBackends()
		def := h.engine.GetDefaultBackend()
		h.sendReply(chatID, msgID, FormatBackends(backends, def))
	case gateway.CmdStatus:
		status, err := h.engine.Status(ctx)
		if err != nil {
			h.sendReply(chatID, msgID, fmt.Sprintf("❌ Error getting status: %v", err))
			return
		}
		h.sendReply(chatID, msgID, FormatStatus(status))
	case gateway.CmdMemory:
		facts, err := h.engine.SearchMemory(ctx, parsed.CleanPrompt, "", 10)
		if err != nil {
			h.sendReply(chatID, msgID, fmt.Sprintf("❌ Memory search failed: %v", err))
			return
		}
		h.sendReply(chatID, msgID, FormatMemory(facts, parsed.CleanPrompt))
	case gateway.CmdSubagent:
		h.handleSubagent(ctx, chatID, msgID, parsed)
	case gateway.CmdPipeline:
		h.handlePipeline(ctx, chatID, msgID, parsed)
	case gateway.CmdGen:
		h.handleGen(ctx, chatID, msgID, parsed)
	default:
		h.sendReply(chatID, msgID, FormatHelp())
	}
}

func (h *Handler) handleSubagent(ctx context.Context, chatID int64, msgID int, parsed gateway.ParsedMessage) {
	subName := parsed.Options.SubagentName
	if subName == "" {
		subName = "director"
	}

	h.sendAction(chatID, "typing")

	resp, err := h.engine.ExecuteSubagent(ctx, subName, parsed.CleanPrompt)
	if err != nil {
		h.sendReply(chatID, msgID, fmt.Sprintf("❌ Subagent @%s failed: %v", subName, err))
		return
	}

	reply := fmt.Sprintf("🤖 **@%s Response:**\n\n%s", subName, resp)
	h.sendReply(chatID, msgID, reply)
}

func (h *Handler) handleGen(ctx context.Context, chatID int64, msgID int, parsed gateway.ParsedMessage) {
	if parsed.CleanPrompt == "" {
		h.sendReply(chatID, msgID, "❌ Error: Prompt cannot be empty. Example: `/gen a cyberpunk city --ratio 16:9`")
		return
	}

	// Submit generation job to queue
	job := gateway.Job{
		Ctx: ctx,
		Task: func(taskCtx context.Context) error {
			ticker := time.NewTicker(4 * time.Second)
			defer ticker.Stop()

			h.sendAction(chatID, "upload_photo")
			stopTyping := make(chan struct{})
			go func() {
				for {
					select {
					case <-stopTyping:
						return
					case <-ticker.C:
						h.sendAction(chatID, "upload_photo")
					}
				}
			}()

			start := time.Now()
			spec, res, err := h.engine.Generate(taskCtx, parsed.CleanPrompt, parsed.Options.GenOpts)
			close(stopTyping)

			if err != nil {
				h.sendReply(chatID, msgID, fmt.Sprintf("❌ Generation failed: %v", err))
				return err
			}

			caption := FormatCaption(spec, res, time.Since(start))
			asDoc := h.sendAsDocument || parsed.Options.SendAsDocument

			return h.deliverImage(chatID, msgID, res.LocalPath, caption, asDoc)
		},
		ReplyErr: func(err error) {
			h.sendReply(chatID, msgID, fmt.Sprintf("❌ Request error: %v", err))
		},
	}

	if err := h.queue.Submit(job); err != nil {
		if err == gateway.ErrQueueFull {
			h.sendReply(chatID, msgID, fmt.Sprintf("⚠️ Queue is currently full (%d pending). Please try again in a moment.", h.queue.Pending()))
		} else {
			h.sendReply(chatID, msgID, fmt.Sprintf("❌ Failed to queue generation: %v", err))
		}
	}
}

func (h *Handler) handlePipeline(ctx context.Context, chatID int64, msgID int, parsed gateway.ParsedMessage) {
	if parsed.CleanPrompt == "" {
		h.sendReply(chatID, msgID, "❌ Error: Prompt cannot be empty for /pipeline.")
		return
	}

	job := gateway.Job{
		Ctx: ctx,
		Task: func(taskCtx context.Context) error {
			ticker := time.NewTicker(4 * time.Second)
			defer ticker.Stop()

			h.sendAction(chatID, "upload_photo")
			stopTyping := make(chan struct{})
			go func() {
				for {
					select {
					case <-stopTyping:
						return
					case <-ticker.C:
						h.sendAction(chatID, "upload_photo")
					}
				}
			}()

			start := time.Now()
			pipeRes, err := h.engine.PipelineGenerate(taskCtx, parsed.CleanPrompt, parsed.Options.PipeOpts)
			close(stopTyping)

			if err != nil {
				h.sendReply(chatID, msgID, fmt.Sprintf("❌ Multi-agent pipeline failed: %v", err))
				return err
			}

			caption := FormatPipelineCaption(pipeRes, time.Since(start))
			asDoc := h.sendAsDocument || parsed.Options.SendAsDocument

			if pipeRes.ImageResult != nil && pipeRes.ImageResult.LocalPath != "" {
				return h.deliverImage(chatID, msgID, pipeRes.ImageResult.LocalPath, caption, asDoc)
			}

			h.sendReply(chatID, msgID, caption)
			return nil
		},
		ReplyErr: func(err error) {
			h.sendReply(chatID, msgID, fmt.Sprintf("❌ Pipeline request error: %v", err))
		},
	}

	if err := h.queue.Submit(job); err != nil {
		if err == gateway.ErrQueueFull {
			h.sendReply(chatID, msgID, fmt.Sprintf("⚠️ Queue is currently full (%d pending). Please try again in a moment.", h.queue.Pending()))
		} else {
			h.sendReply(chatID, msgID, fmt.Sprintf("❌ Failed to queue pipeline: %v", err))
		}
	}
}

func (h *Handler) deliverImage(chatID int64, replyToID int, filePath string, caption string, asDoc bool) error {
	if asDoc {
		doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
		doc.Caption = caption
		doc.ReplyToMessageID = replyToID
		_, err := h.bot.Send(doc)
		return err
	}

	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FilePath(filePath))
	photo.Caption = caption
	photo.ReplyToMessageID = replyToID
	_, err := h.bot.Send(photo)
	return err
}

func (h *Handler) sendReply(chatID int64, replyToID int, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyToMessageID = replyToID
	_, _ = h.bot.Send(msg)
}

func (h *Handler) sendAction(chatID int64, action string) {
	act := tgbotapi.NewChatAction(chatID, action)
	_, _ = h.bot.Send(act)
}

// FormatCaption builds user-friendly metadata text for generated image deliveries.
func FormatCaption(spec *domain.ImageSpec, res *domain.ImageResult, duration time.Duration) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🎨 %s\n\n", spec.EnhancedPrompt))
	sb.WriteString(fmt.Sprintf("📐 %dx%d (Ratio: %s)\n", spec.Width, spec.Height, spec.AspectRatio))
	sb.WriteString(fmt.Sprintf("🔌 %s", spec.Backend))
	if spec.Model != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", spec.Model))
	}
	sb.WriteString(fmt.Sprintf(" | 🌱 %d\n", spec.Seed))
	sb.WriteString(fmt.Sprintf("⏱️ %v", duration.Round(time.Millisecond)))

	if res != nil && res.Metadata != nil {
		if score, ok := res.Metadata["critic_score"]; ok {
			sb.WriteString(fmt.Sprintf(" | 👁️ Critic: %.2f", score))
		}
	}
	return sb.String()
}

// FormatPipelineCaption builds formatted summary for multi-agent pipeline results.
func FormatPipelineCaption(res *services.PipelineResult, duration time.Duration) string {
	var sb strings.Builder
	sb.WriteString("✨ Multi-Agent Synthesis Completed\n\n")
	if res.DirectorConcept != "" {
		sb.WriteString(fmt.Sprintf("🎬 Director: %s\n\n", res.DirectorConcept))
	}
	if res.PromptSmithSpec != nil {
		sb.WriteString(fmt.Sprintf("📐 %dx%d (Ratio: %s) | 🔌 %s\n",
			res.PromptSmithSpec.Width, res.PromptSmithSpec.Height, res.PromptSmithSpec.AspectRatio, res.PromptSmithSpec.Backend))
	}
	if res.CriticScore > 0 {
		sb.WriteString(fmt.Sprintf("👁️ Critic Score: %.2f\n", res.CriticScore))
	}
	if res.EnhancerAdvice != "" {
		sb.WriteString(fmt.Sprintf("💡 Enhancer: %s\n", res.EnhancerAdvice))
	}
	sb.WriteString(fmt.Sprintf("⏱️ Duration: %v", duration.Round(time.Millisecond)))
	return sb.String()
}

// FormatSubagents formats the list of registered subagents.
func FormatSubagents(subagents []domain.SubagentDef) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🤖 ARIS Specialized Subagents (%d active):\n\n", len(subagents)))
	for i, sub := range subagents {
		sb.WriteString(fmt.Sprintf("%d. @%s — %s (%s)\n   %s\n\n", i+1, sub.Name, sub.DisplayName, sub.Role, sub.Description))
	}
	sb.WriteString("💡 Run directly using: `@director <your idea>`")
	return sb.String()
}

// FormatBackends formats the list of registered image backends.
func FormatBackends(backends []string, defaultBackend string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔌 Registered Image Backends (%d available):\n\n", len(backends)))
	for i, b := range backends {
		marker := "  "
		if b == defaultBackend {
			marker = "⭐ [DEFAULT] "
		}
		sb.WriteString(fmt.Sprintf("%d. %s%s\n", i+1, marker, b))
	}
	return sb.String()
}

// FormatMemory formats retrieved Knowledge Graph facts.
func FormatMemory(facts []domain.KnowledgeFact, query string) string {
	if len(facts) == 0 {
		return fmt.Sprintf("🧠 No memory facts found matching %q.", query)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🧠 Knowledge Graph Facts for %q (%d items):\n\n", query, len(facts)))
	for _, f := range facts {
		sb.WriteString(fmt.Sprintf("• [%s] %s -> %s: %s\n", f.Scope, f.Topic, f.Concept, f.Fact))
	}
	return sb.String()
}

// FormatStatus formats gateway runtime engine status.
func FormatStatus(s gateway.GatewayStatus) string {
	return fmt.Sprintf(`⚡ ARIS Gateway Engine Status

⏱️ Uptime:           %v
🔌 Default Backend:  %s
🤖 Default Model:    %s
🧠 LLM Provider:     %s (%s)
👁️ Critic Enabled:   %v
📊 Queue Buffer:     %d / %d pending
`, s.Uptime.Round(time.Second), s.DefaultBackend, s.DefaultModel, s.LLMProvider, s.LLMModel, s.CriticEnabled, s.QueuePending, s.QueueCapacity)
}

// FormatHelp returns the help text for Telegram users.
func FormatHelp() string {
	return `🎨 ARIS Telegram Gateway

Commands:
/gen <prompt> [flags]  - Generate an image
/pipeline <prompt>     - Full multi-agent synthesis
/subagents             - List available visual subagents
/backends              - List supported image backends
/memory <query>        - Search Knowledge Graph
/status                - View engine & queue status
/help                  - Show this help guide

Flags for /gen:
--ratio 16:9 (or -r)   - Aspect ratio: 1:1, 16:9, 9:16, 4:3, 21:9
--backend <name>       - pollinations, comfyui, falai, etc.
--model <name>         - flux, flux-realism, turbo, etc.
--seed <num>           - Seed for reproducibility
--negative <words>     - Negative prompt
--doc                  - Deliver uncompressed document
--critic               - Run VLM visual evaluation

Subagent trigger:
@director cinematic shot of a samurai in bamboo forest
`
}
