package gateway

import (
	"context"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/services"
)

// GatewayAdapter represents a single messaging platform adapter (Telegram, Discord).
type GatewayAdapter interface {
	// Start begins listening for incoming messages/events.
	Start(ctx context.Context) error
	// Stop gracefully shuts down the adapter.
	Stop(ctx context.Context) error
	// Name returns the adapter's identifier (e.g. "telegram", "discord").
	Name() string
}

// GatewayMultiplexer manages multiple GatewayAdapters concurrently.
type GatewayMultiplexer interface {
	// Start starts all registered adapters concurrently.
	Start(ctx context.Context) error
	// Stop gracefully stops all adapters.
	Stop(ctx context.Context) error
}

// GatewayStatus contains engine status information for the /status command.
type GatewayStatus struct {
	Uptime         time.Duration `json:"uptime"`
	DefaultBackend string        `json:"default_backend"`
	DefaultModel   string        `json:"default_model"`
	LLMProvider    string        `json:"llm_provider"`
	LLMModel       string        `json:"llm_model"`
	CriticEnabled  bool          `json:"critic_enabled"`
	QueuePending   int           `json:"queue_pending"`
	QueueCapacity  int           `json:"queue_capacity"`
}

// GatewayEngine abstracts the core ARIS AgentService for the gateway adapters.
type GatewayEngine interface {
	Generate(ctx context.Context, prompt string, opts services.GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error)
	ExecuteSubagent(ctx context.Context, subagent, prompt string) (string, error)
	PipelineGenerate(ctx context.Context, prompt string, opts services.PipelineOptions) (*services.PipelineResult, error)
	SearchMemory(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error)
	ListSubagents(ctx context.Context) ([]domain.SubagentDef, error)
	ListBackends() []string
	GetDefaultBackend() string
	Status(ctx context.Context) (GatewayStatus, error)
}
