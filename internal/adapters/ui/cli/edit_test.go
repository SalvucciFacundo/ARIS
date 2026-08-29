package cli_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"aris/internal/adapters/ui/cli"
)

func createTestPNG(t *testing.T, path string, w, h int) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 200, G: 150, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test png: %v", err)
	}
}

func TestRunner_EditCommand(t *testing.T) {
	runner, err := cli.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	defer runner.Close()

	tmpDir := t.TempDir()
	basePath := filepath.Join(tmpDir, "base.png")
	maskPath := filepath.Join(tmpDir, "mask.png")
	badMaskPath := filepath.Join(tmpDir, "bad_mask.png")

	createTestPNG(t, basePath, 128, 128)
	createTestPNG(t, maskPath, 128, 128)
	createTestPNG(t, badMaskPath, 64, 64)

	t.Run("missing positional args", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "edit"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for missing args")
		}
	})

	t.Run("missing prompt", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "edit", basePath})
		if code == 0 {
			t.Errorf("expected non-zero exit code for missing prompt")
		}
	})

	t.Run("non-existent image file", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "edit", "/invalid/path.png", "make it gothic"})
		if code == 0 {
			t.Errorf("expected non-zero exit code for non-existent image")
		}
	})

	t.Run("mismatched mask dimensions", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "edit", basePath, "remove glasses", "--mask", badMaskPath})
		if code == 0 {
			t.Errorf("expected non-zero exit code for mismatched mask dimensions")
		}
	})
}
