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
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"

	"github.com/google/uuid"
)

var _ ports.ImageBackend = (*ComfyUIBackend)(nil)

// ComfyUIBackend implements ports.ImageBackend for local ComfyUI instances.
type ComfyUIBackend struct {
	baseURL    string
	outputDir  string
	httpClient *http.Client
}

// NewComfyUIBackend creates a new local ComfyUI backend.
func NewComfyUIBackend(baseURL, outputDir string, httpClient *http.Client) *ComfyUIBackend {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8188"
	}
	if outputDir == "" {
		home, _ := os.UserHomeDir()
		outputDir = filepath.Join(home, ".aris", "outputs")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 180 * time.Second}
	}
	return &ComfyUIBackend{
		baseURL:    strings.TrimRight(baseURL, "/"),
		outputDir:  outputDir,
		httpClient: httpClient,
	}
}

func (c *ComfyUIBackend) Name() string {
	return "comfyui"
}

func (c *ComfyUIBackend) SupportsModels() []string {
	return []string{"local-flux", "local-sdxl", "custom-workflow"}
}

// buildComfyGraph generates a standard text2img workflow graph for ComfyUI.
func (c *ComfyUIBackend) buildComfyGraph(spec *domain.ImageSpec, clientID string) map[string]any {
	prompt := spec.EnhancedPrompt
	if prompt == "" {
		prompt = spec.RawPrompt
	}

	w, h := spec.Width, spec.Height
	if w <= 0 || h <= 0 {
		w, h = spec.AspectRatio.Dimensions(1024)
	}

	seed := spec.Seed
	if seed <= 0 {
		seed = time.Now().UnixNano() % 1000000000
	}

	steps := spec.Steps
	if steps <= 0 {
		steps = 20
	}
	cfg := spec.CFGScale
	if cfg <= 0 {
		cfg = 7.0
	}

	// Standard SDXL / Flux workflow graph
	return map[string]any{
		"3": map[string]any{
			"class_type": "KSampler",
			"inputs": map[string]any{
				"cfg":          cfg,
				"denoise":      1.0,
				"latent_image": []any{"5", 0},
				"model":        []any{"4", 0},
				"negative":     []any{"7", 0},
				"positive":     []any{"6", 0},
				"sampler_name": "euler",
				"scheduler":    "normal",
				"seed":         seed,
				"steps":        steps,
			},
		},
		"4": map[string]any{
			"class_type": "CheckpointLoaderSimple",
			"inputs": map[string]any{
				"ckpt_name": "flux1-schnell.safetensors",
			},
		},
		"5": map[string]any{
			"class_type": "EmptyLatentImage",
			"inputs": map[string]any{
				"batch_size": 1,
				"height":     h,
				"width":      w,
			},
		},
		"6": map[string]any{
			"class_type": "CLIPTextEncode",
			"inputs": map[string]any{
				"clip": []any{"4", 1},
				"text": prompt,
			},
		},
		"7": map[string]any{
			"class_type": "CLIPTextEncode",
			"inputs": map[string]any{
				"clip": []any{"4", 1},
				"text": spec.NegativePrompt,
			},
		},
		"8": map[string]any{
			"class_type": "VAEDecode",
			"inputs": map[string]any{
				"samples": []any{"3", 0},
				"vae":     []any{"4", 2},
			},
		},
		"9": map[string]any{
			"class_type": "SaveImage",
			"inputs": map[string]any{
				"filename_prefix": "ARIS",
				"images":          []any{"8", 0},
			},
		},
	}
}

type comfyPromptReq struct {
	Prompt   map[string]any `json:"prompt"`
	ClientID string         `json:"client_id"`
}

type comfyPromptResp struct {
	PromptID string         `json:"prompt_id"`
	Number   int            `json:"number"`
	Error    map[string]any `json:"error,omitempty"`
}

type comfyHistoryResp map[string]struct {
	Outputs map[string]struct {
		Images []struct {
			Filename  string `json:"filename"`
			Subfolder string `json:"subfolder"`
			Type      string `json:"type"`
		} `json:"images"`
	} `json:"outputs"`
}

func (c *ComfyUIBackend) Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error) {
	clientID := uuid.New().String()
	graph := c.buildComfyGraph(spec, clientID)

	payload := comfyPromptReq{
		Prompt:   graph,
		ClientID: clientID,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal comfyui payload: %w", err)
	}

	start := time.Now()
	promptURL := fmt.Sprintf("%s/prompt", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, promptURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create comfyui request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not connect to ComfyUI at %s (is local ComfyUI running?): %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("comfyui returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var promptResp comfyPromptResp
	if err := json.NewDecoder(resp.Body).Decode(&promptResp); err != nil {
		return nil, fmt.Errorf("decode comfyui prompt response: %w", err)
	}

	promptID := promptResp.PromptID
	if promptID == "" {
		return nil, fmt.Errorf("comfyui did not return prompt_id")
	}

	// Poll ComfyUI /history/{promptID} until complete
	var imageFilename, imageSubfolder, imageType string
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}

		histURL := fmt.Sprintf("%s/history/%s", c.baseURL, promptID)
		hReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, histURL, nil)
		hResp, err := c.httpClient.Do(hReq)
		if err != nil {
			continue
		}

		var hist comfyHistoryResp
		err = json.NewDecoder(hResp.Body).Decode(&hist)
		hResp.Body.Close()
		if err != nil {
			continue
		}

		if data, ok := hist[promptID]; ok {
			for _, output := range data.Outputs {
				if len(output.Images) > 0 {
					imageFilename = output.Images[0].Filename
					imageSubfolder = output.Images[0].Subfolder
					imageType = output.Images[0].Type
					break
				}
			}
			if imageFilename != "" {
				break
			}
		}
	}

	// Download generated image from ComfyUI /view
	viewURL := fmt.Sprintf("%s/view?filename=%s&subfolder=%s&type=%s",
		c.baseURL, imageFilename, imageSubfolder, imageType)
	vReq, err := http.NewRequestWithContext(ctx, http.MethodGet, viewURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create comfyui view request: %w", err)
	}

	vResp, err := c.httpClient.Do(vReq)
	if err != nil {
		return nil, fmt.Errorf("download comfyui rendered image: %w", err)
	}
	defer vResp.Body.Close()

	now := time.Now()
	dayDir := filepath.Join(c.outputDir, now.Format("2006-01-02"))
	_ = os.MkdirAll(dayDir, 0755)

	slug := sanitizeSlug(spec.RawPrompt)
	if len(slug) > 25 {
		slug = slug[:25]
	}
	filename := fmt.Sprintf("aris_%s_%s_%s.png", now.Format("20060102_150405"), slug, uuid.New().String()[:8])
	localPath := filepath.Join(dayDir, filename)

	outFile, err := os.Create(localPath)
	if err != nil {
		return nil, fmt.Errorf("create local image file: %w", err)
	}
	defer outFile.Close()

	written, err := io.Copy(outFile, vResp.Body)
	if err != nil {
		_ = os.Remove(localPath)
		return nil, fmt.Errorf("save image bytes: %w", err)
	}

	return &domain.ImageResult{
		ID:          uuid.New().String(),
		SpecID:      spec.ID,
		LocalPath:   localPath,
		RemoteURL:   viewURL,
		Format:      "png",
		SizeInBytes: written,
		Duration:    time.Since(start),
		Metadata: map[string]any{
			"backend":   "comfyui",
			"prompt_id": promptID,
			"filename":  imageFilename,
		},
	}, nil
}
