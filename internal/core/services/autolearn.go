package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"

	"github.com/google/uuid"
)

// ReflectionTurn contains contextual signals from a generation turn.
type ReflectionTurn struct {
	RawInput       string
	PreviousPrompt string
	EnhancedPrompt string
	NegativePrompt string
	UserFeedback   string
}

// AutoLearner analyzes user exchanges and synthesizes persistent KnowledgeGraph facts.
type AutoLearner struct {
	kg  ports.KnowledgeGraphStore
	llm ports.LLMProvider
}

// NewAutoLearner creates an autonomous learning engine.
func NewAutoLearner(kg ports.KnowledgeGraphStore, llm ports.LLMProvider) *AutoLearner {
	return &AutoLearner{
		kg:  kg,
		llm: llm,
	}
}

type extractedFactJSON struct {
	Topic   string `json:"topic"`
	Concept string `json:"concept"`
	Fact    string `json:"fact"`
	Scope   string `json:"scope"` // "user", "style", "project"
}

// ReflectTurn examines the turn and extracts aesthetic rules, negative triggers, or style facts.
func (a *AutoLearner) ReflectTurn(ctx context.Context, turn ReflectionTurn) ([]domain.KnowledgeFact, error) {
	if a.kg == nil {
		return nil, nil
	}

	var newFacts []domain.KnowledgeFact

	// 1. Heuristic Pattern Detection (Fast, offline, zero-token cost)
	heuristicFacts := a.extractHeuristicFacts(turn)
	newFacts = append(newFacts, heuristicFacts...)

	// 2. LLM Distillation if LLM is active and not passthrough
	if a.llm != nil && a.llm.Name() != "passthrough-heuristic" {
		llmFacts, err := a.extractLLMFacts(ctx, turn)
		if err == nil && len(llmFacts) > 0 {
			newFacts = append(newFacts, llmFacts...)
		}
	}

	// 3. Deduplicate and Save to Knowledge Graph
	var savedFacts []domain.KnowledgeFact
	for _, f := range newFacts {
		if strings.TrimSpace(f.Fact) == "" {
			continue
		}

		// Check for existing similar facts to avoid bloat
		existing, err := a.kg.SearchFacts(ctx, f.Fact, f.Scope, 3)
		isDuplicate := false
		if err == nil {
			for _, ef := range existing {
				if strings.EqualFold(ef.Fact, f.Fact) && ef.Topic == f.Topic {
					isDuplicate = true
					break
				}
			}
		}

		if !isDuplicate {
			f.ID = uuid.New().String()
			f.SourceAgent = "aris:autolearn"
			f.CreatedAt = time.Now()
			_, err := a.kg.AddFact(ctx, f)
			if err == nil {
				savedFacts = append(savedFacts, f)
			}
		}
	}

	return savedFacts, nil
}

func (a *AutoLearner) extractHeuristicFacts(turn ReflectionTurn) []domain.KnowledgeFact {
	var facts []domain.KnowledgeFact
	lower := strings.ToLower(turn.RawInput)

	// Preference / Ratio learning
	if strings.Contains(lower, "siempre") || strings.Contains(lower, "always") || strings.Contains(lower, "prefiero") || strings.Contains(lower, "prefer") {
		if strings.Contains(lower, "16:9") || strings.Contains(lower, "horizontal") || strings.Contains(lower, "landscape") {
			facts = append(facts, domain.KnowledgeFact{
				Topic:   "pref:aspect_ratio",
				Concept: "ratio_preference",
				Fact:    "User prefers 16:9 landscape aspect ratio for wide compositions",
				Labels:  []string{"preference", "aspect_ratio", "16:9"},
				Scope:   domain.ScopeUser,
			})
		} else if strings.Contains(lower, "9:16") || strings.Contains(lower, "vertical") || strings.Contains(lower, "portrait") {
			facts = append(facts, domain.KnowledgeFact{
				Topic:   "pref:aspect_ratio",
				Concept: "ratio_preference",
				Fact:    "User prefers 9:16 vertical portrait aspect ratio",
				Labels:  []string{"preference", "aspect_ratio", "9:16"},
				Scope:   domain.ScopeUser,
			})
		}
	}

	// Negative trigger learning from corrections ("sin...", "no pongas...", "remove...", "without...")
	if strings.Contains(lower, "sin ") || strings.Contains(lower, "no pongas ") || strings.Contains(lower, "without ") || strings.Contains(lower, "remove ") {
		triggers := extractNegativeTriggers(turn.RawInput)
		for _, trig := range triggers {
			facts = append(facts, domain.KnowledgeFact{
				Topic:   "pref:negative",
				Concept: "unwanted_element",
				Fact:    fmt.Sprintf("Avoid element: %s", trig),
				Labels:  []string{"negative", "avoid", trig},
				Scope:   domain.ScopeUser,
			})
		}
	}

	// Style pattern learning
	if strings.Contains(lower, "estilo ") || strings.Contains(lower, "style ") || strings.Contains(lower, "aesthetic") {
		styleName := extractStyleName(turn.RawInput)
		if styleName != "" {
			facts = append(facts, domain.KnowledgeFact{
				Topic:   fmt.Sprintf("style:%s", sanitizeSlug(styleName)),
				Concept: "aesthetic_recipe",
				Fact:    fmt.Sprintf("Style %s requested with parameters: %s", styleName, turn.EnhancedPrompt),
				Labels:  []string{"style", styleName},
				Scope:   domain.ScopeStyle,
			})
		}
	}

	return facts
}

func (a *AutoLearner) extractLLMFacts(ctx context.Context, turn ReflectionTurn) ([]domain.KnowledgeFact, error) {
	systemPrompt := `You are the memory distillation subsystem of ARIS (Autonomous Reasoner for Image System).
Analyze this interaction and extract any implicit or explicit user preferences, aesthetic style rules, or negative triggers.
Return ONLY a valid JSON array of facts:
[
  {
    "topic": "pref:lighting" | "style:name" | "pref:negative" | "project:character",
    "concept": "concept_name",
    "fact": "concise description of the learned rule",
    "scope": "user" | "style" | "project"
  }
]
If there are no new persistent facts to learn, return [].`

	userPrompt := fmt.Sprintf("User Input: %q\nPrevious Prompt: %q\nEnhanced Output: %q\nFeedback: %q",
		turn.RawInput, turn.PreviousPrompt, turn.EnhancedPrompt, turn.UserFeedback)

	raw, err := a.llm.Complete(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	cleaned := cleanJSONBlock(raw)
	var parsed []extractedFactJSON
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, err
	}

	var results []domain.KnowledgeFact
	for _, p := range parsed {
		scope := domain.MemoryScope(p.Scope)
		if scope == "" {
			scope = domain.ScopeUser
		}
		results = append(results, domain.KnowledgeFact{
			Topic:   p.Topic,
			Concept: p.Concept,
			Fact:    p.Fact,
			Labels:  []string{p.Topic, p.Concept},
			Scope:   scope,
		})
	}
	return results, nil
}

func extractNegativeTriggers(input string) []string {
	var triggers []string
	parts := strings.Fields(input)
	for i, word := range parts {
		lower := strings.ToLower(word)
		if (lower == "sin" || lower == "without" || lower == "remove") && i+1 < len(parts) {
			trigger := strings.Trim(parts[i+1], ",.;:!\"'")
			if len(trigger) > 2 {
				triggers = append(triggers, trigger)
			}
		}
	}
	return triggers
}

func extractStyleName(input string) string {
	lower := strings.ToLower(input)
	idx := strings.Index(lower, "estilo ")
	if idx != -1 {
		rest := input[idx+7:]
		fields := strings.Fields(rest)
		if len(fields) > 0 {
			return strings.Trim(fields[0], ",.;:!\"'")
		}
	}
	idx = strings.Index(lower, "style ")
	if idx != -1 {
		rest := input[idx+6:]
		fields := strings.Fields(rest)
		if len(fields) > 0 {
			return strings.Trim(fields[0], ",.;:!\"'")
		}
	}
	return ""
}

func cleanJSONBlock(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	s = strings.TrimSuffix(s, "```")
	return strings.TrimSpace(s)
}

func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else if b.Len() > 0 && b.String()[b.Len()-1] != '_' {
			b.WriteRune('_')
		}
	}
	return strings.Trim(b.String(), "_")
}
