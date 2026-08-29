package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"aris/internal/adapters/db"
	"aris/internal/core/domain"
)

func setupTestDB(t *testing.T) (*db.SQLiteDB, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "aris-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")

	sqlDB, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize test db: %v", err)
	}

	cleanup := func() {
		_ = sqlDB.Close()
		_ = os.RemoveAll(tmpDir)
	}
	return sqlDB, cleanup
}

func TestKnowledgeGraph_AddAndSearch(t *testing.T) {
	sqlDB, cleanup := setupTestDB(t)
	defer cleanup()

	kg := db.NewKnowledgeGraph(sqlDB.DB())
	ctx := context.Background()

	fact := domain.KnowledgeFact{
		Topic:       "style:cyberpunk",
		Concept:     "lighting",
		Fact:        "Use volumetric teal and neon magenta reflections with street rain puddles",
		SourceAgent: "test",
		Labels:      []string{"cyberpunk", "lighting", "neon"},
		Scope:       domain.ScopeStyle,
	}

	id, err := kg.AddFact(ctx, fact)
	if err != nil {
		t.Fatalf("AddFact failed: %v", err)
	}
	if id == "" {
		t.Fatalf("expected non-empty ID")
	}

	// Search by keyword
	results, err := kg.SearchFacts(ctx, "volumetric neon", domain.ScopeStyle, 10)
	if err != nil {
		t.Fatalf("SearchFacts failed: %v", err)
	}
	if len(results) == 0 {
		t.Fatalf("expected at least 1 search result, got 0")
	}
	if results[0].Topic != "style:cyberpunk" {
		t.Errorf("expected topic 'style:cyberpunk', got %q", results[0].Topic)
	}

	// List by Topic
	topicFacts, err := kg.GetFactsByTopic(ctx, "style:cyberpunk")
	if err != nil {
		t.Fatalf("GetFactsByTopic failed: %v", err)
	}
	if len(topicFacts) != 1 {
		t.Fatalf("expected 1 fact for topic, got %d", len(topicFacts))
	}

	// Delete fact
	if err := kg.DeleteFact(ctx, id); err != nil {
		t.Fatalf("DeleteFact failed: %v", err)
	}

	deletedResults, err := kg.GetFactsByTopic(ctx, "style:cyberpunk")
	if err != nil {
		t.Fatalf("GetFactsByTopic after delete failed: %v", err)
	}
	if len(deletedResults) != 0 {
		t.Fatalf("expected 0 facts after delete, got %d", len(deletedResults))
	}
}

func TestHistoryStore_SaveAndRetrieve(t *testing.T) {
	sqlDB, cleanup := setupTestDB(t)
	defer cleanup()

	history := db.NewHistoryStore(sqlDB.DB())
	ctx := context.Background()

	spec := &domain.ImageSpec{
		ID:             "spec-123",
		RawPrompt:      "a samurai cat",
		EnhancedPrompt: "a majestic cyberpunk samurai cat standing on a rooftop in Tokyo, cinematic lighting",
		NegativePrompt: "blurry, distorted",
		AspectRatio:    domain.RatioLandscape,
		Width:          1344,
		Height:         768,
		Steps:          25,
		CFGScale:       7.5,
		Seed:           42,
		Backend:        "pollinations",
		Model:          "flux",
		CreatedAt:      time.Now(),
	}

	result := &domain.ImageResult{
		ID:          "gen-123",
		SpecID:      "spec-123",
		LocalPath:   "/tmp/test.png",
		Format:      "png",
		SizeInBytes: 102400,
		Duration:    1500 * time.Millisecond,
	}

	if err := history.SaveGeneration(ctx, spec, result); err != nil {
		t.Fatalf("SaveGeneration failed: %v", err)
	}

	records, err := history.GetHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	if records[0].PromptRaw != "a samurai cat" {
		t.Errorf("expected prompt 'a samurai cat', got %q", records[0].PromptRaw)
	}
	if records[0].Width != 1344 || records[0].Height != 768 {
		t.Errorf("unexpected dimensions: %dx%d", records[0].Width, records[0].Height)
	}

	// Update Rating
	if err := history.UpdateRating(ctx, "gen-123", 1, "Awesome quality!"); err != nil {
		t.Fatalf("UpdateRating failed: %v", err)
	}

	updated, err := history.GetByID(ctx, "gen-123")
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if updated.Rating != 1 || updated.Feedback != "Awesome quality!" {
		t.Errorf("expected rating 1 with feedback, got %d and %q", updated.Rating, updated.Feedback)
	}
}
