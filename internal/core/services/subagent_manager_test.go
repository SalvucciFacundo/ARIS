package services_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"aris/internal/adapters/image"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

// MockSubagentStore implements ports.SubagentStore in memory.
type MockSubagentStore struct {
	items map[string]domain.SubagentDef
}

func NewMockSubagentStore() *MockSubagentStore {
	store := &MockSubagentStore{
		items: make(map[string]domain.SubagentDef),
	}
	for _, sub := range domain.DefaultSubagents() {
		store.items[sub.Name] = sub
	}
	return store
}

func (s *MockSubagentStore) SaveSubagent(ctx context.Context, def domain.SubagentDef) error {
	s.items[def.Name] = def
	return nil
}

func (s *MockSubagentStore) GetSubagent(ctx context.Context, name string) (*domain.SubagentDef, error) {
	def, ok := s.items[name]
	if !ok {
		return nil, fmt.Errorf("subagent @%s not found", name)
	}
	return &def, nil
}

func (s *MockSubagentStore) ListSubagents(ctx context.Context) ([]domain.SubagentDef, error) {
	list := make([]domain.SubagentDef, 0, len(s.items))
	for _, def := range s.items {
		list = append(list, def)
	}
	return list, nil
}

func (s *MockSubagentStore) DeleteSubagent(ctx context.Context, name string) error {
	delete(s.items, name)
	return nil
}

// MockLLMProvider implements ports.LLMProvider.
type MockLLMProvider struct {
	completeFunc func(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	reasonFunc   func(ctx context.Context, input string, facts []domain.KnowledgeFact) (*domain.ImageSpec, error)
}

func (m *MockLLMProvider) Name() string {
	return "mock-llm"
}

func (m *MockLLMProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	if m.completeFunc != nil {
		return m.completeFunc(ctx, systemPrompt, userPrompt)
	}
	return fmt.Sprintf("Response to %q with system: %s", userPrompt, systemPrompt[:min(20, len(systemPrompt))]), nil
}

func (m *MockLLMProvider) ReasonPrompt(ctx context.Context, input string, facts []domain.KnowledgeFact) (*domain.ImageSpec, error) {
	if m.reasonFunc != nil {
		return m.reasonFunc(ctx, input, facts)
	}
	return &domain.ImageSpec{
		ID:             "mock-spec-id",
		RawPrompt:      input,
		EnhancedPrompt: input + ", 8k resolution, cinematic lighting",
		AspectRatio:    domain.RatioLandscape,
		Width:          1344,
		Height:         768,
		Steps:          20,
		CFGScale:       7.0,
		Backend:        "pollinations",
		Model:          "flux",
		CreatedAt:      time.Now(),
	}, nil
}

// MockBackend implements ports.ImageBackend.
type MockBackend struct {
	name         string
	generateFunc func(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error)
}

func (m *MockBackend) Name() string {
	return m.name
}

func (m *MockBackend) SupportsModels() []string {
	return []string{"flux", "sdxl"}
}

func (m *MockBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, spec)
	}
	return &domain.ImageResult{
		ID:          "mock-res-id",
		SpecID:      spec.ID,
		LocalPath:   "/tmp/aris_test.png",
		Format:      "png",
		SizeInBytes: 1024,
		Duration:    100 * time.Millisecond,
	}, nil
}

// MockVisionCritic implements ports.VisionCritic.
type MockVisionCritic struct {
	score    float64
	critique string
	err      error
}

func (m *MockVisionCritic) Evaluate(ctx context.Context, imagePath string, spec *domain.ImageSpec) (float64, string, error) {
	return m.score, m.critique, m.err
}

// MockKnowledgeGraph implements ports.KnowledgeGraphStore.
type MockKnowledgeGraph struct {
	facts []domain.KnowledgeFact
}

func (m *MockKnowledgeGraph) AddFact(ctx context.Context, fact domain.KnowledgeFact) (string, error) {
	m.facts = append(m.facts, fact)
	return "fact-123", nil
}

func (m *MockKnowledgeGraph) SearchFacts(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error) {
	return m.facts, nil
}

func (m *MockKnowledgeGraph) GetFactsByTopic(ctx context.Context, topic string) ([]domain.KnowledgeFact, error) {
	return m.facts, nil
}

func (m *MockKnowledgeGraph) ListAllFacts(ctx context.Context, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error) {
	return m.facts, nil
}

func (m *MockKnowledgeGraph) DeleteFact(ctx context.Context, id string) error {
	return nil
}

func TestParseSubagentRoute(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantName      string
		wantRemaining string
		wantFound     bool
	}{
		{
			name:          "direct mention with space",
			input:         "@director a neon cyberpunk cat in neo tokyo",
			wantName:      "director",
			wantRemaining: "a neon cyberpunk cat in neo tokyo",
			wantFound:     true,
		},
		{
			name:          "mention with colon",
			input:         "@promptsmith: photorealistic portrait 85mm",
			wantName:      "promptsmith",
			wantRemaining: "photorealistic portrait 85mm",
			wantFound:     true,
		},
		{
			name:          "mention with comma",
			input:         "@critic, evaluate this image",
			wantName:      "critic",
			wantRemaining: "evaluate this image",
			wantFound:     true,
		},
		{
			name:          "uppercase name normalized",
			input:         "@CURATOR style catalog",
			wantName:      "curator",
			wantRemaining: "style catalog",
			wantFound:     true,
		},
		{
			name:          "no mention prefix",
			input:         "generate a cat in space",
			wantName:      "",
			wantRemaining: "generate a cat in space",
			wantFound:     false,
		},
		{
			name:          "mention in middle of text",
			input:         "hello @director make it cinematic",
			wantName:      "",
			wantRemaining: "hello @director make it cinematic",
			wantFound:     false,
		},
		{
			name:          "bare @ character",
			input:         "@",
			wantName:      "",
			wantRemaining: "@",
			wantFound:     false,
		},
		{
			name:          "mention only without prompt",
			input:         "@enhancer",
			wantName:      "enhancer",
			wantRemaining: "",
			wantFound:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, remaining, found := services.ParseSubagentRoute(tt.input)
			if found != tt.wantFound {
				t.Errorf("ParseSubagentRoute(%q) found = %v, want %v", tt.input, found, tt.wantFound)
			}
			if name != tt.wantName {
				t.Errorf("ParseSubagentRoute(%q) name = %q, want %q", tt.input, name, tt.wantName)
			}
			if remaining != tt.wantRemaining {
				t.Errorf("ParseSubagentRoute(%q) remaining = %q, want %q", tt.input, remaining, tt.wantRemaining)
			}
		})
	}
}

func TestSubagentManager_ExecuteDirect(t *testing.T) {
	ctx := context.Background()
	store := NewMockSubagentStore()
	llmProvider := &MockLLMProvider{
		completeFunc: func(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
			return fmt.Sprintf("Subagent answer for: %s", userPrompt), nil
		},
	}
	reg := image.NewRegistry()
	mgr := services.NewSubagentManager(store, llmProvider, reg, nil, nil)

	t.Run("execute director successfully", func(t *testing.T) {
		resp, err := mgr.ExecuteDirect(ctx, "director", "neon cyberpunk alley")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "Subagent answer for: neon cyberpunk alley"
		if resp != expected {
			t.Errorf("got %q, want %q", resp, expected)
		}
	})

	t.Run("execute with leading @", func(t *testing.T) {
		resp, err := mgr.ExecuteDirect(ctx, "@promptsmith", "convert to flux")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := "Subagent answer for: convert to flux"
		if resp != expected {
			t.Errorf("got %q, want %q", resp, expected)
		}
	})

	t.Run("non-existent subagent returns error", func(t *testing.T) {
		_, err := mgr.ExecuteDirect(ctx, "unknown_agent", "hello")
		if err == nil {
			t.Fatal("expected error for unknown subagent, got nil")
		}
	})

	t.Run("empty input prompt returns error", func(t *testing.T) {
		_, err := mgr.ExecuteDirect(ctx, "director", "   ")
		if err == nil {
			t.Fatal("expected error for empty prompt, got nil")
		}
	})

	t.Run("empty subagent name returns error", func(t *testing.T) {
		_, err := mgr.ExecuteDirect(ctx, "   ", "hello")
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})
}

func TestSubagentManager_PipelineExecute(t *testing.T) {
	ctx := context.Background()
	store := NewMockSubagentStore()
	llmProvider := &MockLLMProvider{}
	reg := image.NewRegistry()
	mockBackend := &MockBackend{name: "pollinations"}
	_ = reg.Register(mockBackend)
	_ = reg.SetDefault("pollinations")
	critic := &MockVisionCritic{
		score:    0.85,
		critique: "Excellent lighting and composition.",
	}
	kg := &MockKnowledgeGraph{}

	mgr := services.NewSubagentManager(store, llmProvider, reg, critic, kg)

	t.Run("successful pipeline execution", func(t *testing.T) {
		opts := services.PipelineOptions{
			AspectRatio:  domain.RatioLandscape,
			Model:        "flux",
			EnableCritic: true,
		}

		result, err := mgr.PipelineExecute(ctx, "cyberpunk samurai in rain", opts)
		if err != nil {
			t.Fatalf("pipeline execution failed: %v", err)
		}

		if result.DirectorConcept == "" {
			t.Error("expected director concept to be populated")
		}
		if result.PromptSmithSpec == nil {
			t.Fatal("expected promptsmith spec to be populated")
		}
		if result.PromptSmithSpec.AspectRatio != domain.RatioLandscape {
			t.Errorf("got aspect ratio %s, want %s", result.PromptSmithSpec.AspectRatio, domain.RatioLandscape)
		}
		if result.ImageResult == nil {
			t.Fatal("expected image result to be populated")
		}
		if result.CriticScore != 0.85 {
			t.Errorf("got critic score %.2f, want 0.85", result.CriticScore)
		}
		if result.EnhancerAdvice == "" {
			t.Error("expected enhancer advice to be populated")
		}
		if len(result.CuratorSavedFacts) == 0 {
			t.Error("expected curator saved facts to be populated")
		}
	})

	t.Run("empty prompt returns error", func(t *testing.T) {
		_, err := mgr.PipelineExecute(ctx, "   ", services.PipelineOptions{})
		if err == nil {
			t.Fatal("expected error for empty prompt, got nil")
		}
	})

	t.Run("backend failure propagation", func(t *testing.T) {
		failBackend := &MockBackend{
			name: "fail-backend",
			generateFunc: func(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
				return nil, fmt.Errorf("network timeout")
			},
		}
		failReg := image.NewRegistry()
		_ = failReg.Register(failBackend)
		_ = failReg.SetDefault("fail-backend")

		failMgr := services.NewSubagentManager(store, llmProvider, failReg, critic, kg)
		_, err := failMgr.PipelineExecute(ctx, "valid prompt", services.PipelineOptions{})
		if err == nil {
			t.Fatal("expected error when backend fails, got nil")
		}
	})
}

func TestSubagentManager_CRUD(t *testing.T) {
	ctx := context.Background()
	store := NewMockSubagentStore()
	llmProvider := &MockLLMProvider{}
	mgr := services.NewSubagentManager(store, llmProvider, nil, nil, nil)

	t.Run("list subagents", func(t *testing.T) {
		list, err := mgr.ListSubagents(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(list) < 5 {
			t.Errorf("expected at least 5 default subagents, got %d", len(list))
		}
	})

	t.Run("get existing subagent", func(t *testing.T) {
		sub, err := mgr.GetSubagent(ctx, "director")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sub.Name != "director" {
			t.Errorf("got name %q, want %q", sub.Name, "director")
		}
	})

	t.Run("save and delete custom subagent", func(t *testing.T) {
		custom := domain.SubagentDef{
			Name:         "colorist",
			DisplayName:  "Color Grading Specialist",
			Role:         "Color Harmonies & LUTs",
			SystemPrompt: "You are @colorist.",
			Temperature:  0.3,
		}

		err := mgr.SaveSubagent(ctx, custom)
		if err != nil {
			t.Fatalf("failed to save subagent: %v", err)
		}

		saved, err := mgr.GetSubagent(ctx, "colorist")
		if err != nil {
			t.Fatalf("failed to get saved subagent: %v", err)
		}
		if saved.DisplayName != "Color Grading Specialist" {
			t.Errorf("got display name %q, want %q", saved.DisplayName, "Color Grading Specialist")
		}

		err = mgr.DeleteSubagent(ctx, "colorist")
		if err != nil {
			t.Fatalf("failed to delete subagent: %v", err)
		}

		_, err = mgr.GetSubagent(ctx, "colorist")
		if err == nil {
			t.Fatal("expected error getting deleted subagent, got nil")
		}
	})
}
