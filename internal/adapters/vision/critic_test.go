package vision_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"aris/internal/adapters/vision"
	"aris/internal/core/domain"
)

func TestVisionClient_EvaluateMock(t *testing.T) {
	mockResp := map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"content": `{"score": 0.92, "adherence": "High - Samurai cat with sharp armor", "defects": "None", "suggested_fix": ""}`,
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	tmpDir, _ := os.MkdirTemp("", "aris-vision-test-*")
	defer os.RemoveAll(tmpDir)

	imgPath := filepath.Join(tmpDir, "sample.jpg")
	_ = os.WriteFile(imgPath, []byte("FAKE_IMAGE_BYTES"), 0644)

	client := vision.NewVisionClient("mock-vision", "test-key", server.URL, "gpt-4o-mini", server.Client())

	spec := &domain.ImageSpec{
		ID:             "spec-vis-1",
		RawPrompt:      "a samurai cat",
		EnhancedPrompt: "a cyberpunk samurai cat in neo tokyo",
		AspectRatio:    domain.RatioSquare,
	}

	score, critique, err := client.Evaluate(context.Background(), imgPath, spec)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if score != 0.92 {
		t.Errorf("expected score 0.92, got %f", score)
	}
	if critique == "" {
		t.Errorf("expected non-empty critique summary")
	}
}
