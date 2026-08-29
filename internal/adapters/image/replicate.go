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

var _ ports.ImageBackend = (*ReplicateBackend)(nil)

// ReplicateBackend implements ports.ImageBackend for Replicate API.
type ReplicateBackend struct {
	apiToken   string
	baseURL    string
	outputDir  string
	httpClient *http.Client
}

// NewReplicateBackend creates a Replicate image backend.
func NewReplicateBackend(apiToken, outputDir string, httpClient *http.Client) *ReplicateBackend {
	if outputDir == "" {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, ".aris", "outputs")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	return &ReplicateBackend{
		apiToken:   apiToken,
		baseURL:    "https://api.replicate.com/v1",
		outputDir:  outputDir,
		httpClient: httpClient,
	}
}

func (r *ReplicateBackend) Name() string {
	return "replicate"
}

func (r *ReplicateBackend) SupportsModels() []string {
	return []string{
		"black-forest-labs/flux-schnell",
		"black-forest-labs/flux-dev",
		"stability-ai/sdxl",
	}
}

type replicatePredictionReq struct {
	Version string         `json:"version,omitempty"`
	Input   map[string]any `json:"input"`
}

type replicatePredictionResp struct {
	ID     string `json:"id"`
	Status string `json:"status"` // "starting", "processing", "succeeded", "failed", "canceled"
	Output any    `json:"output"` // string or []string
	Error  string `json:"error"`
	URLs   struct {
		Get string `json:"get"`
	} `json:"urls"`
}

func (r *ReplicateBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	if r.apiToken == "" {
		return nil, fmt.Errorf("REPLICATE_API_TOKEN is not configured")
	}

	model := spec.Model
	if model == "" || model == "flux" {
		model = "black-forest-labs/flux-schnell"
	}

	prompt := spec.EnhancedPrompt
	if prompt == "" {
		prompt = spec.RawPrompt
	}

	var ratio string
	switch spec.AspectRatio {
	case domain.RatioLandscape, domain.RatioWide:
		ratio = "16:9"
	case domain.RatioPortrait, domain.RatioPoster:
		ratio = "9:16"
	case domain.RatioPhoto:
		ratio = "4:3"
	default:
		ratio = "1:1"
	}

	inputParams := map[string]any{
		"prompt":       prompt,
		"aspect_ratio": ratio,
		"output_format": "jpg",
	}
	if spec.NegativePrompt != "" {
		inputParams["negative_prompt"] = spec.NegativePrompt
	}
	if spec.Seed > 0 {
		inputParams["seed"] = spec.Seed
	}

	reqPayload := replicatePredictionReq{
		Input: inputParams,
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal replicate request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/models/%s/predictions", r.baseURL, model)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create replicate request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Prefer", "wait") // Replicate synchronous wait option

	start := time.Now()
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("replicate prediction request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read replicate response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("replicate returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var pred replicatePredictionResp
	if err := json.Unmarshal(respBytes, &pred); err != nil {
		return nil, fmt.Errorf("unmarshal replicate response: %w", err)
	}

	// Poll if status is still starting or processing
	pollURL := pred.URLs.Get
	for pred.Status != "succeeded" && pred.Status != "failed" && pred.Status != "canceled" {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}

		if pollURL == "" {
			pollURL = fmt.Sprintf("%s/predictions/%s", r.baseURL, pred.ID)
		}

		pReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, pollURL, nil)
		pReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.apiToken))
		pResp, err := r.httpClient.Do(pReq)
		if err != nil {
			return nil, fmt.Errorf("poll replicate status: %w", err)
		}

		pBytes, _ := io.ReadAll(pResp.Body)
		pResp.Body.Close()
		_ = json.Unmarshal(pBytes, &pred)
	}

	if pred.Status != "succeeded" {
		return nil, fmt.Errorf("replicate generation failed with status %s: %s", pred.Status, pred.Error)
	}

	// Extract Image URL
	var imageURL string
	switch out := pred.Output.(type) {
	case string:
		imageURL = out
	case []any:
		if len(out) > 0 {
			if s, ok := out[0].(string); ok {
				imageURL = s
			}
		}
	}

	if imageURL == "" {
		return nil, fmt.Errorf("could not extract image URL from replicate output: %v", pred.Output)
	}

	// Download image
	imgReq, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	imgResp, err := r.httpClient.Do(imgReq)
	if err != nil {
		return nil, fmt.Errorf("download replicate image: %w", err)
	}
	defer imgResp.Body.Close()

	now := time.Now()
	dayDir := filepath.Join(r.outputDir, now.Format("2006-01-02"))
	_ = os.MkdirAll(dayDir, 0755)

	slug := sanitizeSlug(spec.RawPrompt)
	if len(slug) > 25 {
		slug = slug[:25]
	}
	filename := fmt.Sprintf("aris_%s_%s_%s.jpg", now.Format("20060102_150405"), slug, uuid.New().String()[:8])
	localPath := filepath.Join(dayDir, filename)

	outFile, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("create local file: %w", err)
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
			"backend": "replicate",
			"model":   model,
			"id":      pred.ID,
		},
	}, nil
}
