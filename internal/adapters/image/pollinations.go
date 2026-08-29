package image

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"

	"github.com/google/uuid"
)

var _ ports.ImageBackend = (*PollinationsBackend)(nil)

// PollinationsBackend implements ports.ImageBackend using Pollinations.ai free API.
type PollinationsBackend struct {
	client    *http.Client
	outputDir string
	baseURL   string
}

// Option configures PollinationsBackend.
type Option func(*PollinationsBackend)

// WithHTTPClient sets a custom http.Client (e.g. for testing mocks).
func WithHTTPClient(client *http.Client) Option {
	return func(p *PollinationsBackend) {
		p.client = client
	}
}

// WithOutputDir sets the directory where generated images are saved.
func WithOutputDir(dir string) Option {
	return func(p *PollinationsBackend) {
		p.outputDir = dir
	}
}

// WithBaseURL sets custom base URL (for testing or proxying).
func WithBaseURL(rawURL string) Option {
	return func(p *PollinationsBackend) {
		p.baseURL = rawURL
	}
}

// NewPollinationsBackend creates a new Pollinations backend instance.
func NewPollinationsBackend(opts ...Option) *PollinationsBackend {
	home, _ := os.UserHomeDir()
	defaultOut := filepath.Join(home, ".aris", "outputs")

	p := &PollinationsBackend{
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
		outputDir: defaultOut,
		baseURL:   "https://image.pollinations.ai/prompt",
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// Name returns the backend identifier.
func (p *PollinationsBackend) Name() string {
	return "pollinations"
}

// SupportsModels returns the models supported by Pollinations.
func (p *PollinationsBackend) SupportsModels() []string {
	return []string{"flux", "flux-realism", "flux-cablyai", "flux-anime", "flux-3d", "turbo"}
}

// BuildURL constructs the full HTTP GET URL for Pollinations.
func (p *PollinationsBackend) BuildURL(spec *domain.ImageSpec) string {
	prompt := spec.EnhancedPrompt
	if prompt == "" {
		prompt = spec.RawPrompt
	}

	// Escape path component for prompt
	encodedPrompt := url.PathEscape(prompt)
	fullURL := fmt.Sprintf("%s/%s", strings.TrimRight(p.baseURL, "/"), encodedPrompt)

	model := spec.Model
	if model == "" {
		model = "flux"
	}

	w, h := spec.Width, spec.Height
	if w <= 0 || h <= 0 {
		w, h = spec.AspectRatio.Dimensions(1024)
	}

	q := url.Values{}
	q.Set("width", fmt.Sprintf("%d", w))
	q.Set("height", fmt.Sprintf("%d", h))
	q.Set("model", model)
	q.Set("nologo", "true")
	q.Set("enhance", "false") // ARIS does its own reasoning

	if spec.Seed > 0 {
		q.Set("seed", fmt.Sprintf("%d", spec.Seed))
	}
	if spec.NegativePrompt != "" {
		q.Set("negative", spec.NegativePrompt)
	}
	if spec.InputImagePath != "" && (strings.HasPrefix(spec.InputImagePath, "http://") || strings.HasPrefix(spec.InputImagePath, "https://")) {
		q.Set("image", spec.InputImagePath)
	}

	return fmt.Sprintf("%s?%s", fullURL, q.Encode())
}

// Generate fetches the rendered image from Pollinations and writes it to disk.
func (p *PollinationsBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	spec.ApplyDefaults()
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid spec: %w", err)
	}

	if spec.IsInpaint() {
		return nil, fmt.Errorf("backend 'pollinations' does not support masked inpainting; please use falai, comfyui, or openai")
	}

	if spec.IsUpscale() {
		return nil, fmt.Errorf("backend 'pollinations' does not support super-resolution upscaling or face restoration; please use falai or comfyui")
	}

	start := time.Now()
	targetURL := p.BuildURL(spec)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "ARIS-Agent/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pollinations request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("pollinations returned status %d: %s", resp.StatusCode, string(body))
	}

	// Determine output directory: outputDir/YYYY-MM-DD
	now := time.Now()
	dayDir := filepath.Join(p.outputDir, now.Format("2006-01-02"))
	if err := os.MkdirAll(dayDir, 0755); err != nil {
		return nil, fmt.Errorf("create output directory: %w", err)
	}

	slug := sanitizeSlug(spec.RawPrompt)
	if len(slug) > 30 {
		slug = slug[:30]
	}
	if slug == "" {
		slug = "generation"
	}

	filename := fmt.Sprintf("aris_%s_%s_%s.jpg", now.Format("20060102_150405"), slug, uuid.New().String()[:8])
	localPath := filepath.Join(dayDir, filename)

	outFile, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("create local image file: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		_ = os.Remove(localPath)
		return nil, fmt.Errorf("write image data to disk: %w", err)
	}

	duration := time.Since(start)
	result := &domain.ImageResult{
		ID:          uuid.New().String(),
		SpecID:      spec.ID,
		LocalPath:   localPath,
		RemoteURL:   targetURL,
		Format:      "jpg",
		SizeInBytes: written,
		Duration:    duration,
		Metadata: map[string]any{
			"backend": "pollinations",
			"model":   spec.Model,
			"width":   spec.Width,
			"height":  spec.Height,
			"seed":    spec.Seed,
		},
	}

	return result, nil
}

var nonAlphaNum = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func sanitizeSlug(s string) string {
	s = strings.ToLower(s)
	s = nonAlphaNum.ReplaceAllString(s, "_")
	return strings.Trim(s, "_")
}
