package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"

	"github.com/google/uuid"
)

var _ ports.ImageBackend = (*FalAIBackend)(nil)

// FalAIBackend implements ports.ImageBackend for Fal.ai fast inference.
type FalAIBackend struct {
	apiKey     string
	baseURL    string
	outputDir  string
	httpClient *http.Client
}

// NewFalAIBackend creates a new Fal.ai image backend.
func NewFalAIBackend(apiKey string, outputDir string, httpClient *http.Client) *FalAIBackend {
	if outputDir == "" {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, ".aris", "outputs")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	return &FalAIBackend{
		apiKey:     apiKey,
		baseURL:    "https://fal.run",
		outputDir:  outputDir,
		httpClient: httpClient,
	}
}

func (f *FalAIBackend) Name() string {
	return "falai"
}

func (f *FalAIBackend) SupportsModels() []string {
	return []string{
		"fal-ai/flux-pro/v1.1",
		"fal-ai/flux/dev",
		"fal-ai/flux/schnell",
		"fal-ai/flux-realism",
	}
}

type falRequest struct {
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	ImageSize      string `json:"image_size,omitempty"`
	NumInferenceSteps int `json:"num_inference_steps,omitempty"`
	GuidanceScale  float64 `json:"guidance_scale,omitempty"`
	Seed           int64  `json:"seed,omitempty"`
	EnableSafetyChecker bool `json:"enable_safety_checker"`
}

type falResponse struct {
	Images []struct {
		URL         string `json:"url"`
		Width       int    `json:"width"`
		Height      int    `json:"height"`
		ContentType string `json:"content_type"`
	} `json:"images"`
	Timings struct {
		Inference float64 `json:"inference"`
	} `json:"timings"`
	Detail string `json:"detail,omitempty"`
}

func (f *FalAIBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	if f.apiKey == "" {
		return nil, fmt.Errorf("FAL_KEY or fal.ai api key is not configured. Set FAL_KEY in environment or ~/.aris/config.yaml")
	}

	model := spec.Model
	if model == "" || model == "flux" {
		model = "fal-ai/flux/schnell"
	}

	prompt := spec.EnhancedPrompt
	if prompt == "" {
		prompt = spec.RawPrompt
	}

	var imageSize string
	switch spec.AspectRatio {
	case domain.RatioLandscape, domain.RatioWide:
		imageSize = "landscape_16_9"
	case domain.RatioPortrait, domain.RatioPoster:
		imageSize = "portrait_16_9"
	default:
		imageSize = "square_hd"
	}

	reqPayload := falRequest{
		Prompt:              prompt,
		NegativePrompt:      spec.NegativePrompt,
		ImageSize:           imageSize,
		NumInferenceSteps:   spec.Steps,
		GuidanceScale:       spec.CFGScale,
		Seed:                spec.Seed,
		EnableSafetyChecker: false,
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal fal request: %w", err)
	}

	targetURL := fmt.Sprintf("%s/%s", f.baseURL, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create fal request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Key %s", f.apiKey))
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fal request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read fal response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fal api returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var falResp falResponse
	if err := json.Unmarshal(respBytes, &falResp); err != nil {
		return nil, fmt.Errorf("unmarshal fal response: %w", err)
	}

	if len(falResp.Images) == 0 || falResp.Images[0].URL == "" {
		return nil, fmt.Errorf("fal returned no images in response")
	}

	imageURL := falResp.Images[0].URL

	// Download image from CDN to local cache
	imgReq, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create image download request: %w", err)
	}

	imgResp, err := f.httpClient.Do(imgReq)
	if err != nil {
		return nil, fmt.Errorf("download fal image from CDN: %w", err)
	}
	defer imgResp.Body.Close()

	now := time.Now()
	dayDir := filepath.Join(f.outputDir, now.Format("2006-01-02"))
	_ = os.MkdirAll(dayDir, 0755)

	slug := sanitizeSlug(spec.RawPrompt)
	if len(slug) > 25 {
		slug = slug[:25]
	}
	filename := fmt.Sprintf("aris_%s_%s_%s.jpg", now.Format("20060102_150405"), slug, uuid.New().String()[:8])
	localPath := filepath.Join(dayDir, filename)

	outFile, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("create local image file: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, imgResp.Body)
	if err != nil {
		_ = os.Remove(localPath)
		return nil, fmt.Errorf("save image bytes: %w", err)
	}

	return &domain.ImageResult{
		ID:          uuid.New().String(),
		SpecID:      spec.ID,
		LocalPath:   localPath,
		RemoteURL:   imageURL,
		Format:      "jpg",
		SizeInBytes: written,
		Duration:    time.Since(start),
		Metadata: map[string]any{
			"backend": "falai",
			"model":   model,
			"seed":    spec.Seed,
			"timings": falResp.Timings.Inference,
		},
	}, nil
}
