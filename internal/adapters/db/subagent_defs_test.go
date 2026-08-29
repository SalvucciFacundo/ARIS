package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"aris/internal/adapters/db"
	"aris/internal/core/domain"
)

func TestSubagentStore_BootstrapAndCRUD(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "aris-subagent-test-*")
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	sqlDB, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}
	defer sqlDB.Close()

	store := db.NewSubagentStore(sqlDB.DB())
	ctx := context.Background()

	// 1. Verify 5 default subagents were auto-bootstrapped
	list, err := store.ListSubagents(ctx)
	if err != nil {
		t.Fatalf("ListSubagents failed: %v", err)
	}
	if len(list) != 5 {
		t.Fatalf("expected 5 default subagents, got %d", len(list))
	}

	// 2. Fetch specific subagent
	director, err := store.GetSubagent(ctx, "director")
	if err != nil {
		t.Fatalf("GetSubagent director failed: %v", err)
	}
	if director.Name != "director" || director.DisplayName != "Art Director" {
		t.Errorf("unexpected director definition: %+v", director)
	}

	// 3. Add custom subagent
	custom := domain.SubagentDef{
		Name:         "illustrator",
		DisplayName:  "Anime Illustrator",
		Role:         "Anime Style Specialist",
		Description:  "Specializes in Japanese anime illustration",
		SystemPrompt: "You are @illustrator...",
		Personality:  "Artistic",
		Temperature:  0.7,
		AllowedTools: []string{"search_memory"},
	}

	if err := store.SaveSubagent(ctx, custom); err != nil {
		t.Fatalf("SaveSubagent failed: %v", err)
	}

	updatedList, err := store.ListSubagents(ctx)
	if err != nil {
		t.Fatalf("ListSubagents after add failed: %v", err)
	}
	if len(updatedList) != 6 {
		t.Fatalf("expected 6 subagents after custom add, got %d", len(updatedList))
	}

	// 4. Delete custom subagent
	if err := store.DeleteSubagent(ctx, "illustrator"); err != nil {
		t.Fatalf("DeleteSubagent failed: %v", err)
	}

	finalList, _ := store.ListSubagents(ctx)
	if len(finalList) != 5 {
		t.Fatalf("expected 5 subagents after delete, got %d", len(finalList))
	}
}
