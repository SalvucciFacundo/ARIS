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

var _ ports.ImageBackend = (*OpenAIBackend)(nil)

// OpenAIBackend implements ports.ImageBackend for DALL-E 3 / DALL-E 2.
type OpenAIBackend struct {
	apiKey     string
	baseURL    string
	outputDir  string
	httpClient *http.Client
}

// NewOpenAIBackend creates an OpenAI DALL-E backend.
func NewOpenAIBackend(apiKey, baseURL, outputDir string, httpClient *http.Client) *OpenAIBackend {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if outputDir == "" {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, ".aris", "outputs")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 90 * time.Second}
	}
	return &OpenAIBackend{
		apiKey:     apiKey,
		baseURL:    baseURL,
		outputDir:  outputDir,
		httpClient: httpClient,
	}
}

func (o *OpenAIBackend) Name() string {
	return "openai"
}

func (o *OpenAIBackend) SupportsModels() []string {
	return []string{"dall-e-3", "dall-e-2"}
}

type openAIImageReq struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	N       int    `json:"n"`
	Size    string `json:"size"`
	Quality string `json:"quality,omitempty"`
	Style   string `json:"style,omitempty"`
}

type openAIImageResp struct {
	Data []struct {
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (o *OpenAIBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	if o.apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not configured")
	}

	model := spec.Model
	if model == "" || model == "flux" {
		model = "dall-e-3"
	}

	prompt := spec.EnhancedPrompt
	if prompt == "" {
		prompt = spec.RawPrompt
	}

	var size string
	switch spec.AspectRatio {
	case domain.RatioLandscape, domain.RatioWide:
		size = "1792x1024"
	case domain.RatioPortrait, domain.RatioPoster:
		size = "1024x1792"
	default:
		size = "1024x1024"
	}

	reqPayload := openAIImageReq{
		Model:   model,
		Prompt:  prompt,
		N:       1,
		Size:    size,
		Quality: "standard",
		Style:   "vivid",
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/images/generations", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai image request failed: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var parsed openAIImageResp
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal openai response: %w", err)
	}

	if parsed.Error != nil {
		return nil, fmt.Errorf("openai error: %s", parsed.Error.Message)
	}
	if len(parsed.Data) == 0 || parsed.Data[0].URL == "" {
		return nil, fmt.Errorf("openai returned no image URL")
	}

	imageURL := parsed.Data[0].URL

	// Download image
	imgReq, err := http.NewRequestWithContext(ctx, http.MethodGet, imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create download request: %w", err)
	}
	imgResp, err := o.httpClient.Do(imgReq)
	if err != nil {
		return nil, fmt.Errorf("download image: %w", err)
	}
	defer imgResp.Body.Close()

	now := time.Now()
	dayDir := filepath.Join(o.outputDir, now.Format("2006-01-02"))
	_ = os.MkdirAll(dayDir, 0755)

	slug := sanitizeSlug(spec.RawPrompt)
	if len(slug) > 25 {
		slug = slug[:25]
	}
	filename := fmt.Sprintf("aris_%s_%s_%s.png", now.Format("20060102_150405"), slug, uuid.New().String()[:8])
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
		Format:      "png",
		SizeInBytes: written,
		Duration:    time.Since(start),
		Metadata: map[string]any{
			"backend":        "openai",
			"model":          model,
			"revised_prompt": parsed.Data[0].RevisedPrompt,
		},
	}, nil
}
