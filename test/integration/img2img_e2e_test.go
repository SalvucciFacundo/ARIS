package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	imgadapter "aris/internal/adapters/image"
	"aris/internal/adapters/llm"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

func createE2EPNG(t *testing.T, path string, w, h int) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 50, G: 120, B: 220, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test png: %v", err)
	}
}

func TestImg2Img_E2E_Pipeline(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "outputs")
	_ = os.MkdirAll(outDir, 0755)

	baseImgPath := filepath.Join(tmpDir, "input_base.png")
	maskImgPath := filepath.Join(tmpDir, "input_mask.png")
	createE2EPNG(t, baseImgPath, 256, 256)
	createE2EPNG(t, maskImgPath, 256, 256)

	// Mock Fal.ai Server
	var capturedEndpoint string
	var capturedPayload map[string]any

	cdnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("E2E_RENDERED_FAL_IMAGE"))
	}))
	defer cdnServer.Close()

	falServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedEndpoint = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&capturedPayload)

		resp := map[string]any{
			"images": []map[string]any{
				{
					"url":          cdnServer.URL + "/output.jpg",
					"width":        256,
					"height":       256,
					"content_type": "image/jpeg",
				},
			},
			"timings": map[string]any{"inference": 0.5},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer falServer.Close()

	falBackend := imgadapter.NewFalAIBackend("e2e-fal-key", outDir, falServer.Client())
	falBackend.SetBaseURL(falServer.URL)

	reg := imgadapter.NewRegistry()
	_ = reg.Register(falBackend)
	_ = reg.SetDefault("falai")

	llmProvider := llm.NewPassthroughProvider()
	agent := services.NewAgentService(llmProvider, reg, nil, nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Test Img2Img transformation flow
	t.Run("img2img execution", func(t *testing.T) {
		opts := services.GenerateOptions{
			InputImage:      baseImgPath,
			DenoiseStrength: 0.65,
			Backend:         "falai",
			Mode:            domain.ModeImg2Img,
		}

		spec, res, err := agent.Generate(ctx, "cyberpunk aesthetic transformation", opts)
		if err != nil {
			t.Fatalf("Generate img2img failed: %v", err)
		}
		if spec.Mode != domain.ModeImg2Img {
			t.Errorf("expected ModeImg2Img, got %s", spec.Mode)
		}
		if !strings.Contains(capturedEndpoint, "image-to-image") {
			t.Errorf("expected falai image-to-image endpoint, got %s", capturedEndpoint)
		}
		if res.LocalPath == "" {
			t.Fatalf("expected output image saved locally")
		}
		data, err := os.ReadFile(res.LocalPath)
		if err != nil || string(data) != "E2E_RENDERED_FAL_IMAGE" {
			t.Errorf("unexpected image content on disk")
		}
	})

	// 2. Test Inpainting flow with mask
	t.Run("inpaint execution with mask", func(t *testing.T) {
		opts := services.GenerateOptions{
			InputImage:      baseImgPath,
			MaskImage:       maskImgPath,
			DenoiseStrength: 0.80,
			Backend:         "falai",
			Mode:            domain.ModeInpaint,
		}

		spec, res, err := agent.Generate(ctx, "replace background with neon city", opts)
		if err != nil {
			t.Fatalf("Generate inpaint failed: %v", err)
		}
		if spec.Mode != domain.ModeInpaint {
			t.Errorf("expected ModeInpaint, got %s", spec.Mode)
		}
		if !strings.Contains(capturedEndpoint, "inpainting") {
			t.Errorf("expected falai inpainting endpoint, got %s", capturedEndpoint)
		}
		if capturedPayload["mask_url"] == "" {
			t.Errorf("expected mask_url in inpaint payload")
		}
		if res.LocalPath == "" {
			t.Fatalf("expected inpaint output saved locally")
		}
	})
}
