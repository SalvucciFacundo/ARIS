package image_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"aris/internal/adapters/image"
	"aris/internal/core/domain"
)

func TestPollinationsBackend_BuildURL(t *testing.T) {
	backend := image.NewPollinationsBackend()

	spec := &domain.ImageSpec{
		RawPrompt:      "samurai cat",
		EnhancedPrompt: "a majestic cyberpunk samurai cat in Tokyo",
		NegativePrompt: "blurry, low quality",
		AspectRatio:    domain.RatioLandscape,
		Width:          1344,
		Height:         768,
		Seed:           42,
		Model:          "flux",
	}

	urlStr := backend.BuildURL(spec)

	if !strings.Contains(urlStr, "https://image.pollinations.ai/prompt/") {
		t.Errorf("expected pollinations base url in %s", urlStr)
	}
	if !strings.Contains(urlStr, "width=1344") {
		t.Errorf("expected width=1344 in %s", urlStr)
	}
	if !strings.Contains(urlStr, "height=768") {
		t.Errorf("expected height=768 in %s", urlStr)
	}
	if !strings.Contains(urlStr, "seed=42") {
		t.Errorf("expected seed=42 in %s", urlStr)
	}
	if !strings.Contains(urlStr, "model=flux") {
		t.Errorf("expected model=flux in %s", urlStr)
	}
	if !strings.Contains(urlStr, "nologo=true") {
		t.Errorf("expected nologo=true in %s", urlStr)
	}
	if !strings.Contains(urlStr, "negative=blurry%2C+low+quality") && !strings.Contains(urlStr, "negative=blurry%2C%20low%20quality") {
		t.Errorf("expected encoded negative prompt in %s", urlStr)
	}
}

func TestPollinationsBackend_GenerateMock(t *testing.T) {
	fakeImageData := []byte("FAKE_JPEG_IMAGE_DATA_1234567890")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fakeImageData)
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "aris-image-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	backend := image.NewPollinationsBackend(
		image.WithBaseURL(server.URL),
		image.WithOutputDir(tmpDir),
		image.WithHTTPClient(server.Client()),
	)

	spec := &domain.ImageSpec{
		ID:             "spec-test-1",
		RawPrompt:      "a dog on the moon",
		EnhancedPrompt: "a golden retriever on the lunar surface",
		AspectRatio:    domain.RatioSquare,
		Width:          1024,
		Height:         1024,
		Model:          "flux",
		Seed:           100,
	}

	result, err := backend.Generate(context.Background(), spec)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.LocalPath == "" {
		t.Errorf("expected local path to be populated")
	}

	data, err := os.ReadFile(result.LocalPath)
	if err != nil {
		t.Fatalf("failed to read generated image file: %v", err)
	}
	if string(data) != string(fakeImageData) {
		t.Errorf("expected image data %q, got %q", string(fakeImageData), string(data))
	}
}
