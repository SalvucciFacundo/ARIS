package domain

import (
	"fmt"
	"time"
)

// AspectRatio represents the target visual proportion.
type AspectRatio string

const (
	RatioSquare    AspectRatio = "1:1"   // 1024x1024
	RatioLandscape AspectRatio = "16:9"  // 1344x768
	RatioPortrait  AspectRatio = "9:16"  // 768x1344
	RatioPhoto     AspectRatio = "4:3"   // 1152x864
	RatioPoster    AspectRatio = "3:4"   // 864x1152
	RatioWide      AspectRatio = "21:9"  // 1536x640
)

// Dimensions calculates width and height for standard diffusion models based on base resolution.
func (r AspectRatio) Dimensions(base int) (int, int) {
	if base <= 0 {
		base = 1024
	}
	switch r {
	case RatioLandscape:
		return (base * 16 / 12), (base * 9 / 12)
	case RatioPortrait:
		return (base * 9 / 12), (base * 16 / 12)
	case RatioPhoto:
		return (base * 4 / 3), base
	case RatioPoster:
		return base, (base * 4 / 3)
	case RatioWide:
		return (base * 21 / 14), (base * 9 / 14)
	case RatioSquare:
		fallthrough
	default:
		return base, base
	}
}

// ParseAspectRatio parses string into a valid AspectRatio or returns default Square.
func ParseAspectRatio(s string) AspectRatio {
	switch s {
	case "16:9", "landscape", "horizontal", "wide":
		return RatioLandscape
	case "9:16", "portrait", "vertical", "story":
		return RatioPortrait
	case "4:3", "photo":
		return RatioPhoto
	case "3:4", "poster":
		return RatioPoster
	case "21:9", "ultrawide", "cinematic":
		return RatioWide
	case "1:1", "square":
		fallthrough
	default:
		return RatioSquare
	}
}

// MemoryScope defines the boundary of a knowledge fact.
type MemoryScope string

const (
	ScopeUser    MemoryScope = "user"    // Global user aesthetic & generation preferences
	ScopeStyle   MemoryScope = "style"   // Curated artistic recipes, lighting, cameras, models
	ScopeProject MemoryScope = "project" // Specific character sheet, campaign, or collection
)

// ImageSpec defines the complete technical blueprint for an image generation.
type ImageSpec struct {
	ID             string         `json:"id"`
	RawPrompt      string         `json:"raw_prompt"`
	EnhancedPrompt string         `json:"enhanced_prompt"`
	NegativePrompt string         `json:"negative_prompt"`
	AspectRatio    AspectRatio    `json:"aspect_ratio"`
	Width          int            `json:"width"`
	Height         int            `json:"height"`
	Steps          int            `json:"steps"`
	CFGScale       float64        `json:"cfg_scale"`
	Seed           int64          `json:"seed"`
	Backend        string         `json:"backend"`
	Model          string         `json:"model"`
	InputImagePath string         `json:"input_image_path,omitempty"` // For img2img
	ExtraParams    map[string]any `json:"extra_params,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// String returns a summary of the ImageSpec for logging/display.
func (s *ImageSpec) String() string {
	return fmt.Sprintf("[%s/%s] %dx%d (seed: %d) prompt: %q", s.Backend, s.Model, s.Width, s.Height, s.Seed, s.EnhancedPrompt)
}

// ImageResult represents the rendered output asset.
type ImageResult struct {
	ID          string         `json:"id"`
	SpecID      string         `json:"spec_id"`
	LocalPath   string         `json:"local_path"`
	RemoteURL   string         `json:"remote_url,omitempty"`
	Format      string         `json:"format"` // png, webp, jpg
	SizeInBytes int64          `json:"size_in_bytes"`
	Duration    time.Duration  `json:"duration"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// KnowledgeFact represents an atomic unit of recalled or learned knowledge.
type KnowledgeFact struct {
	ID          string      `json:"id"`
	Topic       string      `json:"topic"`        // e.g. "style:cyberpunk", "pref:lighting"
	Concept     string      `json:"concept"`      // e.g. "neon_reflections", "negative_defaults"
	Fact        string      `json:"fact"`         // e.g. "Use volumetric teal/magenta neon fog"
	SourceAgent string      `json:"source_agent"` // "aris:reasoner", "user:feedback"
	Labels      []string    `json:"labels"`
	Project     string      `json:"project"`
	Scope       MemoryScope `json:"scope"`
	CreatedAt   time.Time   `json:"created_at"`
}

// GenerationRecord is the persistent log entry of a generation run.
type GenerationRecord struct {
	ID             string      `json:"id"`
	PromptRaw      string      `json:"prompt_raw"`
	PromptEnhanced string      `json:"prompt_enhanced"`
	NegativePrompt string      `json:"negative_prompt"`
	Backend        string      `json:"backend"`
	Model          string      `json:"model"`
	Width          int         `json:"width"`
	Height         int         `json:"height"`
	Steps          int         `json:"steps"`
	CFGScale       float64     `json:"cfg_scale"`
	Seed           int64       `json:"seed"`
	ImagePath      string      `json:"image_path"`
	ThumbPath      string      `json:"thumb_path,omitempty"`
	DurationMs     int64       `json:"duration_ms"`
	Rating         int         `json:"rating"` // -1, 0, 1
	Feedback       string      `json:"feedback,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
}
