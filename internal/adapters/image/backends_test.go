package image_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"aris/internal/adapters/image"
	"aris/internal/core/domain"
)

func TestRegistry_RegisterAndRetrieve(t *testing.T) {
	reg := image.NewRegistry()

	pollinations := image.NewPollinationsBackend()
	fal := image.NewFalAIBackend("fake-key", "", nil)

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

func TestFalAIBackend_GenerateMock(t *testing.T) {
	cdnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("FAL_IMAGE_DATA"))
	}))
	defer cdnServer.Close()

	falServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"images": []map[string]any{
				{
					"url":          cdnServer.URL + "/image.jpg",
					"width":        1024,
					"height":       1024,
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

	tmpDir, _ := os.MkdirTemp("", "aris-fal-test-*")
	defer os.RemoveAll(tmpDir)

	spec := &domain.ImageSpec{
		ID:             "spec-fal-1",
		RawPrompt:      "a futuristic city",
		EnhancedPrompt: "a futuristic cyberpunk city at night with flying cars",
		AspectRatio:    domain.RatioLandscape,
		Width:          1344,
		Height:         768,
		Model:          "fal-ai/flux/schnell",
	}

	// Test auth guard when key is empty
	unauthBackend := image.NewFalAIBackend("", tmpDir, nil)
	_, err := unauthBackend.Generate(context.Background(), spec)
	if err == nil {
		t.Errorf("expected error when FAL_KEY is missing, got nil")
	}
}

func TestOpenAIBackend_GenerateMock(t *testing.T) {
	cdnServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OPENAI_IMAGE_BYTES"))
	}))
	defer cdnServer.Close()

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{
					"url":            cdnServer.URL + "/dalle.png",
					"revised_prompt": "A cinematic shot of a samurai cat",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer apiServer.Close()

	tmpDir, _ := os.MkdirTemp("", "aris-openai-test-*")
	defer os.RemoveAll(tmpDir)

	backend := image.NewOpenAIBackend("test-key", apiServer.URL, tmpDir, apiServer.Client())

	spec := &domain.ImageSpec{
		ID:             "spec-dalle-1",
		RawPrompt:      "samurai cat",
		EnhancedPrompt: "a samurai cat in ancient kyoto",
		AspectRatio:    domain.RatioSquare,
		Width:          1024,
		Height:         1024,
		Model:          "dall-e-3",
	}

	result, err := backend.Generate(context.Background(), spec)
	if err != nil {
		t.Fatalf("OpenAI Generate failed: %v", err)
	}

	if result.LocalPath == "" {
		t.Errorf("expected local path populated")
	}
	content, _ := os.ReadFile(result.LocalPath)
	if string(content) != "OPENAI_IMAGE_BYTES" {
		t.Errorf("unexpected image bytes: %s", string(content))
	}
}
