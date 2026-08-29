package integration_test

import (
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	imgadapter "aris/internal/adapters/image"
	"aris/internal/adapters/llm"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

func createTestPNG(t *testing.T, path string, w, h int) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create image file: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode PNG: %v", err)
	}
}

func TestLoRAAndControlNet_E2E_ComfyUI(t *testing.T) {
	tmpDir := t.TempDir()
	cnetImgPath := filepath.Join(tmpDir, "pose_ref.png")
	createTestPNG(t, cnetImgPath, 64, 64)

	var promptGraph map[string]any

	comfyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/upload/image"):
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "uploaded_cnet_pose.png"})
		case strings.HasPrefix(r.URL.Path, "/prompt"):
			var body struct {
				Prompt map[string]any `json:"prompt"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			promptGraph = body.Prompt
			_ = json.NewEncoder(w).Encode(map[string]any{"prompt_id": "comfy-e2e-1"})
		case strings.HasPrefix(r.URL.Path, "/history/comfy-e2e-1"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"comfy-e2e-1": map[string]any{
					"outputs": map[string]any{
						"9": map[string]any{
							"images": []map[string]any{
								{"filename": "out_e2e.png", "subfolder": "", "type": "output"},
							},
						},
					},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/view"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("COMFY_E2E_RENDERED_IMAGE"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer comfyServer.Close()

	reg := imgadapter.NewRegistry()
	comfyBackend := imgadapter.NewComfyUIBackend(comfyServer.URL, tmpDir, comfyServer.Client())
	_ = reg.Register(comfyBackend)
	_ = reg.SetDefault("comfyui")

	passthroughLLM := llm.NewPassthroughProvider()
	agent := services.NewAgentService(passthroughLLM, reg, nil, nil, nil)

	rawPrompt := "cyberpunk warrior <lora:neon_cyber:0.85> in neon city"
	opts := services.GenerateOptions{
		Backend: "comfyui",
		LoRAs: []domain.LoRAConfig{
			{Name: "detail_booster", Scale: 0.70},
		},
		ControlNets: []domain.ControlNetConfig{
			{Type: "canny", Strength: 0.80, ReferenceImage: cnetImgPath},
		},
	}

	spec, result, err := agent.Generate(context.Background(), rawPrompt, opts)
	if err != nil {
		t.Fatalf("Agent.Generate failed: %v", err)
	}

	if spec == nil || result == nil {
		t.Fatalf("expected non-nil spec and result")
	}

	// Verify inline LoRA and flag LoRA were combined
	if len(spec.LoRAs) != 2 {
		t.Fatalf("expected 2 LoRAs, got %d", len(spec.LoRAs))
	}
	if len(spec.ControlNets) != 1 {
		t.Fatalf("expected 1 ControlNet, got %d", len(spec.ControlNets))
	}

	// Verify graph contains LoraLoader and ControlNet nodes
	loraNodes := 0
	cnetNodes := 0
	for _, node := range promptGraph {
		if nodeMap, ok := node.(map[string]any); ok {
			classType, _ := nodeMap["class_type"].(string)
			if classType == "LoraLoader" {
				loraNodes++
			}
			if classType == "ApplyControlNet" {
				cnetNodes++
			}
		}
	}
	if loraNodes != 2 {
		t.Errorf("expected 2 LoraLoader nodes in ComfyUI graph, got %d", loraNodes)
	}
	if cnetNodes != 1 {
		t.Errorf("expected 1 ApplyControlNet node in ComfyUI graph, got %d", cnetNodes)
	}
}

func TestLoRAAndControlNet_E2E_FalAI(t *testing.T) {
	tmpDir := t.TempDir()
	cnetImgPath := filepath.Join(tmpDir, "cnet_input.png")
	createTestPNG(t, cnetImgPath, 64, 64)

	var capturedPayload map[string]any

	cdnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("FAL_E2E_OUTPUT_IMAGE"))
	}))
	defer cdnServer.Close()

	falServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &capturedPayload)

		resp := map[string]any{
			"images": []map[string]any{
				{
					"url":          cdnServer.URL + "/output.jpg",
					"width":        64,
					"height":       64,
					"content_type": "image/jpeg",
				},
			},
			"timings": map[string]any{
				"inference": 1.05,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer falServer.Close()

	reg := imgadapter.NewRegistry()
	falBackend := imgadapter.NewFalAIBackend("fake-key", tmpDir, falServer.Client())
	falBackend.SetBaseURL(falServer.URL)
	_ = reg.Register(falBackend)
	_ = reg.SetDefault("falai")

	passthroughLLM := llm.NewPassthroughProvider()
	agent := services.NewAgentService(passthroughLLM, reg, nil, nil, nil)

	rawPrompt := "futuristic skyline <lora:architectural_style:1.1>"
	opts := services.GenerateOptions{
		Backend: "falai",
		ControlNets: []domain.ControlNetConfig{
			{Type: "canny", Strength: 0.90, ReferenceImage: cnetImgPath},
		},
	}

	spec, result, err := agent.Generate(context.Background(), rawPrompt, opts)
	if err != nil {
		t.Fatalf("Agent.Generate failed: %v", err)
	}

	if spec == nil || result == nil {
		t.Fatalf("expected non-nil spec and result")
	}

	if len(spec.LoRAs) != 1 {
		t.Fatalf("expected 1 LoRA, got %d", len(spec.LoRAs))
	}
	if spec.LoRAs[0].Name != "architectural_style" {
		t.Errorf("expected LoRA name architectural_style, got %s", spec.LoRAs[0].Name)
	}

	// Check payload
	loras, ok := capturedPayload["loras"].([]any)
	if !ok || len(loras) != 1 {
		t.Fatalf("expected 1 lora in fal payload, got %v", capturedPayload["loras"])
	}
	cnet, ok := capturedPayload["controlnets"].([]any)
	if !ok || len(cnet) != 1 {
		t.Fatalf("expected 1 controlnet in fal payload, got %v", capturedPayload["controlnets"])
	}
}
