package cli_test

import (
	"path/filepath"
	"testing"

	"aris/internal/adapters/ui/cli"
)

func TestRunner_UpscaleCommand(t *testing.T) {
	runner, err := cli.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	defer runner.Close()

	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "input_photo.png")
	createTestPNG(t, imagePath, 128, 128)

	t.Run("missing positional args", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "upscale"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for missing image path argument")
		}
	})

	t.Run("non-existent image file", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "upscale", "/invalid/non_existent.png"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for non-existent image")
		}
	})

	t.Run("invalid scale factor", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "upscale", imagePath, "--scale", "5"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for invalid scale factor 5")
		}
	})
}
