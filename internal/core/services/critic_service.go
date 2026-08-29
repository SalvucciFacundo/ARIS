package services

import (
	"context"
	"fmt"
	"math/rand"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
)

// CriticService coordinates quality assurance and automated self-healing.
type CriticService struct {
	critic    ports.VisionCritic
	threshold float64
	autoHeal  bool
}

// NewCriticService creates a new critic service.
func NewCriticService(critic ports.VisionCritic, threshold float64, autoHeal bool) *CriticService {
	if threshold <= 0 {
		threshold = 0.60
	}
	return &CriticService{
		critic:    critic,
		threshold: threshold,
		autoHeal:  autoHeal,
	}
}

// InspectAndHeal evaluates the rendered image and triggers a single re-roll if score is below threshold.
func (c *CriticService) InspectAndHeal(
	ctx context.Context,
	backend ports.ImageBackend,
	spec *domain.ImageSpec,
	initialResult *domain.ImageResult,
) (*domain.ImageSpec, *domain.ImageResult, error) {
	if c.critic == nil {
		return spec, initialResult, nil
	}

	score, critique, err := c.critic.Evaluate(ctx, initialResult.LocalPath, spec)
	if err != nil {
		// Non-fatal: if critic fails, deliver initial result with note
		if initialResult.Metadata == nil {
			initialResult.Metadata = make(map[string]any)
		}
		initialResult.Metadata["critic_error"] = err.Error()
		return spec, initialResult, nil
	}

	if initialResult.Metadata == nil {
		initialResult.Metadata = make(map[string]any)
	}
	initialResult.Metadata["critic_score"] = score
	initialResult.Metadata["critic_notes"] = critique

	// Check if self-healing is warranted
	if score >= c.threshold || !c.autoHeal {
		return spec, initialResult, nil
	}

	// Trigger Self-Healing Re-roll
	healedSpec := *spec
	healedSpec.Seed = int64(rand.Intn(9000000) + 1000000)

	// Strengthen positive and negative prompts
	healedSpec.EnhancedPrompt = spec.EnhancedPrompt + ", crisp sharp focus, award winning lighting"
	if healedSpec.NegativePrompt != "" {
		healedSpec.NegativePrompt = spec.NegativePrompt + ", blurry, distorted, extra limbs, artifacts, malformed"
	} else {
		healedSpec.NegativePrompt = "blurry, distorted, extra limbs, artifacts, malformed"
	}

	healedResult, err := backend.Generate(ctx, &healedSpec)
	if err != nil {
		// If re-roll fails, fallback to initial result
		initialResult.Metadata["self_heal_error"] = err.Error()
		return spec, initialResult, nil
	}

	// Re-evaluate healed result
	finalScore, finalCritique, fErr := c.critic.Evaluate(ctx, healedResult.LocalPath, &healedSpec)
	if fErr == nil {
		if healedResult.Metadata == nil {
			healedResult.Metadata = make(map[string]any)
		}
		healedResult.Metadata["self_healed"] = true
		healedResult.Metadata["initial_score"] = score
		healedResult.Metadata["critic_score"] = finalScore
		healedResult.Metadata["critic_notes"] = fmt.Sprintf("Healed from %.2f -> %.2f: %s", score, finalScore, finalCritique)
	}

	return &healedSpec, healedResult, nil
}
