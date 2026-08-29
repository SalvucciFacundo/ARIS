package image_test

import (
	"bytes"
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
	"aris/internal/core/domain"
)

func TestRegistry_RegisterAndRetrieve(t *testing.T) {
	reg := imgadapter.NewRegistry()

	pollinations := imgadapter.NewPollinationsBackend()
	fal := imgadapter.NewFalAIBackend("fake-key", "", nil)

	if err := reg.Register(pollinations); err != nil {
		t.Fatalf("Register pollinations failed: %v", err)
	}
	if err := reg.Register(fal); err != nil {
		t.Fatalf("Register fal failed: %v", err)
	}

	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("expected 2 backends, got %d", len(list))
	}

	b, err := reg.Get("pollinations")
	if err != nil {
		t.Fatalf("Get pollinations failed: %v", err)
	}
	if b.Name() != "pollinations" {
		t.Errorf("expected name pollinations, got %s", b.Name())
	}

	if reg.GetDefault() == nil || reg.GetDefault().Name() != "pollinations" {
		t.Errorf("expected default backend pollinations")
	}

	_ = reg.SetDefault("falai")
	if reg.GetDefault().Name() != "falai" {
		t.Errorf("expected default backend falai after SetDefault")
	}
}

func createTestPNGFile(t *testing.T, dir, name string, w, h int) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 120, G: 200, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	filePath := filepath.Join(dir, name)
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test png: %v", err)
	}
	return filePath
}

func TestFalAIBackend_Img2ImgAndInpaintPayload(t *testing.T) {
	tmpDir := t.TempDir()
	baseImg := createTestPNGFile(t, tmpDir, "base.png", 64, 64)
	maskImg := createTestPNGFile(t, tmpDir, "mask.png", 64, 64)

	var capturedPath string
	var capturedPayload map[string]any

	cdnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("FAL_EDITED_IMAGE"))
	}))
	defer cdnServer.Close()

	falServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
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
				"inference": 0.85,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer falServer.Close()

	backend := imgadapter.NewFalAIBackend("fake-fal-key", tmpDir, falServer.Client())
	// Test Img2Img
	specI2I := &domain.ImageSpec{
		ID:              "fal-i2i",
		RawPrompt:       "cyberpunk neon overhaul",
		EnhancedPrompt:  "cyberpunk neon overhaul cinematic",
		Mode:            domain.ModeImg2Img,
		InputImagePath:  baseImg,
		DenoiseStrength: 0.65,
	}

	// Swap base URL via test or reflection if supported, or via custom constructor/internal helper
	// Let's test with the fal backend
	result, err := backend.GenerateWithBaseURL(context.Background(), specI2I, falServer.URL)
	if err != nil {
		t.Fatalf("Fal img2img failed: %v", err)
	}
	if result == nil || result.LocalPath == "" {
		t.Fatalf("expected valid result from Fal img2img")
	}
	if !strings.Contains(capturedPath, "image-to-image") {
		t.Errorf("expected endpoint to contain image-to-image, got %s", capturedPath)
	}
	if capturedPayload["image_url"] == "" {
		t.Errorf("expected image_url in payload")
	}
	if strength, ok := capturedPayload["strength"].(float64); !ok || strength != 0.65 {
		t.Errorf("expected strength 0.65, got %v", capturedPayload["strength"])
	}

	// Test Inpaint
	specInpaint := &domain.ImageSpec{
		ID:              "fal-inpaint",
		RawPrompt:       "remove background",
		EnhancedPrompt:  "remove background with transparent blend",
		Mode:            domain.ModeInpaint,
		InputImagePath:  baseImg,
		MaskImagePath:   maskImg,
		DenoiseStrength: 0.80,
	}

	resultInpaint, err := backend.GenerateWithBaseURL(context.Background(), specInpaint, falServer.URL)
	if err != nil {
		t.Fatalf("Fal inpaint failed: %v", err)
	}
	if resultInpaint == nil {
		t.Fatalf("expected valid result from Fal inpaint")
	}
	if !strings.Contains(capturedPath, "inpaint") {
		t.Errorf("expected inpainting endpoint, got %s", capturedPath)
	}
	if capturedPayload["mask_url"] == "" {
		t.Errorf("expected mask_url in inpaint payload")
	}
}

func TestComfyUIBackend_Img2ImgAndInpaint(t *testing.T) {
	tmpDir := t.TempDir()
	baseImg := createTestPNGFile(t, tmpDir, "base.png", 64, 64)
	maskImg := createTestPNGFile(t, tmpDir, "mask.png", 64, 64)

	var uploadedImages []string
	var promptGraph map[string]any

	comfyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/upload/image"):
			uploadedImages = append(uploadedImages, "uploaded_file.png")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "uploaded_file.png"})
		case strings.HasPrefix(r.URL.Path, "/prompt"):
			var body struct {
				Prompt map[string]any `json:"prompt"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			promptGraph = body.Prompt
			_ = json.NewEncoder(w).Encode(map[string]any{"prompt_id": "comfy-123"})
		case strings.HasPrefix(r.URL.Path, "/history/comfy-123"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"comfy-123": map[string]any{
					"outputs": map[string]any{
						"9": map[string]any{
							"images": []map[string]any{
								{"filename": "out.png", "subfolder": "", "type": "output"},
							},
						},
					},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/view"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("COMFY_RENDERED_BYTES"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer comfyServer.Close()

	backend := imgadapter.NewComfyUIBackend(comfyServer.URL, tmpDir, comfyServer.Client())

	// Test inpainting workflow graph construction
	specInpaint := &domain.ImageSpec{
		ID:              "comfy-inp",
		RawPrompt:       "fix face",
		Mode:            domain.ModeInpaint,
		InputImagePath:  baseImg,
		MaskImagePath:   maskImg,
		DenoiseStrength: 0.75,
	}

	result, err := backend.Generate(context.Background(), specInpaint)
	if err != nil {
		t.Fatalf("ComfyUI inpaint failed: %v", err)
	}
	if result == nil || result.LocalPath == "" {
		t.Fatalf("expected valid result from comfyui")
	}

	// Verify node graph has inpainting nodes
	hasInpaintNode := false
	for _, node := range promptGraph {
		if nodeMap, ok := node.(map[string]any); ok {
			if classType, ok := nodeMap["class_type"].(string); ok {
				if strings.Contains(classType, "Inpaint") || classType == "LoadImage" {
					hasInpaintNode = true
					break
				}
			}
		}
	}
	if !hasInpaintNode {
		t.Errorf("expected ComfyUI inpaint graph to contain inpainting or LoadImage nodes")
	}
}

func TestOpenAIBackend_EditMultipart(t *testing.T) {
	tmpDir := t.TempDir()
	baseImg := createTestPNGFile(t, tmpDir, "base.png", 64, 64)
	maskImg := createTestPNGFile(t, tmpDir, "mask.png", 64, 64)

	var capturedEndpoint string
	var isMultipart bool

	cdnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OPENAI_EDIT_OUTPUT"))
	}))
	defer cdnServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedEndpoint = r.URL.Path
		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			isMultipart = true
		}

		resp := map[string]any{
			"data": []map[string]any{
				{
					"url":            cdnServer.URL + "/dalle2_edit.png",
					"revised_prompt": "An edited cat",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	backend := imgadapter.NewOpenAIBackend("test-key", apiServer.URL, tmpDir, apiServer.Client())

	spec := &domain.ImageSpec{
		ID:              "dalle-edit",
		RawPrompt:       "add sunglasses",
		Mode:            domain.ModeInpaint,
		InputImagePath:  baseImg,
		MaskImagePath:   maskImg,
		DenoiseStrength: 0.8,
	}

	result, err := backend.Generate(context.Background(), spec)
	if err != nil {
		t.Fatalf("OpenAI edit failed: %v", err)
	}
	if result == nil {
		t.Fatalf("expected valid result from openai edit")
	}
	if !strings.Contains(capturedEndpoint, "/images/edits") {
		t.Errorf("expected /images/edits endpoint, got %s", capturedEndpoint)
	}
	if !isMultipart {
		t.Errorf("expected multipart/form-data request")
	}
}

func TestFalAIBackend_UpscaleAndFaceRestorePayload(t *testing.T) {
	tmpDir := t.TempDir()
	baseImg := createTestPNGFile(t, tmpDir, "base.png", 64, 64)

	var capturedPath string
	var capturedPayload map[string]any

	cdnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("FAL_UPSCALED_IMAGE"))
	}))
	defer cdnServer.Close()

	falServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		bodyBytes, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(bodyBytes, &capturedPayload)

		resp := map[string]any{
			"images": []map[string]any{
				{
					"url":          cdnServer.URL + "/upscaled.jpg",
					"width":        256,
					"height":       256,
					"content_type": "image/jpeg",
				},
			},
			"timings": map[string]any{
				"inference": 1.25,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer falServer.Close()

	backend := imgadapter.NewFalAIBackend("fake-fal-key", tmpDir, falServer.Client())

	specUpscale := &domain.ImageSpec{
		ID:             "fal-upscale-test",
		Mode:           domain.ModeUpscale,
		InputImagePath: baseImg,
		ScaleFactor:    4,
		RestoreFaces:   true,
		FaceFidelity:   0.8,
	}

	result, err := backend.GenerateWithBaseURL(context.Background(), specUpscale, falServer.URL)
	if err != nil {
		t.Fatalf("Fal upscale failed: %v", err)
	}
	if result == nil || result.LocalPath == "" {
		t.Fatalf("expected valid result from Fal upscale")
	}

	if !strings.Contains(capturedPath, "esrgan") && !strings.Contains(capturedPath, "aura-sr") {
		t.Errorf("expected upscale endpoint (esrgan or aura-sr), got %s", capturedPath)
	}
	if capturedPayload["image_url"] == "" {
		t.Errorf("expected image_url in upscale payload")
	}
	scaleVal, ok := capturedPayload["scale"].(float64)
	if !ok || int(scaleVal) != 4 {
		t.Errorf("expected scale 4 in payload, got %v", capturedPayload["scale"])
	}
	if restoreFaces, ok := capturedPayload["face_enhancer"].(bool); !ok || !restoreFaces {
		if restoreAlt, ok := capturedPayload["restore_faces"].(bool); !ok || !restoreAlt {
			t.Errorf("expected face enhancement toggle in payload")
		}
	}
}

func TestComfyUIBackend_UpscaleAndFaceRestoreGraph(t *testing.T) {
	tmpDir := t.TempDir()
	baseImg := createTestPNGFile(t, tmpDir, "base.png", 64, 64)

	var uploadedImages []string
	var promptGraph map[string]any

	comfyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/upload/image"):
			uploadedImages = append(uploadedImages, "uploaded_file.png")
			_ = json.NewEncoder(w).Encode(map[string]any{"name": "uploaded_file.png"})
		case strings.HasPrefix(r.URL.Path, "/prompt"):
			var body struct {
				Prompt map[string]any `json:"prompt"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			promptGraph = body.Prompt
			_ = json.NewEncoder(w).Encode(map[string]any{"prompt_id": "comfy-upscale-123"})
		case strings.HasPrefix(r.URL.Path, "/history/comfy-upscale-123"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"comfy-upscale-123": map[string]any{
					"outputs": map[string]any{
						"9": map[string]any{
							"images": []map[string]any{
								{"filename": "upscaled.png", "subfolder": "", "type": "output"},
							},
						},
					},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/view"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("COMFY_UPSCALED_BYTES"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer comfyServer.Close()

	backend := imgadapter.NewComfyUIBackend(comfyServer.URL, tmpDir, comfyServer.Client())

	// 1. With Face Restoration
	specWithFace := &domain.ImageSpec{
		ID:             "comfy-upscale-face",
		Mode:           domain.ModeUpscale,
		InputImagePath: baseImg,
		ScaleFactor:    4,
		RestoreFaces:   true,
		FaceFidelity:   0.85,
	}

	result, err := backend.Generate(context.Background(), specWithFace)
	if err != nil {
		t.Fatalf("ComfyUI upscale with face restore failed: %v", err)
	}
	if result == nil || result.LocalPath == "" {
		t.Fatalf("expected valid result from ComfyUI upscale")
	}

	hasUpscaleNode := false
	hasFaceRestoreNode := false
	for _, node := range promptGraph {
		if nodeMap, ok := node.(map[string]any); ok {
			if classType, ok := nodeMap["class_type"].(string); ok {
				if strings.Contains(classType, "Upscale") || strings.Contains(classType, "UpscaleModelLoader") {
					hasUpscaleNode = true
				}
				if strings.Contains(classType, "FaceRestore") || strings.Contains(classType, "CodeFormer") || strings.Contains(classType, "GFPGAN") {
					hasFaceRestoreNode = true
				}
			}
		}
	}
	if !hasUpscaleNode {
		t.Errorf("expected upscale node in workflow graph")
	}
	if !hasFaceRestoreNode {
		t.Errorf("expected face restore node in workflow graph when RestoreFaces is true")
	}

	// 2. Without Face Restoration
	specWithoutFace := &domain.ImageSpec{
		ID:             "comfy-upscale-noface",
		Mode:           domain.ModeUpscale,
		InputImagePath: baseImg,
		ScaleFactor:    2,
		RestoreFaces:   false,
	}

	_, err = backend.Generate(context.Background(), specWithoutFace)
	if err != nil {
		t.Fatalf("ComfyUI upscale without face restore failed: %v", err)
	}

	hasFaceRestoreNode2 := false
	for _, node := range promptGraph {
		if nodeMap, ok := node.(map[string]any); ok {
			if classType, ok := nodeMap["class_type"].(string); ok {
				if strings.Contains(classType, "FaceRestore") || strings.Contains(classType, "CodeFormer") || strings.Contains(classType, "GFPGAN") {
					hasFaceRestoreNode2 = true
				}
			}
		}
	}
	if hasFaceRestoreNode2 {
		t.Errorf("expected no face restore node when RestoreFaces is false")
	}
}

func TestOpenAIBackend_UpscaleUnsupported(t *testing.T) {
	backend := imgadapter.NewOpenAIBackend("test-key", "https://api.openai.com/v1", t.TempDir(), nil)

	spec := &domain.ImageSpec{
		Mode:           domain.ModeUpscale,
		InputImagePath: "base.png",
		ScaleFactor:    4,
	}

	_, err := backend.Generate(context.Background(), spec)
	if err == nil {
		t.Fatalf("expected error for unsupported OpenAI upscaling, got nil")
	}
	if !strings.Contains(err.Error(), "does not support") && !strings.Contains(err.Error(), "upscaling") {
		t.Errorf("expected clear unsupported upscaling error, got: %v", err)
	}
}

func TestPollinationsBackend_UpscaleUnsupported(t *testing.T) {
	backend := imgadapter.NewPollinationsBackend()

	spec := &domain.ImageSpec{
		Mode:           domain.ModeUpscale,
		InputImagePath: "base.png",
		ScaleFactor:    4,
	}

	_, err := backend.Generate(context.Background(), spec)
	if err == nil {
		t.Fatalf("expected error for unsupported Pollinations upscaling, got nil")
	}
	if !strings.Contains(err.Error(), "does not support") && !strings.Contains(err.Error(), "upscaling") {
		t.Errorf("expected clear unsupported upscaling error, got: %v", err)
	}
}

func TestPollinationsBackend_InpaintUnsupported(t *testing.T) {
	backend := imgadapter.NewPollinationsBackend()

	spec := &domain.ImageSpec{
		RawPrompt:      "inpaint attempt",
		Mode:           domain.ModeInpaint,
		InputImagePath: "base.png",
		MaskImagePath:  "mask.png",
	}

	_, err := backend.Generate(context.Background(), spec)
	if err == nil {
		t.Fatalf("expected error for unsupported pollinations inpainting, got nil")
	}
	if !strings.Contains(err.Error(), "does not support") && !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("expected unsupported capability error, got: %v", err)
	}
}
