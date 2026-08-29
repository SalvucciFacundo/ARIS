package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"aris/internal/adapters/ui/cli"
)

func TestRunner_BatchCommand(t *testing.T) {
	runner, err := cli.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	defer runner.Close()

	t.Run("missing prompt", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch"})
		if code != 1 {
			t.Errorf("expected exit code 1 for missing prompt, got %d", code)
		}
	})

	t.Run("empty prompt", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch", "   "})
		if code != 1 {
			t.Errorf("expected exit code 1 for whitespace prompt, got %d", code)
		}
	})

	t.Run("dry run mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		outDir := filepath.Join(tmpDir, "batch_dry_run")

		code := runner.Execute([]string{
			"aris", "batch",
			"a [cyberpunk|steampunk] warrior",
			"--seed-sweep", "10-11",
			"--dry-run",
			"-o", outDir,
		})

		if code != 0 {
			t.Errorf("expected exit code 0 for dry-run, got %d", code)
		}

		if _, err := os.Stat(outDir); !os.IsNotExist(err) {
			t.Errorf("output directory should not exist after dry run")
		}
	})

	t.Run("invalid count", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "batch", "a car", "-c", "-1"})
		if code != 1 {
			t.Errorf("expected exit code 1 for negative count, got %d", code)
		}
	})

	t.Run("matrix upper bound exceeded without force", func(t *testing.T) {
		code := runner.Execute([]string{
			"aris", "batch",
			"a [a|b|c|d|e] [1|2|3|4|5] [x|y|z]",
			"--max-matrix-jobs", "10",
			"--dry-run",
		})
		if code != 1 {
			t.Errorf("expected exit code 1 when matrix exceeds limit, got %d", code)
		}
	})
}
