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

var _ ports.ImageBackend = (*HuggingFaceBackend)(nil)

// HuggingFaceBackend implements ports.ImageBackend for HuggingFace Inference API.
type HuggingFaceBackend struct {
	token      string
	baseURL    string
	outputDir  string
	httpClient *http.Client
}

// NewHuggingFaceBackend creates a new HuggingFace backend.
func NewHuggingFaceBackend(token, outputDir string, httpClient *http.Client) *HuggingFaceBackend {
	if outputDir == "" {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, ".aris", "outputs")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	return &HuggingFaceBackend{
		token:      token,
		baseURL:    "https://api-inference.huggingface.co/models",
		outputDir:  outputDir,
		httpClient: httpClient,
	}
}

func (h *HuggingFaceBackend) Name() string {
	return "huggingface"
}

func (h *HuggingFaceBackend) SupportsModels() []string {
	return []string{
		"stabilityai/stable-diffusion-3.5-large",
		"black-forest-labs/FLUX.1-dev",
		"black-forest-labs/FLUX.1-schnell",
	}
}

type hfRequest struct {
	Inputs     string         `json:"inputs"`
	Parameters map[string]any `json:"parameters,omitempty"`
}

func (h *HuggingFaceBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	model := spec.Model
	if model == "" || model == "flux" {
		model = "black-forest-labs/FLUX.1-schnell"
	}

	prompt := spec.EnhancedPrompt
	if prompt == "" {
		prompt = spec.RawPrompt
	}

	reqPayload := hfRequest{
		Inputs: prompt,
		Parameters: map[string]any{
			"width":  spec.Width,
			"height": spec.Height,
		},
	}
	if spec.NegativePrompt != "" {
		reqPayload.Parameters["negative_prompt"] = spec.NegativePrompt
	}
	if spec.Seed > 0 {
		reqPayload.Parameters["seed"] = spec.Seed
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal huggingface request: %w", err)
	}

	targetURL := fmt.Sprintf("%s/%s", h.baseURL, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create huggingface request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if h.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", h.token))
	}

	start := time.Now()
	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("huggingface request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("huggingface returned status %d: %s", resp.StatusCode, string(respBody))
	}

	now := time.Now()
	dayDir := filepath.Join(h.outputDir, now.Format("2006-01-02"))
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

	written, err := io.Copy(outFile, resp.Body)
	if err != nil {
		_ = os.Remove(localPath)
		return nil, fmt.Errorf("save image bytes: %w", err)
	}

	return &domain.ImageResult{
		ID:          uuid.New().String(),
		SpecID:      spec.ID,
		LocalPath:   localPath,
		RemoteURL:   targetURL,
		Format:      "jpg",
		SizeInBytes: written,
		Duration:    time.Since(start),
		Metadata: map[string]any{
			"backend": "huggingface",
			"model":   model,
		},
	}, nil
}
