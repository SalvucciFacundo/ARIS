package services_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"aris/internal/adapters/db"
	"aris/internal/adapters/llm"
	"aris/internal/core/domain"
	"aris/internal/core/services"
)

func TestAutoLearner_HeuristicReflection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "aris-autolearn-test-*")
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
	learner := services.NewAutoLearner(kg, llm.NewPassthroughProvider())
	ctx := context.Background()

	// 1. Test preference learning: aspect ratio
	turn1 := services.ReflectionTurn{
		RawInput:       "siempre que pido fondos usa 16:9",
		EnhancedPrompt: "a cyberpunk city, 16:9",
	}

	facts1, err := learner.ReflectTurn(ctx, turn1)
	if err != nil {
		t.Fatalf("ReflectTurn failed: %v", err)
	}
	if len(facts1) == 0 {
		t.Fatalf("expected at least 1 learned fact from turn1")
	}
	if facts1[0].Scope != domain.ScopeUser {
		t.Errorf("expected user scope for preference, got %s", facts1[0].Scope)
	}

	// 2. Test negative trigger extraction: "sin casco"
	turn2 := services.ReflectionTurn{
		RawInput:       "hacela de nuevo pero sin casco",
		EnhancedPrompt: "a dog jumping on the moon without helmet",
	}

	facts2, err := learner.ReflectTurn(ctx, turn2)
	if err != nil {
		t.Fatalf("ReflectTurn failed: %v", err)
	}
	if len(facts2) == 0 {
		t.Fatalf("expected at least 1 learned fact from turn2")
	}
	if facts2[0].Topic != "pref:negative" {
		t.Errorf("expected pref:negative topic, got %s", facts2[0].Topic)
	}

	// 3. Verify facts are present in SQLite Knowledge Graph
	stored, err := kg.SearchFacts(ctx, "casco", "", 5)
	if err != nil {
		t.Fatalf("SearchFacts failed: %v", err)
	}
	if len(stored) == 0 {
		t.Fatalf("expected stored fact for 'casco' in knowledge graph")
	}
}
