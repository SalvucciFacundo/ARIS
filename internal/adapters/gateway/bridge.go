package gateway

import (
	"context"
	"time"

	"aris/internal/config"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

// EngineBridge adapts the core ARIS AgentService to the GatewayEngine interface.
type EngineBridge struct {
	agent     *services.AgentService
	cfg       *config.Config
	queue     *JobQueue
	startTime time.Time
}

// NewEngineBridge creates a new GatewayEngine bridge wrapping AgentService.
func NewEngineBridge(agent *services.AgentService, cfg *config.Config, queue *JobQueue) *EngineBridge {
	return &EngineBridge{
		agent:     agent,
		cfg:       cfg,
		queue:     queue,
		startTime: time.Now(),
	}
}

// Generate delegates to AgentService.Generate.
func (b *EngineBridge) Generate(ctx context.Context, prompt string, opts services.GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error) {
	return b.agent.Generate(ctx, prompt, opts)
}

// ExecuteSubagent delegates to AgentService.ExecuteSubagent.
func (b *EngineBridge) ExecuteSubagent(ctx context.Context, subagent, prompt string) (string, error) {
	return b.agent.ExecuteSubagent(ctx, subagent, prompt)
}

// PipelineGenerate delegates to AgentService.PipelineGenerate.
func (b *EngineBridge) PipelineGenerate(ctx context.Context, prompt string, opts services.PipelineOptions) (*services.PipelineResult, error) {
	return b.agent.PipelineGenerate(ctx, prompt, opts)
}

// SearchMemory delegates to AgentService.SearchMemory.
func (b *EngineBridge) SearchMemory(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error) {
	return b.agent.SearchMemory(ctx, query, scope, limit)
}

// ListSubagents retrieves active subagent definitions.
func (b *EngineBridge) ListSubagents(ctx context.Context) ([]domain.SubagentDef, error) {
	sm := b.agent.Subagents()
	if sm == nil {
		return nil, nil
	}
	return sm.ListSubagents(ctx)
}

// ListBackends lists registered backend names.
func (b *EngineBridge) ListBackends() []string {
	reg := b.agent.Registry()
	if reg == nil {
		return nil
	}
	return reg.List()
}

// GetDefaultBackend returns the name of the active default backend.
func (b *EngineBridge) GetDefaultBackend() string {
	reg := b.agent.Registry()
	if reg == nil {
		return ""
	}
	def := reg.GetDefault()
	if def == nil {
		return ""
	}
	return def.Name()
}

// Status returns runtime metrics and engine status.
func (b *EngineBridge) Status(ctx context.Context) (GatewayStatus, error) {
	pending := 0
	cap := 0
	if b.queue != nil {
		pending = b.queue.Pending()
		cap = b.queue.Capacity()
	}

	defBackend := b.GetDefaultBackend()
	defModel := ""
	if b.cfg != nil {
		defModel = b.cfg.Image.DefaultModel
	}

	llmProvider := ""
	llmModel := ""
	criticEnabled := false
	if b.cfg != nil {
		llmProvider = b.cfg.LLM.Provider
		llmModel = b.cfg.LLM.Model
		criticEnabled = b.cfg.Critic.Enabled
	}

	return GatewayStatus{
		Uptime:         time.Since(b.startTime),
		DefaultBackend: defBackend,
		DefaultModel:   defModel,
		LLMProvider:    llmProvider,
		LLMModel:       llmModel,
		CriticEnabled:  criticEnabled,
		QueuePending:   pending,
		QueueCapacity:  cap,
	}, nil
}
