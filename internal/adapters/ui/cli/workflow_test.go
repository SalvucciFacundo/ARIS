package cli_test

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"testing"

	"aris/internal/adapters/ui/cli"
	"aris/pkg/imgutil"
)

func createTestPNGWithMeta(t *testing.T, filePath string, meta map[string]string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	img.Set(0, 0, color.RGBA{R: 100, G: 150, B: 200, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}

	var injected bytes.Buffer
	if err := imgutil.InjectPNGMetadata(bytes.NewReader(buf.Bytes()), &injected, meta); err != nil {
		t.Fatalf("failed to inject metadata: %v", err)
	}

	if err := os.WriteFile(filePath, injected.Bytes(), 0644); err != nil {
		t.Fatalf("failed to write test png: %v", err)
	}
}

func TestWorkflowCommands(t *testing.T) {
	runner, err := cli.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	defer runner.Close()

	tmpDir := t.TempDir()
	comfyImgPath := filepath.Join(tmpDir, "comfy_render.png")
	plainImgPath := filepath.Join(tmpDir, "plain_render.png")
	nonPngPath := filepath.Join(tmpDir, "not_an_image.txt")

	_ = os.WriteFile(nonPngPath, []byte("plain text content"), 0644)

	workflowPayload := `{"nodes": [{"id": 1, "type": "KSampler", "pos": [100, 100]}], "links": []}`
	promptPayload := `{"1": {"class_type": "KSampler", "inputs": {"seed": 123456}}}`
	paramsPayload := "cyberpunk city\nSteps: 25, Sampler: Euler, CFG scale: 7.0, Seed: 123456, Size: 1024x1024, Model: flux"

	createTestPNGWithMeta(t, comfyImgPath, map[string]string{
		"workflow":   workflowPayload,
		"prompt":     promptPayload,
		"parameters": paramsPayload,
	})

	createTestPNGWithMeta(t, plainImgPath, map[string]string{})

	t.Run("workflow inspect without args", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "workflow", "inspect"})
		if code == 0 {
			t.Errorf("expected non-zero exit code when inspecting without image path")
		}
	})

	t.Run("workflow inspect non-existent file", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "workflow", "inspect", filepath.Join(tmpDir, "missing.png")})
		if code == 0 {
			t.Errorf("expected non-zero exit code for non-existent file")
		}
	})

	t.Run("workflow inspect non-png file", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "workflow", "inspect", nonPngPath})
		if code == 0 {
			t.Errorf("expected non-zero exit code for non-png file")
		}
	})

	t.Run("workflow inspect human-readable format", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "workflow", "inspect", comfyImgPath})
		if code != 0 {
			t.Errorf("expected exit code 0, got %d", code)
		}
	})

	t.Run("workflow inspect --json format", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		code := runner.Execute([]string{"aris", "workflow", "inspect", comfyImgPath, "--json"})

		w.Close()
		os.Stdout = oldStdout

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}

		outBytes, _ := io.ReadAll(r)
		var parsed map[string]any
		if err := json.Unmarshal(outBytes, &parsed); err != nil {
			t.Fatalf("inspect --json output is not valid JSON: %v\nOutput was: %s", err, string(outBytes))
		}
		if _, ok := parsed["workflow"]; !ok {
			t.Errorf("expected workflow in json output")
		}
	})

	t.Run("workflow export default path", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "workflow", "export", comfyImgPath})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}

		expectedExportPath := filepath.Join(tmpDir, "comfy_render.workflow.json")
		data, err := os.ReadFile(expectedExportPath)
		if err != nil {
			t.Fatalf("expected exported file at %s: %v", expectedExportPath, err)
		}

		var parsed map[string]any
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("exported workflow is not valid JSON: %v", err)
		}
	})

	t.Run("workflow export without --force when file exists", func(t *testing.T) {
		// comfy_render.workflow.json already exists from previous test
		code := runner.Execute([]string{"aris", "workflow", "export", comfyImgPath})
		if code == 0 {
			t.Errorf("expected non-zero exit code when destination exists and --force is not provided")
		}
	})

	t.Run("workflow export with --force overwrites file", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "workflow", "export", comfyImgPath, "--force"})
		if code != 0 {
			t.Fatalf("expected exit code 0 with --force, got %d", code)
		}
	})

	t.Run("workflow export custom destination -o", func(t *testing.T) {
		customDest := filepath.Join(tmpDir, "custom_export.json")
		code := runner.Execute([]string{"aris", "workflow", "export", comfyImgPath, "-o", customDest})
		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}

		if _, err := os.Stat(customDest); os.IsNotExist(err) {
			t.Fatalf("expected custom export file at %s", customDest)
		}
	})

	t.Run("workflow export to stdout -o -", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		code := runner.Execute([]string{"aris", "workflow", "export", comfyImgPath, "-o", "-"})

		w.Close()
		os.Stdout = oldStdout

		if code != 0 {
			t.Fatalf("expected exit code 0, got %d", code)
		}

		outBytes, _ := io.ReadAll(r)
		var parsed map[string]any
		if err := json.Unmarshal(outBytes, &parsed); err != nil {
			t.Fatalf("exported stdout is not valid JSON: %v", err)
		}
	})

	t.Run("workflow export image without workflow metadata", func(t *testing.T) {
		code := runner.Execute([]string{"aris", "workflow", "export", plainImgPath})
		if code == 0 {
			t.Errorf("expected non-zero exit code when image has no workflow metadata")
		}
	})
}
