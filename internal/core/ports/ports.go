package ports

import (
	"context"

	"aris/internal/core/domain"
)

// LLMProvider generates reasoning, prompt architecture, and parameter selection.
type LLMProvider interface {
	Name() string
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	ReasonPrompt(ctx context.Context, input string, facts []domain.KnowledgeFact) (*domain.ImageSpec, error)
}

// ImageBackend handles actual image rendering.
type ImageBackend interface {
	Name() string
	SupportsModels() []string
	Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error)
}

// BackendRegistry manages available image generation providers.
type BackendRegistry interface {
	Register(backend ImageBackend) error
	Get(name string) (ImageBackend, error)
	List() []string
	SetDefault(name string) error
	GetDefault() ImageBackend
}

// VisionCritic evaluates generated images against quality and prompt constraints.
type VisionCritic interface {
	Evaluate(ctx context.Context, imagePath string, spec *domain.ImageSpec) (score float64, critique string, err error)
}

// KnowledgeGraphStore handles persistent 3-scope memory and full-text search.
type KnowledgeGraphStore interface {
	AddFact(ctx context.Context, fact domain.KnowledgeFact) (string, error)
	SearchFacts(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error)
	GetFactsByTopic(ctx context.Context, topic string) ([]domain.KnowledgeFact, error)
	ListAllFacts(ctx context.Context, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error)
	DeleteFact(ctx context.Context, id string) error
}

// HistoryStore records generation runs, specs, outputs, and ratings.
type HistoryStore interface {
	SaveGeneration(ctx context.Context, spec *domain.ImageSpec, result *domain.ImageResult) error
	UpdateRating(ctx context.Context, id string, rating int, feedback string) error
	GetHistory(ctx context.Context, limit, offset int) ([]domain.GenerationRecord, error)
	GetByID(ctx context.Context, id string) (*domain.GenerationRecord, error)
}

// SubagentStore manages persistent subagent definitions in SQLite.
type SubagentStore interface {
	SaveSubagent(ctx context.Context, def domain.SubagentDef) error
	GetSubagent(ctx context.Context, name string) (*domain.SubagentDef, error)
	ListSubagents(ctx context.Context) ([]domain.SubagentDef, error)
	DeleteSubagent(ctx context.Context, name string) error
}
