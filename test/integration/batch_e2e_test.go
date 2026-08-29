package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"aris/internal/adapters/ui/cli"
)

func TestBatch_CLI_Validation(t *testing.T) {
	runner, err := cli.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	defer runner.Close()

	t.Run("missing prompt", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for missing prompt")
		}
	})

	t.Run("empty prompt", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch", "   "})
		if code == 0 {
			t.Errorf("expected non-zero exit code for empty prompt")
		}
	})

	t.Run("invalid count <= 0", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch", "a car", "--count", "0"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for count 0")
		}
	})

	t.Run("conflicting count and seed-sweep", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch", "a car", "--count", "3", "--seed-sweep", "10-15"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for conflicting count and seed-sweep")
		}
	})

	t.Run("invalid seed-sweep range", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch", "a car", "--seed-sweep", "20-10"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for inverted sweep range")
		}
	})

	t.Run("unknown backend in list", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch", "a car", "--backends", "unknown_backend_xyz"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for unknown backend")
		}
	})
}

func TestBatch_CLI_DryRun(t *testing.T) {
	runner, err := cli.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	defer runner.Close()

	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "dry_run_batch")

	code := runner.Execute([]string{
		"aris", "batch",
		"a [cyberpunk|steampunk] cat",
		"--seed-sweep", "100-101",
		"--backends", "pollinations",
		"--dry-run",
		"-o", outDir,
	})

	if code != 0 {
		t.Fatalf("expected exit code 0 for dry-run, got %d", code)
	}

	// Dry run MUST NOT create the output directory or files
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("dry-run should not create output directory %q", outDir)
	}
}

func TestBatch_CLI_Execution(t *testing.T) {
	runner, err := cli.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	defer runner.Close()

	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "real_batch")

	// Note: pollinations backend is registered in Default SQLite Runner
	code := runner.Execute([]string{
		"aris", "batch",
		"a [red|blue] neon sphere",
		"--seed-sweep", "1-2",
		"--backends", "pollinations",
		"--concurrency", "2",
		"-o", outDir,
	})

	if code != 0 {
		t.Fatalf("expected exit code 0 for batch execution, got %d", code)
	}

	// Verify generated files
	jsonPath := filepath.Join(outDir, "batch_meta.json")
	mdPath := filepath.Join(outDir, "summary.md")
	htmlPath := filepath.Join(outDir, "index.html")

	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Errorf("missing %s", jsonPath)
	}
	if _, err := os.Stat(mdPath); os.IsNotExist(err) {
		t.Errorf("missing %s", mdPath)
	}
	if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
		t.Errorf("missing %s", htmlPath)
	}
}
