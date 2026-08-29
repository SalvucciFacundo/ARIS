package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
)

// AgentService is the core orchestrator of ARIS.
type AgentService struct {
	llm      ports.LLMProvider
	registry ports.BackendRegistry
	kg       ports.KnowledgeGraphStore
	history  ports.HistoryStore
	critic   ports.VisionCritic
	criticSvc *CriticService
	learner   *AutoLearner
	subagents *SubagentManager
}

// NewAgentService creates a new ARIS agent service.
func NewAgentService(
	llm ports.LLMProvider,
	registry ports.BackendRegistry,
	kg ports.KnowledgeGraphStore,
	history ports.HistoryStore,
	critic ports.VisionCritic,
) *AgentService {
	var learner *AutoLearner
	if kg != nil {
		learner = NewAutoLearner(kg, llm)
	}
	var criticSvc *CriticService
	if critic != nil {
		criticSvc = NewCriticService(critic, 0.60, true)
	}
	return &AgentService{
		llm:       llm,
		registry:  registry,
		kg:        kg,
		history:   history,
		critic:    critic,
		criticSvc: criticSvc,
		learner:   learner,
	}
}

// GenerateOptions contains overrides for a single generation request.
type GenerateOptions struct {
	AspectRatio     domain.AspectRatio
	Model           string
	Backend         string
	Seed            int64
	NegativePrompt  string
	InputImage      string
	MaskImage       string
	DenoiseStrength float64
	Mode            domain.ReferenceMode
	Project         string
	EnableCritic    bool
	AutoHeal        bool
}

// Generate runs the autonomous lifecycle: Recall -> Reason -> Synthesize -> Persist.
func (s *AgentService) Generate(ctx context.Context, input string, opts GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error) {
	if strings.TrimSpace(input) == "" {
		return nil, nil, fmt.Errorf("prompt cannot be empty")
	}

	// 1. RECALL: Query Knowledge Graph for relevant facts and styles
	var recalledFacts []domain.KnowledgeFact
	if s.kg != nil {
		facts, err := s.kg.SearchFacts(ctx, input, "", 5)
		if err == nil {
			recalledFacts = facts
		}
	}

	// 2. REASON: LLM / Heuristic acts as Art Director & Prompt Architect
	spec, err := s.llm.ReasonPrompt(ctx, input, recalledFacts)
	if err != nil {
		return nil, nil, fmt.Errorf("reasoning failed: %w", err)
	}

	// Apply CLI/User overrides
	if opts.AspectRatio != "" {
		spec.AspectRatio = opts.AspectRatio
		w, h := opts.AspectRatio.Dimensions(1024)
		spec.Width = w
		spec.Height = h
	}
	if opts.Model != "" {
		spec.Model = opts.Model
	}
	if opts.Backend != "" {
		spec.Backend = opts.Backend
	}
	if opts.Seed > 0 {
		spec.Seed = opts.Seed
	}
	if opts.NegativePrompt != "" {
		if spec.NegativePrompt != "" {
			spec.NegativePrompt = spec.NegativePrompt + ", " + opts.NegativePrompt
		} else {
			spec.NegativePrompt = opts.NegativePrompt
		}
	}
	if opts.Mode != "" {
		spec.Mode = opts.Mode
	}
	if opts.InputImage != "" {
		spec.InputImagePath = opts.InputImage
	}
	if opts.MaskImage != "" {
		spec.MaskImagePath = opts.MaskImage
	}
	if opts.DenoiseStrength > 0.0 {
		spec.DenoiseStrength = opts.DenoiseStrength
	}
	spec.ApplyDefaults()

	// 3. DISPATCH & RENDER: Resolve Image Backend from Registry
	var targetBackend ports.ImageBackend
	if opts.Backend != "" {
		b, err := s.registry.Get(opts.Backend)
		if err != nil {
			return spec, nil, fmt.Errorf("resolve backend: %w", err)
		}
		targetBackend = b
	} else if spec.Backend != "" {
		b, err := s.registry.Get(spec.Backend)
		if err == nil {
			targetBackend = b
		}
	}

	if targetBackend == nil {
		targetBackend = s.registry.GetDefault()
	}
	if targetBackend == nil {
		return spec, nil, fmt.Errorf("no image backend available in registry")
	}

	spec.Backend = targetBackend.Name()

	result, err := targetBackend.Generate(ctx, spec)
	if err != nil {
		return spec, nil, fmt.Errorf("backend %s generation failed: %w", targetBackend.Name(), err)
	}

	// 4. (Optional) VLM CRITIC & SELF-HEALING LOOP
	if s.criticSvc != nil && (opts.EnableCritic || s.critic != nil) {
		healedSpec, healedResult, err := s.criticSvc.InspectAndHeal(ctx, targetBackend, spec, result)
		if err == nil {
			spec = healedSpec
			result = healedResult
		}
	}

	// 5. PERSIST: Save to SQLite History
	if s.history != nil {
		_ = s.history.SaveGeneration(ctx, spec, result)
	}

	// 6. AUTO-LEARN: Autonomous memory reflection loop (Hermes / GAIA style)
	if s.learner != nil {
		go func() {
			bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_, _ = s.learner.ReflectTurn(bgCtx, ReflectionTurn{
				RawInput:       input,
				EnhancedPrompt: spec.EnhancedPrompt,
				NegativePrompt: spec.NegativePrompt,
			})
		}()
	}

	return spec, result, nil
}

// SetSubagents attaches a subagent manager to the agent service.
func (s *AgentService) SetSubagents(sm *SubagentManager) {
	s.subagents = sm
}

// Subagents returns the configured subagent manager.
func (s *AgentService) Subagents() *SubagentManager {
	return s.subagents
}

// ExecuteSubagent runs direct subagent reasoning under isolated context.
func (s *AgentService) ExecuteSubagent(ctx context.Context, subagentName, input string) (string, error) {
	if s.subagents == nil {
		return "", fmt.Errorf("subagent manager not configured")
	}
	return s.subagents.ExecuteDirect(ctx, subagentName, input)
}

// PipelineGenerate executes the full multi-agent generation pipeline.
func (s *AgentService) PipelineGenerate(ctx context.Context, prompt string, opts PipelineOptions) (*PipelineResult, error) {
	if s.subagents == nil {
		return nil, fmt.Errorf("subagent manager not configured")
	}
	return s.subagents.PipelineExecute(ctx, prompt, opts)
}

// Registry returns the backend registry.
func (s *AgentService) Registry() ports.BackendRegistry {
	return s.registry
}

// LearnFact saves a newly discovered style or user preference into the Knowledge Graph.
func (s *AgentService) LearnFact(ctx context.Context, topic, concept, fact string, scope domain.MemoryScope, labels []string) (string, error) {
	if s.kg == nil {
		return "", fmt.Errorf("knowledge graph not initialized")
	}

	kf := domain.KnowledgeFact{
		Topic:       topic,
		Concept:     concept,
		Fact:        fact,
		SourceAgent: "user",
		Labels:      labels,
		Scope:       scope,
		CreatedAt:   time.Now(),
	}

	return s.kg.AddFact(ctx, kf)
}

// SearchMemory retrieves facts from the Knowledge Graph.
func (s *AgentService) SearchMemory(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error) {
	if s.kg == nil {
		return nil, fmt.Errorf("knowledge graph not initialized")
	}
	return s.kg.SearchFacts(ctx, query, scope, limit)
}

// GetHistory retrieves past generation history.
func (s *AgentService) GetHistory(ctx context.Context, limit, offset int) ([]domain.GenerationRecord, error) {
	if s.history == nil {
		return nil, fmt.Errorf("history store not initialized")
	}
	return s.history.GetHistory(ctx, limit, offset)
}
