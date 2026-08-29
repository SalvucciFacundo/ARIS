package discord

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aris/internal/adapters/gateway"
	"aris/internal/core/domain"
	"aris/internal/core/services"
	"github.com/bwmarrin/discordgo"
)

// Session abstracts the Discord client API for message delivery and event registration.
type Session interface {
	ChannelMessageSend(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSendComplex(channelID string, data *discordgo.MessageSend, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelTyping(channelID string, options ...discordgo.RequestOption) error
	AddHandler(handler interface{}) func()
	Open() error
	Close() error
}

// Handler handles incoming Discord message events and orchestrates generation jobs.
type Handler struct {
	session Session
	engine  gateway.GatewayEngine
	queue   *gateway.JobQueue
	auth    *Authorizer
}

// NewHandler creates a new Discord message event handler.
func NewHandler(session Session, engine gateway.GatewayEngine, queue *gateway.JobQueue, auth *Authorizer) *Handler {
	return &Handler{
		session: session,
		engine:  engine,
		queue:   queue,
		auth:    auth,
	}
}

// HandleMessage handles an incoming MessageCreate event from Discord.
func (h *Handler) HandleMessage(ctx context.Context, m *discordgo.MessageCreate) {
	if m == nil || m.Message == nil || m.Author == nil {
		return
	}

	if !h.auth.IsAuthorized(m) {
		return
	}

	parsed := gateway.ParseMessage(m.Content)

	switch parsed.Type {
	case gateway.CmdHelp:
		h.sendText(m.ChannelID, FormatHelp())
	case gateway.CmdSubagents:
		subagents, err := h.engine.ListSubagents(ctx)
		if err != nil {
			h.sendText(m.ChannelID, fmt.Sprintf("❌ Error listing subagents: %v", err))
			return
		}
		h.sendText(m.ChannelID, FormatSubagents(subagents))
	case gateway.CmdBackends:
		backends := h.engine.ListBackends()
		def := h.engine.GetDefaultBackend()
		h.sendText(m.ChannelID, FormatBackends(backends, def))
	case gateway.CmdStatus:
		status, err := h.engine.Status(ctx)
		if err != nil {
			h.sendText(m.ChannelID, fmt.Sprintf("❌ Error getting status: %v", err))
			return
		}
		h.sendText(m.ChannelID, FormatStatus(status))
	case gateway.CmdMemory:
		facts, err := h.engine.SearchMemory(ctx, parsed.CleanPrompt, "", 10)
		if err != nil {
			h.sendText(m.ChannelID, fmt.Sprintf("❌ Memory search failed: %v", err))
			return
		}
		h.sendText(m.ChannelID, FormatMemory(facts, parsed.CleanPrompt))
	case gateway.CmdSubagent:
		h.handleSubagent(ctx, m.ChannelID, parsed)
	case gateway.CmdPipeline:
		h.handlePipeline(ctx, m.ChannelID, parsed)
	case gateway.CmdGen:
		h.handleGen(ctx, m.ChannelID, parsed)
	default:
		h.sendText(m.ChannelID, FormatHelp())
	}
}

func (h *Handler) handleSubagent(ctx context.Context, channelID string, parsed gateway.ParsedMessage) {
	subName := parsed.Options.SubagentName
	if subName == "" {
		subName = "director"
	}

	_ = h.session.ChannelTyping(channelID)

	resp, err := h.engine.ExecuteSubagent(ctx, subName, parsed.CleanPrompt)
	if err != nil {
		h.sendText(channelID, fmt.Sprintf("❌ Subagent @%s failed: %v", subName, err))
		return
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🤖 @%s Response", subName),
		Description: resp,
		Color:       0x00FFCC, // Cyberpunk Cyan
	}

	_, _ = h.session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embed: embed,
	})
}

func (h *Handler) handleGen(ctx context.Context, channelID string, parsed gateway.ParsedMessage) {
	if parsed.CleanPrompt == "" {
		h.sendText(channelID, "❌ Error: Prompt cannot be empty. Usage: `/gen <prompt> [--ratio 16:9]`")
		return
	}

	job := gateway.Job{
		Ctx: ctx,
		Task: func(taskCtx context.Context) error {
			ticker := time.NewTicker(4 * time.Second)
			defer ticker.Stop()

			_ = h.session.ChannelTyping(channelID)
			stopTyping := make(chan struct{})
			go func() {
				for {
					select {
					case <-stopTyping:
						return
					case <-ticker.C:
						_ = h.session.ChannelTyping(channelID)
					}
				}
			}()

			start := time.Now()
			spec, res, err := h.engine.Generate(taskCtx, parsed.CleanPrompt, parsed.Options.GenOpts)
			close(stopTyping)

			if err != nil {
				h.sendText(channelID, fmt.Sprintf("❌ Generation failed: %v", err))
				return err
			}

			return h.deliverImage(channelID, spec, res, time.Since(start))
		},
		ReplyErr: func(err error) {
			h.sendText(channelID, fmt.Sprintf("❌ Request error: %v", err))
		},
	}

	if err := h.queue.Submit(job); err != nil {
		if err == gateway.ErrQueueFull {
			h.sendText(channelID, fmt.Sprintf("⚠️ Generation queue is full (%d pending). Please try again shortly.", h.queue.Pending()))
		} else {
			h.sendText(channelID, fmt.Sprintf("❌ Failed to enqueue generation: %v", err))
		}
	}
}

func (h *Handler) handlePipeline(ctx context.Context, channelID string, parsed gateway.ParsedMessage) {
	if parsed.CleanPrompt == "" {
		h.sendText(channelID, "❌ Error: Prompt cannot be empty for /pipeline.")
		return
	}

	job := gateway.Job{
		Ctx: ctx,
		Task: func(taskCtx context.Context) error {
			ticker := time.NewTicker(4 * time.Second)
			defer ticker.Stop()

			_ = h.session.ChannelTyping(channelID)
			stopTyping := make(chan struct{})
			go func() {
				for {
					select {
					case <-stopTyping:
						return
					case <-ticker.C:
						_ = h.session.ChannelTyping(channelID)
					}
				}
			}()

			start := time.Now()
			pipeRes, err := h.engine.PipelineGenerate(taskCtx, parsed.CleanPrompt, parsed.Options.PipeOpts)
			close(stopTyping)

			if err != nil {
				h.sendText(channelID, fmt.Sprintf("❌ Multi-agent pipeline failed: %v", err))
				return err
			}

			return h.deliverPipelineResult(channelID, pipeRes, time.Since(start))
		},
		ReplyErr: func(err error) {
			h.sendText(channelID, fmt.Sprintf("❌ Pipeline request error: %v", err))
		},
	}

	if err := h.queue.Submit(job); err != nil {
		if err == gateway.ErrQueueFull {
			h.sendText(channelID, fmt.Sprintf("⚠️ Generation queue is full (%d pending). Please try again shortly.", h.queue.Pending()))
		} else {
			h.sendText(channelID, fmt.Sprintf("❌ Failed to enqueue pipeline: %v", err))
		}
	}
}

func (h *Handler) deliverImage(channelID string, spec *domain.ImageSpec, res *domain.ImageResult, duration time.Duration) error {
	file, err := os.Open(res.LocalPath)
	if err != nil {
		h.sendText(channelID, fmt.Sprintf("❌ Failed to read generated image: %v", err))
		return err
	}
	defer file.Close()

	fileName := filepath.Base(res.LocalPath)
	if fileName == "" {
		fileName = "aris_output.jpg"
	}

	embed := BuildImageEmbed(spec, res, fileName, duration)

	msgSend := &discordgo.MessageSend{
		Embed: embed,
		Files: []*discordgo.File{
			{
				Name:        fileName,
				ContentType: "image/jpeg",
				Reader:      file,
			},
		},
	}

	_, err = h.session.ChannelMessageSendComplex(channelID, msgSend)
	return err
}

func (h *Handler) deliverPipelineResult(channelID string, res *services.PipelineResult, duration time.Duration) error {
	if res.ImageResult == nil || res.ImageResult.LocalPath == "" {
		h.sendText(channelID, "✨ Multi-agent pipeline completed without image output.")
		return nil
	}

	file, err := os.Open(res.ImageResult.LocalPath)
	if err != nil {
		h.sendText(channelID, fmt.Sprintf("❌ Failed to read generated pipeline image: %v", err))
		return err
	}
	defer file.Close()

	fileName := filepath.Base(res.ImageResult.LocalPath)
	if fileName == "" {
		fileName = "pipeline_output.jpg"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✨ Multi-Agent Synthesis Completed",
		Description: fmt.Sprintf("**Concept:** %s\n**Prompt:** %s", res.DirectorConcept, res.PromptSmithSpec.EnhancedPrompt),
		Color:       0xFF007F, // Cyberpunk Magenta
		Image: &discordgo.MessageEmbedImage{
			URL: fmt.Sprintf("attachment://%s", fileName),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📐 Dimensions",
				Value:  fmt.Sprintf("%dx%d (%s)", res.PromptSmithSpec.Width, res.PromptSmithSpec.Height, res.PromptSmithSpec.AspectRatio),
				Inline: true,
			},
			{
				Name:   "🔌 Backend",
				Value:  res.PromptSmithSpec.Backend,
				Inline: true,
			},
			{
				Name:   "⏱️ Duration",
				Value:  fmt.Sprintf("%v", duration.Round(time.Millisecond)),
				Inline: true,
			},
		},
	}

	if res.CriticScore > 0 {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "👁️ Critic Score",
			Value:  fmt.Sprintf("%.2f", res.CriticScore),
			Inline: true,
		})
	}
	if res.EnhancerAdvice != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "💡 Enhancer Advice",
			Value:  res.EnhancerAdvice,
			Inline: false,
		})
	}

	msgSend := &discordgo.MessageSend{
		Embed: embed,
		Files: []*discordgo.File{
			{
				Name:        fileName,
				ContentType: "image/jpeg",
				Reader:      file,
			},
		},
	}

	_, err = h.session.ChannelMessageSendComplex(channelID, msgSend)
	return err
}

func (h *Handler) sendText(channelID, text string) {
	_, _ = h.session.ChannelMessageSend(channelID, text)
}

// BuildImageEmbed constructs a Discord Rich Embed with image attachment reference and metadata fields.
func BuildImageEmbed(spec *domain.ImageSpec, res *domain.ImageResult, fileName string, duration time.Duration) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "🎨 ARIS Image Synthesized",
		Description: spec.EnhancedPrompt,
		Color:       0x00FFCC,
		Image: &discordgo.MessageEmbedImage{
			URL: fmt.Sprintf("attachment://%s", fileName),
		},
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📐 Dimensions",
				Value:  fmt.Sprintf("%dx%d (%s)", spec.Width, spec.Height, spec.AspectRatio),
				Inline: true,
			},
			{
				Name:   "🔌 Backend",
				Value:  spec.Backend,
				Inline: true,
			},
			{
				Name:   "🌱 Seed",
				Value:  fmt.Sprintf("%d", spec.Seed),
				Inline: true,
			},
			{
				Name:   "⏱️ Duration",
				Value:  fmt.Sprintf("%v", duration.Round(time.Millisecond)),
				Inline: true,
			},
		},
	}

	if spec.Model != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🤖 Model",
			Value:  spec.Model,
			Inline: true,
		})
	}

	if res != nil && res.Metadata != nil {
		if score, ok := res.Metadata["critic_score"]; ok {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "👁️ Critic Score",
				Value:  fmt.Sprintf("%.2f", score),
				Inline: true,
			})
		}
	}

	return embed
}

// FormatSubagents formats subagents list for Discord.
func FormatSubagents(subagents []domain.SubagentDef) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🤖 **ARIS Specialized Subagents (%d active):**\n\n", len(subagents)))
	for i, sub := range subagents {
		sb.WriteString(fmt.Sprintf("**%d. @%s** — %s (%s)\n> %s\n\n", i+1, sub.Name, sub.DisplayName, sub.Role, sub.Description))
	}
	sb.WriteString("💡 *Trigger directly using:* `@director cinematic neo tokyo street`")
	return sb.String()
}

// FormatBackends formats backends list for Discord.
func FormatBackends(backends []string, defaultBackend string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔌 **Registered Image Backends (%d available):**\n\n", len(backends)))
	for i, b := range backends {
		marker := "  "
		if b == defaultBackend {
			marker = "⭐ **[DEFAULT]** "
		}
		sb.WriteString(fmt.Sprintf("%d. %s`%s`\n", i+1, marker, b))
	}
	return sb.String()
}

// FormatMemory formats knowledge graph facts for Discord.
func FormatMemory(facts []domain.KnowledgeFact, query string) string {
	if len(facts) == 0 {
		return fmt.Sprintf("🧠 No memory facts found matching `%s`.", query)
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🧠 **Knowledge Graph Facts for `%s` (%d items):**\n\n", query, len(facts)))
	for _, f := range facts {
		sb.WriteString(fmt.Sprintf("• `[%s]` **%s** ➔ %s: %s\n", f.Scope, f.Topic, f.Concept, f.Fact))
	}
	return sb.String()
}

// FormatStatus formats gateway runtime status for Discord.
func FormatStatus(s gateway.GatewayStatus) string {
	return fmt.Sprintf(`⚡ **ARIS Gateway Engine Status**

⏱️ **Uptime:** %v
🔌 **Default Backend:** %s
🤖 **Default Model:** %s
🧠 **LLM Provider:** %s (%s)
👁️ **Critic Enabled:** %v
📊 **Queue Buffer:** %d / %d pending
`, s.Uptime.Round(time.Second), s.DefaultBackend, s.DefaultModel, s.LLMProvider, s.LLMModel, s.CriticEnabled, s.QueuePending, s.QueueCapacity)
}

// FormatHelp returns user help for Discord.
func FormatHelp() string {
	return `🎨 **ARIS Discord Gateway**

**Commands:**
• ` + "`/gen <prompt> [flags]`" + ` — Generate an image
• ` + "`/pipeline <prompt>`" + ` — Run multi-agent director pipeline
• ` + "`/subagents`" + ` — List active visual subagents
• ` + "`/backends`" + ` — List supported image backends
• ` + "`/memory <query>`" + ` — Search Knowledge Graph facts
• ` + "`/status`" + ` — View engine & queue status
• ` + "`/help`" + ` — Show this help guide

**Flags for ` + "`/gen`" + `:**
` + "`--ratio 16:9`" + ` (or ` + "`-r`" + `) — 1:1, 16:9, 9:16, 4:3, 21:9
` + "`--backend <name>`" + ` — pollinations, comfyui, falai, etc.
` + "`--model <name>`" + ` — flux, flux-realism, etc.
` + "`--seed <num>`" + ` — Seed for reproducibility
` + "`--critic`" + ` — Run VLM visual critique

**Subagent Routing:**
` + "`@director futuristic neon alley in tokyo`" + `
`
}
