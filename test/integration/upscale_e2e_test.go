package integration_test

import (
	"context"
	"encoding/json"
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

func TestUpscale_E2E_Pipeline(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "outputs")
	_ = os.MkdirAll(outDir, 0755)

	inputImgPath := filepath.Join(tmpDir, "input_portrait.png")
	createE2EPNG(t, inputImgPath, 256, 256)

	// Mock Fal.ai Upscale Server
	var capturedEndpoint string
	var capturedPayload map[string]any

	cdnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("E2E_UPSCALED_FAL_IMAGE"))
	}))
	defer cdnServer.Close()

	falServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedEndpoint = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&capturedPayload)

		resp := map[string]any{
			"images": []map[string]any{
				{
					"url":          cdnServer.URL + "/upscaled_output.jpg",
					"width":        1024,
					"height":       1024,
					"content_type": "image/jpeg",
				},
			},
			"timings": map[string]any{"inference": 1.1},
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

	t.Run("super resolution with face restoration", func(t *testing.T) {
		opts := services.GenerateOptions{
			InputImage:    inputImgPath,
			Mode:          domain.ModeUpscale,
			ScaleFactor:   4,
			RestoreFaces:  true,
			FaceFidelity:  0.85,
			UpscalerModel: "fal-ai/esrgan",
			Backend:       "falai",
		}

		spec, res, err := agent.Generate(ctx, "upscale portrait with face restore", opts)
		if err != nil {
			t.Fatalf("Generate upscale failed: %v", err)
		}
		if spec.Mode != domain.ModeUpscale {
			t.Errorf("expected ModeUpscale, got %s", spec.Mode)
		}
		if spec.ScaleFactor != 4 {
			t.Errorf("expected ScaleFactor 4, got %d", spec.ScaleFactor)
		}
		if !spec.RestoreFaces {
			t.Errorf("expected RestoreFaces true")
		}
		if spec.FaceFidelity != 0.85 {
			t.Errorf("expected FaceFidelity 0.85, got %f", spec.FaceFidelity)
		}
		if !strings.Contains(capturedEndpoint, "esrgan") {
			t.Errorf("expected falai esrgan endpoint, got %s", capturedEndpoint)
		}
		if capturedPayload["image_url"] == "" {
			t.Errorf("expected image_url in upscale payload")
		}
		if res.LocalPath == "" {
			t.Fatalf("expected output image saved locally")
		}
		data, err := os.ReadFile(res.LocalPath)
		if err != nil || string(data) != "E2E_UPSCALED_FAL_IMAGE" {
			t.Errorf("unexpected image content on disk")
		}
	})
}
