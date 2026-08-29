package services_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aris/internal/adapters/image"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

type mockCritic struct {
	scores []float64
	idx    int
}

func (m *mockCritic) Name() string {
	return "mock-critic"
}

func (m *mockCritic) Evaluate(ctx context.Context, imagePath string, spec *domain.ImageSpec) (float64, string, error) {
	if m.idx >= len(m.scores) {
		return 0.90, "Good", nil
	}
	s := m.scores[m.idx]
	m.idx++
	return s, "Mock critique", nil
}

func TestCriticService_SelfHealing(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "aris-critic-test-*")
	defer os.RemoveAll(tmpDir)

	backend := image.NewPollinationsBackend(image.WithOutputDir(tmpDir))

	// Mock critic: first returns 0.40 (triggering heal), second returns 0.88
	critic := &mockCritic{scores: []float64{0.40, 0.88}}
	svc := services.NewCriticService(critic, 0.60, true)

	spec := &domain.ImageSpec{
		ID:             "spec-test",
		RawPrompt:      "a cyberpunk city",
		EnhancedPrompt: "a cyberpunk city at night",
		AspectRatio:    domain.RatioSquare,
		Width:          512,
		Height:         512,
		Seed:           42,
		Backend:        "pollinations",
		Model:          "flux",
	}

	initialResult := &domain.ImageResult{
		ID:          "res-1",
		SpecID:      "spec-test",
		LocalPath:   filepath.Join(tmpDir, "dummy.jpg"),
		Duration:    100 * time.Millisecond,
		SizeInBytes: 100,
	}
	_ = os.WriteFile(initialResult.LocalPath, []byte("IMG"), 0644)

	finalSpec, finalResult, err := svc.InspectAndHeal(context.Background(), backend, spec, initialResult)
	if err != nil {
		t.Fatalf("InspectAndHeal failed: %v", err)
	}

	// Verify that self-healing adjusted prompt and changed seed
	if finalSpec.Seed == 42 {
		t.Errorf("expected seed to change during self-healing re-roll")
	}
	if finalResult.Metadata["self_healed"] != true {
		t.Errorf("expected self_healed metadata flag to be true")
	}
}
