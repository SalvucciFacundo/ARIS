package services_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aris/internal/adapters/db"
	"aris/internal/adapters/image"
	"aris/internal/adapters/llm"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

func TestAgentService_GenerateWorkflow(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aris-agent-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	sqlDB, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer sqlDB.Close()

	kg := db.NewKnowledgeGraph(sqlDB.DB())
	history := db.NewHistoryStore(sqlDB.DB())
	llmProvider := llm.NewPassthroughProvider()

	reg := image.NewRegistry()
	mockBackend := image.NewPollinationsBackend(
		image.WithOutputDir(tmpDir),
	)
	_ = reg.Register(mockBackend)

	agent := services.NewAgentService(llmProvider, reg, kg, history, nil)
	ctx := context.Background()

	// 1. Add style memory fact
	_, err = agent.LearnFact(ctx, "style:anime", "lineart", "clean thick anime lineart, studio ghibli aesthetic", domain.ScopeStyle, []string{"anime", "ghibli"})
	if err != nil {
		t.Fatalf("LearnFact failed: %v", err)
	}

	// 2. Search Memory
	facts, err := agent.SearchMemory(ctx, "anime", domain.ScopeStyle, 5)
	if err != nil {
		t.Fatalf("SearchMemory failed: %v", err)
	}
	if len(facts) == 0 {
		t.Fatalf("expected to find at least 1 fact")
	}

	// 3. Test Reasoning
	spec, err := llmProvider.ReasonPrompt(ctx, "a peaceful floating island in the sky", facts)
	if err != nil {
		t.Fatalf("ReasonPrompt failed: %v", err)
	}
	if spec.RawPrompt != "a peaceful floating island in the sky" {
		t.Errorf("unexpected raw prompt: %s", spec.RawPrompt)
	}

	// Save generation to history
	res := &domain.ImageResult{
		ID:          "gen-mock-1",
		SpecID:      spec.ID,
		LocalPath:   filepath.Join(tmpDir, "output.png"),
		Format:      "png",
		SizeInBytes: 512,
		Duration:    100 * time.Millisecond,
	}
	_ = history.SaveGeneration(ctx, spec, res)

	historyList, err := agent.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(historyList) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(historyList))
	}
}
