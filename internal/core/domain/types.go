package domain

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// LoRAConfig represents a Low-Rank Adaptation model and its weight scale.
type LoRAConfig struct {
	Name  string  `json:"name"`
	Scale float64 `json:"scale"`
	Path  string  `json:"path,omitempty"`
}

// ControlNetConfig represents a structural conditioning configuration.
type ControlNetConfig struct {
	Type             string  `json:"type"`                        // e.g. "canny", "depth", "openpose", "lineart", "scribble"
	Strength         float64 `json:"strength"`                    // Conditioning scale [0.0, 2.0]
	ReferenceImage   string  `json:"reference_image,omitempty"`   // Local path or URL to reference image
	InputPath        string  `json:"input_path,omitempty"`        // Alias for reference image path
	Preprocess       bool    `json:"preprocess,omitempty"`        // Flag to trigger local preprocessing
	ProcessedPath    string  `json:"processed_path,omitempty"`    // Path to preprocessed edge/pose/depth map
	PreprocessedHash string  `json:"preprocessed_hash,omitempty"` // Cache key or hash
}

// RefImage returns ReferenceImage or InputPath if either is set.
func (c *ControlNetConfig) RefImage() string {
	if c.ReferenceImage != "" {
		return c.ReferenceImage
	}
	return c.InputPath
}

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

// ReferenceMode defines the generation mode of the image pipeline.
type ReferenceMode string

const (
	ModeText2Img      ReferenceMode = "text2img"
	ModeImg2Img       ReferenceMode = "img2img"
	ModeInpaint       ReferenceMode = "inpaint"
	ModeStyleTransfer ReferenceMode = "style_transfer"
	ModeUpscale       ReferenceMode = "upscale"
)

// ImageSpec defines the complete technical blueprint for an image generation.
type ImageSpec struct {
	ID              string         `json:"id"`
	RawPrompt       string         `json:"raw_prompt"`
	EnhancedPrompt  string         `json:"enhanced_prompt"`
	NegativePrompt  string         `json:"negative_prompt"`
	AspectRatio     AspectRatio    `json:"aspect_ratio"`
	Width           int            `json:"width"`
	Height          int            `json:"height"`
	Steps           int            `json:"steps"`
	CFGScale        float64        `json:"cfg_scale"`
	Seed            int64          `json:"seed"`
	Backend         string         `json:"backend"`
	Model           string         `json:"model"`
	Mode            ReferenceMode  `json:"mode,omitempty"`
	InputImagePath  string         `json:"input_image_path,omitempty"`  // Base reference image (local path or URL)
	MaskImagePath   string         `json:"mask_image_path,omitempty"`   // Inpainting mask image (local path or URL)
	DenoiseStrength float64        `json:"denoise_strength,omitempty"` // [0.0, 1.0] divergence from source
	ScaleFactor     int            `json:"scale_factor,omitempty"`     // 2, 4, 8 super-resolution scale
	RestoreFaces    bool           `json:"restore_faces,omitempty"`    // Toggles facial reconstruction
	FaceFidelity    float64        `json:"face_fidelity,omitempty"`    // [0.0, 1.0] facial fidelity weighting
	UpscalerModel   string              `json:"upscaler_model,omitempty"`   // Specific upscaler or face model name
	LoRAs           []LoRAConfig        `json:"loras,omitempty"`            // LoRA models and scales
	ControlNets     []ControlNetConfig  `json:"controlnets,omitempty"`      // ControlNet structural conditioning
	ExtraParams     map[string]any      `json:"extra_params,omitempty"`
	CreatedAt       time.Time           `json:"created_at"`
}

// IsImg2Img returns true if the spec describes an image-to-image or style-transfer task.
func (s *ImageSpec) IsImg2Img() bool {
	if s.Mode == ModeImg2Img || s.Mode == ModeStyleTransfer {
		return true
	}
	return s.InputImagePath != "" && s.MaskImagePath == "" && s.Mode != ModeInpaint && !s.IsUpscale()
}

// IsInpaint returns true if the spec describes an inpainting task.
func (s *ImageSpec) IsInpaint() bool {
	if s.Mode == ModeInpaint {
		return true
	}
	return s.MaskImagePath != ""
}

// IsUpscale returns true if the spec describes an upscaling or face restoration task.
func (s *ImageSpec) IsUpscale() bool {
	return s.Mode == ModeUpscale || s.ScaleFactor > 1 || s.RestoreFaces
}

// HasLoRA returns true if the spec has at least one configured LoRA.
func (s *ImageSpec) HasLoRA() bool {
	return len(s.LoRAs) > 0
}

// HasControlNet returns true if the spec has at least one configured ControlNet.
func (s *ImageSpec) HasControlNet() bool {
	return len(s.ControlNets) > 0
}

// ApplyDefaults applies default values for mode, denoise strength, scale factors, and clamps parameters.
func (s *ImageSpec) ApplyDefaults() {
	if s.Mode == "" {
		if s.IsUpscale() {
			s.Mode = ModeUpscale
		} else if s.MaskImagePath != "" {
			s.Mode = ModeInpaint
		} else if s.InputImagePath != "" {
			s.Mode = ModeImg2Img
		} else {
			s.Mode = ModeText2Img
		}
	}

	if s.IsUpscale() {
		if s.ScaleFactor == 0 {
			s.ScaleFactor = 4
		}
		if s.RestoreFaces {
			if s.FaceFidelity == 0.0 {
				s.FaceFidelity = 0.75
			}
			if s.FaceFidelity < 0.0 {
				s.FaceFidelity = 0.0
			} else if s.FaceFidelity > 1.0 {
				s.FaceFidelity = 1.0
			}
		}
	}

	if s.Mode != ModeText2Img && s.Mode != ModeUpscale && s.DenoiseStrength == 0.0 {
		s.DenoiseStrength = 0.70
	}

	// Clamp denoise strength to [0.0, 1.0]
	if s.DenoiseStrength < 0.0 {
		s.DenoiseStrength = 0.0
	} else if s.DenoiseStrength > 1.0 {
		s.DenoiseStrength = 1.0
	}

	// Apply defaults and clamping to LoRAs
	for i := range s.LoRAs {
		if s.LoRAs[i].Scale == 0.0 {
			s.LoRAs[i].Scale = 1.0
		}
		if s.LoRAs[i].Scale > 2.0 {
			s.LoRAs[i].Scale = 2.0
		} else if s.LoRAs[i].Scale < 0.0 {
			s.LoRAs[i].Scale = 0.0
		}
	}

	// Apply defaults and clamping to ControlNets
	for i := range s.ControlNets {
		if s.ControlNets[i].Strength == 0.0 {
			s.ControlNets[i].Strength = 1.0
		}
		if s.ControlNets[i].Strength > 2.0 {
			s.ControlNets[i].Strength = 2.0
		} else if s.ControlNets[i].Strength < 0.0 {
			s.ControlNets[i].Strength = 0.0
		}
		if s.ControlNets[i].ReferenceImage == "" && s.ControlNets[i].InputPath != "" {
			s.ControlNets[i].ReferenceImage = s.ControlNets[i].InputPath
		} else if s.ControlNets[i].InputPath == "" && s.ControlNets[i].ReferenceImage != "" {
			s.ControlNets[i].InputPath = s.ControlNets[i].ReferenceImage
		}
	}
}

// Validate checks internal consistency of the ImageSpec fields.
func (s *ImageSpec) Validate() error {
	if s.MaskImagePath != "" && s.InputImagePath == "" {
		return fmt.Errorf("mask requires a base reference image")
	}
	if s.IsUpscale() {
		if s.ScaleFactor != 2 && s.ScaleFactor != 4 && s.ScaleFactor != 8 {
			return fmt.Errorf("unsupported scale factor %d: supported scale factors are 2, 4, and 8", s.ScaleFactor)
		}
	}

	validCNTypes := map[string]bool{
		"canny":    true,
		"depth":    true,
		"openpose": true,
		"lineart":  true,
		"scribble": true,
	}

	for _, cn := range s.ControlNets {
		cnType := strings.ToLower(strings.TrimSpace(cn.Type))
		if !validCNTypes[cnType] {
			return fmt.Errorf("unsupported controlnet type %q: supported types are canny, depth, openpose, lineart, scribble", cn.Type)
		}
		refImg := cn.RefImage()
		if refImg != "" && !strings.HasPrefix(refImg, "http://") && !strings.HasPrefix(refImg, "https://") && !strings.HasPrefix(refImg, "data:") {
			if _, err := os.Stat(refImg); os.IsNotExist(err) {
				return fmt.Errorf("controlnet reference image does not exist: %s", refImg)
			}
		}
	}
	return nil
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
