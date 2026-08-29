package llm

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"

	"github.com/google/uuid"
)

var _ ports.LLMProvider = (*PassthroughProvider)(nil)

// PassthroughProvider provides prompt synthesis without external LLM API dependencies.
type PassthroughProvider struct{}

// NewPassthroughProvider creates a zero-dependency heuristic prompt architect.
func NewPassthroughProvider() *PassthroughProvider {
	return &PassthroughProvider{}
}

func (p *PassthroughProvider) Name() string {
	return "passthrough-heuristic"
}

func (p *PassthroughProvider) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	return userPrompt, nil
}

// ReasonPrompt heuristically enriches the raw prompt with artistic modifiers and user facts.
func (p *PassthroughProvider) ReasonPrompt(ctx context.Context, input string, facts []domain.KnowledgeFact) (*domain.ImageSpec, error) {
	// Detect aspect ratio keywords in input
	ratio := domain.RatioSquare
	lower := strings.ToLower(input)

	if strings.Contains(lower, "16:9") || strings.Contains(lower, "landscape") || strings.Contains(lower, "wallpaper") || strings.Contains(lower, "horizontal") {
		ratio = domain.RatioLandscape
	} else if strings.Contains(lower, "9:16") || strings.Contains(lower, "portrait") || strings.Contains(lower, "vertical") || strings.Contains(lower, "story") {
		ratio = domain.RatioPortrait
	} else if strings.Contains(lower, "4:3") {
		ratio = domain.RatioPhoto
	} else if strings.Contains(lower, "3:4") {
		ratio = domain.RatioPoster
	} else if strings.Contains(lower, "21:9") || strings.Contains(lower, "ultrawide") || strings.Contains(lower, "cinematic") {
		ratio = domain.RatioWide
	}

	// Check if user preferences dictate aspect ratio
	for _, f := range facts {
		if strings.Contains(strings.ToLower(f.Concept), "aspect_ratio") || strings.Contains(strings.ToLower(f.Topic), "aspect_ratio") {
			ratio = domain.ParseAspectRatio(f.Fact)
		}
	}

	w, h := ratio.Dimensions(1024)

	var enhancedParts []string
	enhancedParts = append(enhancedParts, input)

	// Append style facts from knowledge graph
	for _, f := range facts {
		if f.Scope == domain.ScopeStyle && f.Fact != "" {
			enhancedParts = append(enhancedParts, f.Fact)
		}
	}

	// Default quality enhancers for diffusion
	if !strings.Contains(lower, "photo") && !strings.Contains(lower, "illustration") && !strings.Contains(lower, "render") {
		enhancedParts = append(enhancedParts, "highly detailed, 8k resolution, cinematic lighting, masterpiece")
	}

	enhanced := strings.Join(enhancedParts, ", ")

	negatives := []string{"blurry", "low quality", "distorted", "watermark", "extra limbs", "jpeg artifacts"}
	for _, f := range facts {
		if strings.Contains(strings.ToLower(f.Concept), "negative") {
			negatives = append(negatives, f.Fact)
		}
	}

	// Random seed
	seed := time.Now().UnixNano() % 10000000
	if seed < 0 {
		seed = -seed
	}
	if seed == 0 {
		seed = int64(rand.Intn(999999) + 1)
	}

	return &domain.ImageSpec{
		ID:             uuid.New().String(),
		RawPrompt:      input,
		EnhancedPrompt: enhanced,
		NegativePrompt: strings.Join(negatives, ", "),
		AspectRatio:    ratio,
		Width:          w,
		Height:         h,
		Steps:          20,
		CFGScale:       7.0,
		Seed:           seed,
		Backend:        "pollinations",
		Model:          "flux",
		CreatedAt:      time.Now(),
	}, nil
}
