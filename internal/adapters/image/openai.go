package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
	"aris/pkg/imgutil"

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

	spec.ApplyDefaults()
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid spec: %w", err)
	}

	prompt := spec.EnhancedPrompt
	if prompt == "" {
		prompt = spec.RawPrompt
	}

	start := time.Now()

	// If Img2Img or Inpainting, use /v1/images/edits endpoint with multipart/form-data
	if spec.IsImg2Img() || spec.IsInpaint() {
		return o.generateEdit(ctx, spec, prompt, start)
	}

	// Standard text-to-image generation
	model := spec.Model
	if model == "" || model == "flux" {
		model = "dall-e-3"
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

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai image request failed: %w", err)
	}
	defer resp.Body.Close()

	return o.parseAndDownloadResponse(ctx, spec, model, prompt, resp, start)
}

func (o *OpenAIBackend) generateEdit(ctx context.Context, spec *domain.ImageSpec, prompt string, start time.Time) (*domain.ImageResult, error) {
	model := spec.Model
	if model == "" || model == "flux" || model == "dall-e-3" {
		model = "dall-e-2" // OpenAI edits require dall-e-2
	}

	baseData, _, err := imgutil.LoadAndValidateImage(spec.InputImagePath, imgutil.MaxImageSize)
	if err != nil {
		return nil, fmt.Errorf("load base image for openai edit: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add base image
	partImg, err := writer.CreateFormFile("image", "image.png")
	if err != nil {
		return nil, fmt.Errorf("create image form file: %w", err)
	}
	if _, err := partImg.Write(baseData); err != nil {
		return nil, fmt.Errorf("write image part: %w", err)
	}

	// Add mask if present
	if spec.MaskImagePath != "" {
		maskData, _, err := imgutil.LoadAndValidateImage(spec.MaskImagePath, imgutil.MaxImageSize)
		if err != nil {
			return nil, fmt.Errorf("load mask image for openai edit: %w", err)
		}
		partMask, err := writer.CreateFormFile("mask", "mask.png")
		if err != nil {
			return nil, fmt.Errorf("create mask form file: %w", err)
		}
		if _, err := partMask.Write(maskData); err != nil {
			return nil, fmt.Errorf("write mask part: %w", err)
		}
	}

	_ = writer.WriteField("prompt", prompt)
	_ = writer.WriteField("model", model)
	_ = writer.WriteField("n", "1")
	_ = writer.WriteField("size", "1024x1024")

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close multipart writer: %w", err)
	}

	endpoint := fmt.Sprintf("%s/images/edits", o.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return nil, fmt.Errorf("create openai edit request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", o.apiKey))
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai edit request failed: %w", err)
	}
	defer resp.Body.Close()

	return o.parseAndDownloadResponse(ctx, spec, model, prompt, resp, start)
}

func (o *OpenAIBackend) parseAndDownloadResponse(ctx context.Context, spec *domain.ImageSpec, model, prompt string, resp *http.Response, start time.Time) (*domain.ImageResult, error) {
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
