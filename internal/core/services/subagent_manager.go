package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
)

// SubagentManager manages specialized autonomous subagents and coordinates multi-agent pipelines.
type SubagentManager struct {
	store    ports.SubagentStore
	llm      ports.LLMProvider
	registry ports.BackendRegistry
	critic   ports.VisionCritic
	kg       ports.KnowledgeGraphStore
}

// NewSubagentManager creates a new subagent manager.
func NewSubagentManager(
	store ports.SubagentStore,
	llm ports.LLMProvider,
	registry ports.BackendRegistry,
	critic ports.VisionCritic,
	kg ports.KnowledgeGraphStore,
) *SubagentManager {
	return &SubagentManager{
		store:    store,
		llm:      llm,
		registry: registry,
		critic:   critic,
		kg:       kg,
	}
}

// ParseSubagentRoute detects if a user input starts with @<name> and extracts the subagent name and clean prompt.
func ParseSubagentRoute(input string) (name string, remaining string, found bool) {
	trimmed := strings.TrimSpace(input)
	if !strings.HasPrefix(trimmed, "@") {
		return "", input, false
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return "", input, false
	}

	rawTarget := strings.TrimPrefix(parts[0], "@")
	rawTarget = strings.TrimRight(rawTarget, ":,-")
	if rawTarget == "" {
		return "", input, false
	}

	remaining = strings.TrimSpace(trimmed[len(parts[0]):])
	return strings.ToLower(rawTarget), remaining, true
}

// ExecuteDirect executes a prompt under the specialized system prompt and temperature of a specific subagent.
func (m *SubagentManager) ExecuteDirect(ctx context.Context, subagentName, input string) (string, error) {
	cleanName := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(subagentName), "@"))
	if cleanName == "" {
		return "", fmt.Errorf("subagent name cannot be empty")
	}
	if strings.TrimSpace(input) == "" {
		return "", fmt.Errorf("input prompt cannot be empty")
	}
	if m.store == nil {
		return "", fmt.Errorf("subagent store not configured")
	}
	if m.llm == nil {
		return "", fmt.Errorf("llm provider not configured")
	}

	def, err := m.store.GetSubagent(ctx, cleanName)
	if err != nil {
		return "", fmt.Errorf("get subagent @%s: %w", cleanName, err)
	}

	resp, err := m.llm.Complete(ctx, def.SystemPrompt, input)
	if err != nil {
		return "", fmt.Errorf("execute subagent @%s: %w", cleanName, err)
	}

	return resp, nil
}

// PipelineOptions defines configuration overrides for a multi-agent pipeline run.
type PipelineOptions struct {
	AspectRatio     domain.AspectRatio
	Model           string
	Backend         string
	Seed            int64
	NegativePrompt  string
	InputImage      string
	MaskImage       string
	DenoiseStrength float64
	Mode            domain.ReferenceMode
	ScaleFactor     int
	RestoreFaces    bool
	FaceFidelity    float64
	UpscalerModel   string
	EnableCritic    bool
}

// PipelineResult encapsulates the outputs of all stages in a multi-agent sequence.
type PipelineResult struct {
	DirectorConcept   string                 `json:"director_concept"`
	PromptSmithSpec   *domain.ImageSpec      `json:"promptsmith_spec"`
	ImageResult       *domain.ImageResult    `json:"image_result"`
	CriticScore       float64                `json:"critic_score"`
	CriticCritique    string                 `json:"critic_critique"`
	EnhancerAdvice    string                 `json:"enhancer_advice"`
	UpscalerAdvice    string                 `json:"upscaler_advice,omitempty"`
	CuratorSavedFacts []domain.KnowledgeFact `json:"curator_saved_facts,omitempty"`
	Duration          time.Duration          `json:"duration"`
}

// PipelineExecute coordinates the full multi-agent sequence:
// 1. Orchestrator -> @director: Conceptualizes scene, lighting, camera.
// 2. @director -> @promptsmith: Compiles into technical ImageSpec.
// 3. @promptsmith -> ImageBackend: Synthesizes image.
// 4. ImageBackend -> @critic: Inspects rendered output.
// 5. @critic -> @enhancer: Recommends post-processing / upscaling.
// 6. Orchestrator -> @curator: Saves successful recipes to Knowledge Graph.
func (m *SubagentManager) PipelineExecute(ctx context.Context, prompt string, opts PipelineOptions) (*PipelineResult, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}
	if m.llm == nil {
		return nil, fmt.Errorf("llm provider not configured")
	}
	if m.registry == nil {
		return nil, fmt.Errorf("backend registry not configured")
	}

	startTime := time.Now()
	res := &PipelineResult{}

	// 1. @director: Conceptualize scene & art direction
	directorConcept, err := m.ExecuteDirect(ctx, "director", prompt)
	if err != nil {
		// Fallback to raw prompt if direct subagent execution fails
		directorConcept = prompt
	}
	res.DirectorConcept = directorConcept

	// 2. @promptsmith: Compile technical ImageSpec
	var recalledFacts []domain.KnowledgeFact
	if m.kg != nil {
		facts, err := m.kg.SearchFacts(ctx, prompt, "", 5)
		if err == nil {
			recalledFacts = facts
		}
	}

	spec, err := m.llm.ReasonPrompt(ctx, directorConcept, recalledFacts)
	if err != nil {
		return nil, fmt.Errorf("promptsmith compilation failed: %w", err)
	}

	// Apply user overrides
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
	if opts.ScaleFactor > 0 {
		spec.ScaleFactor = opts.ScaleFactor
	}
	if opts.RestoreFaces {
		spec.RestoreFaces = true
	}
	if opts.FaceFidelity > 0.0 {
		spec.FaceFidelity = opts.FaceFidelity
	}
	if opts.UpscalerModel != "" {
		spec.UpscalerModel = opts.UpscalerModel
	}
	spec.ApplyDefaults()

	res.PromptSmithSpec = spec

	// 3. ImageBackend: Synthesize image
	var targetBackend ports.ImageBackend
	if spec.Backend != "" {
		b, err := m.registry.Get(spec.Backend)
		if err == nil {
			targetBackend = b
		}
	}
	if targetBackend == nil {
		targetBackend = m.registry.GetDefault()
	}
	if targetBackend == nil {
		return res, fmt.Errorf("no image backend available")
	}

	imgResult, err := targetBackend.Generate(ctx, spec)
	if err != nil {
		return res, fmt.Errorf("backend %s generation failed: %w", targetBackend.Name(), err)
	}
	res.ImageResult = imgResult

	// 4. @critic: Inspect rendered output
	if m.critic != nil && (opts.EnableCritic || true) {
		score, critique, cErr := m.critic.Evaluate(ctx, imgResult.LocalPath, spec)
		if cErr == nil {
			res.CriticScore = score
			res.CriticCritique = critique
		}
	}

	// 5. @upscaler / @enhancer: Recommend post-processing / upscaling
	enhancerPrompt := fmt.Sprintf("Image rendered at %dx%d with backend %s. Critic score: %.2f, notes: %s. Recommend post-processing and upscaling settings.",
		spec.Width, spec.Height, spec.Backend, res.CriticScore, res.CriticCritique)
	upscalerAdvice, uErr := m.ExecuteDirect(ctx, "upscaler", enhancerPrompt)
	if uErr == nil {
		res.UpscalerAdvice = upscalerAdvice
		res.EnhancerAdvice = upscalerAdvice
	} else {
		enhancerAdvice, eErr := m.ExecuteDirect(ctx, "enhancer", enhancerPrompt)
		if eErr == nil {
			res.EnhancerAdvice = enhancerAdvice
			res.UpscalerAdvice = enhancerAdvice
		}
	}

	// 6. @curator: Save successful recipe to Knowledge Graph
	if m.kg != nil && res.CriticScore >= 0.5 {
		fact := domain.KnowledgeFact{
			Topic:       "recipe:pipeline",
			Concept:     "generation_recipe",
			Fact:        fmt.Sprintf("Prompt: %s -> Enhanced: %s (Ratio: %s, Backend: %s)", prompt, spec.EnhancedPrompt, spec.AspectRatio, spec.Backend),
			SourceAgent: "curator",
			Labels:      []string{"pipeline", "recipe", string(spec.AspectRatio)},
			Scope:       domain.ScopeStyle,
			CreatedAt:   time.Now(),
		}
		if _, err := m.kg.AddFact(ctx, fact); err == nil {
			res.CuratorSavedFacts = append(res.CuratorSavedFacts, fact)
		}
	}

	res.Duration = time.Since(startTime)
	return res, nil
}

// GetSubagent retrieves a subagent definition by name.
func (m *SubagentManager) GetSubagent(ctx context.Context, name string) (*domain.SubagentDef, error) {
	if m.store == nil {
		return nil, fmt.Errorf("subagent store not configured")
	}
	cleanName := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "@"))
	return m.store.GetSubagent(ctx, cleanName)
}

// ListSubagents returns all registered subagents.
func (m *SubagentManager) ListSubagents(ctx context.Context) ([]domain.SubagentDef, error) {
	if m.store == nil {
		return nil, fmt.Errorf("subagent store not configured")
	}
	return m.store.ListSubagents(ctx)
}

// SaveSubagent saves or updates a subagent definition.
func (m *SubagentManager) SaveSubagent(ctx context.Context, def domain.SubagentDef) error {
	if m.store == nil {
		return fmt.Errorf("subagent store not configured")
	}
	return m.store.SaveSubagent(ctx, def)
}

// DeleteSubagent removes a subagent definition by name.
func (m *SubagentManager) DeleteSubagent(ctx context.Context, name string) error {
	if m.store == nil {
		return fmt.Errorf("subagent store not configured")
	}
	cleanName := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "@"))
	return m.store.DeleteSubagent(ctx, cleanName)
}
