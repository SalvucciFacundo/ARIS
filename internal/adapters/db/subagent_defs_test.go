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

	// 1. Verify default subagents were auto-bootstrapped
	defaultCount := len(domain.DefaultSubagents())
	list, err := store.ListSubagents(ctx)
	if err != nil {
		t.Fatalf("ListSubagents failed: %v", err)
	}
	if len(list) != defaultCount {
		t.Fatalf("expected %d default subagents, got %d", defaultCount, len(list))
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
	if len(updatedList) != defaultCount+1 {
		t.Fatalf("expected %d subagents after custom add, got %d", defaultCount+1, len(updatedList))
	}

	// 4. Delete custom subagent
	if err := store.DeleteSubagent(ctx, "illustrator"); err != nil {
		t.Fatalf("DeleteSubagent failed: %v", err)
	}

	finalList, _ := store.ListSubagents(ctx)
	if len(finalList) != defaultCount {
		t.Fatalf("expected %d subagents after delete, got %d", defaultCount, len(finalList))
	}
}
