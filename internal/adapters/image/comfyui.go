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
	"strings"
	"time"

	"aris/internal/core/domain"
	"aris/internal/core/ports"
	"aris/pkg/imgutil"

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

// uploadImage uploads an image to ComfyUI /upload/image endpoint and returns the stored filename.
func (c *ComfyUIBackend) uploadImage(ctx context.Context, imagePath string) (string, error) {
	data, _, err := imgutil.LoadAndValidateImage(imagePath, imgutil.MaxImageSize)
	if err != nil {
		return "", fmt.Errorf("load image for ComfyUI upload: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	filename := filepath.Base(imagePath)
	if filename == "" || filename == "." {
		filename = "input_" + uuid.New().String()[:8] + ".png"
	}

	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return "", fmt.Errorf("create multipart form file: %w", err)
	}
	if _, err := part.Write(data); err != nil {
		return "", fmt.Errorf("write form file payload: %w", err)
	}

	_ = writer.WriteField("overwrite", "true")
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	uploadURL := fmt.Sprintf("%s/upload/image", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uploadURL, &body)
	if err != nil {
		return "", fmt.Errorf("create upload request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("comfyui image upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	var uploadResp struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&uploadResp); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}

	if uploadResp.Name == "" {
		uploadResp.Name = filename
	}

	return uploadResp.Name, nil
}

// buildComfyGraph generates a workflow graph for ComfyUI (text2img, img2img, or inpaint).
func (c *ComfyUIBackend) buildComfyGraph(ctx context.Context, spec *domain.ImageSpec, clientID string) (map[string]any, error) {
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

	denoise := 1.0
	if spec.DenoiseStrength > 0.0 {
		denoise = spec.DenoiseStrength
	}

	if spec.IsUpscale() {
		baseName, err := c.uploadImage(ctx, spec.InputImagePath)
		if err != nil {
			return nil, fmt.Errorf("upload upscale base image: %w", err)
		}

		upscalerModel := spec.UpscalerModel
		if upscalerModel == "" {
			upscalerModel = "RealESRGAN_x4plus.pth"
		}

		graph := map[string]any{
			"10": map[string]any{
				"class_type": "LoadImage",
				"inputs": map[string]any{
					"image": baseName,
				},
			},
			"13": map[string]any{
				"class_type": "UpscaleModelLoader",
				"inputs": map[string]any{
					"model_name": upscalerModel,
				},
			},
			"14": map[string]any{
				"class_type": "ImageUpscaleWithModel",
				"inputs": map[string]any{
					"image":         []any{"10", 0},
					"upscale_model": []any{"13", 0},
				},
			},
		}

		if spec.RestoreFaces {
			graph["15"] = map[string]any{
				"class_type": "FaceRestoreCFWithModel",
				"inputs": map[string]any{
					"fidelity": spec.FaceFidelity,
					"image":    []any{"14", 0},
				},
			}
			graph["9"] = map[string]any{
				"class_type": "SaveImage",
				"inputs": map[string]any{
					"filename_prefix": "ARIS_UPSCALE",
					"images":          []any{"15", 0},
				},
			}
		} else {
			graph["9"] = map[string]any{
				"class_type": "SaveImage",
				"inputs": map[string]any{
					"filename_prefix": "ARIS_UPSCALE",
					"images":          []any{"14", 0},
				},
			}
		}

		return graph, nil
	}

	if spec.IsInpaint() {
		baseName, err := c.uploadImage(ctx, spec.InputImagePath)
		if err != nil {
			return nil, fmt.Errorf("upload inpaint base image: %w", err)
		}
		maskName, err := c.uploadImage(ctx, spec.MaskImagePath)
		if err != nil {
			return nil, fmt.Errorf("upload inpaint mask image: %w", err)
		}

		return map[string]any{
			"10": map[string]any{
				"class_type": "LoadImage",
				"inputs": map[string]any{
					"image": baseName,
				},
			},
			"11": map[string]any{
				"class_type": "LoadImage",
				"inputs": map[string]any{
					"image": maskName,
				},
			},
			"12": map[string]any{
				"class_type": "VAEEncodeForInpaint",
				"inputs": map[string]any{
					"grow_mask_by": 6,
					"mask":         []any{"11", 0},
					"pixels":       []any{"10", 0},
					"vae":          []any{"4", 2},
				},
			},
			"3": map[string]any{
				"class_type": "KSampler",
				"inputs": map[string]any{
					"cfg":          cfg,
					"denoise":      denoise,
					"latent_image": []any{"12", 0},
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
		}, nil
	}

	if spec.IsImg2Img() {
		baseName, err := c.uploadImage(ctx, spec.InputImagePath)
		if err != nil {
			return nil, fmt.Errorf("upload img2img base image: %w", err)
		}

		return map[string]any{
			"10": map[string]any{
				"class_type": "LoadImage",
				"inputs": map[string]any{
					"image": baseName,
				},
			},
			"12": map[string]any{
				"class_type": "VAEEncode",
				"inputs": map[string]any{
					"pixels": []any{"10", 0},
					"vae":    []any{"4", 2},
				},
			},
			"3": map[string]any{
				"class_type": "KSampler",
				"inputs": map[string]any{
					"cfg":          cfg,
					"denoise":      denoise,
					"latent_image": []any{"12", 0},
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
		}, nil
	}

	// Standard text2img
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
	}, nil
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
	spec.ApplyDefaults()
	if err := spec.Validate(); err != nil {
		return nil, fmt.Errorf("invalid spec: %w", err)
	}

	clientID := uuid.New().String()
	graph, err := c.buildComfyGraph(ctx, spec, clientID)
	if err != nil {
		return nil, fmt.Errorf("build comfyui graph: %w", err)
	}

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
		case <-time.After(50 * time.Millisecond):
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
