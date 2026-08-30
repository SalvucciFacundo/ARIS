package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	imgadapter "aris/internal/adapters/image"
	"aris/internal/adapters/ui/cli"
	"aris/internal/core/domain"
	"aris/pkg/imgutil"
)

func TestWorkflow_E2E_FullLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	outDir := filepath.Join(tmpDir, "outputs")
	_ = os.MkdirAll(outDir, 0755)

	// Create mock ComfyUI server returning valid PNG
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	img.Set(0, 0, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	var rawPngBuf bytes.Buffer
	_ = png.Encode(&rawPngBuf, img)

	comfyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/prompt" && r.Method == http.MethodPost:
			resp := map[string]any{"prompt_id": "e2e-prompt-123"}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case r.URL.Path == "/history/e2e-prompt-123":
			resp := map[string]any{
				"e2e-prompt-123": map[string]any{
					"outputs": map[string]any{
						"9": map[string]any{
							"images": []map[string]any{
								{"filename": "e2e_output.png", "subfolder": "", "type": "output"},
							},
						},
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(resp)
		case r.URL.Path == "/view":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(rawPngBuf.Bytes())
		default:
			http.NotFound(w, r)
		}
	}))
	defer comfyServer.Close()

	// 1. Generate an image with ComfyUI adapter
	backend := imgadapter.NewComfyUIBackend(comfyServer.URL, outDir, comfyServer.Client())
	spec := &domain.ImageSpec{
		ID:        "e2e-spec-1",
		RawPrompt: "cyberpunk floating citadel over ocean",
	}

	res, err := backend.Generate(context.Background(), spec)
	if err != nil {
		t.Fatalf("ComfyUI Generate failed in E2E test: %v", err)
	}

	if res.LocalPath == "" {
		t.Fatalf("expected valid local path")
	}

	// 2. Verify metadata is directly extractable with imgutil
	meta, err := imgutil.ExtractPNGMetadataFile(res.LocalPath)
	if err != nil {
		t.Fatalf("failed to extract metadata from generated PNG: %v", err)
	}

	if _, ok := meta["prompt"]; !ok {
		t.Fatalf("missing 'prompt' metadata in generated PNG")
	}
	if _, ok := meta["workflow"]; !ok {
		t.Fatalf("missing 'workflow' metadata in generated PNG")
	}

	// 3. Test CLI workflow inspect --json
	runner, err := cli.NewRunner()
	if err != nil {
		t.Fatalf("NewRunner failed: %v", err)
	}
	defer runner.Close()

	code := runner.Execute([]string{"aris", "workflow", "inspect", res.LocalPath, "--json"})
	if code != 0 {
		t.Fatalf("expected exit code 0 for inspect --json, got %d", code)
	}

	// 4. Test CLI workflow export
	exportDest := filepath.Join(tmpDir, "exported_workflow.json")
	exportCode := runner.Execute([]string{"aris", "workflow", "export", res.LocalPath, "-o", exportDest})
	if exportCode != 0 {
		t.Fatalf("expected exit code 0 for workflow export, got %d", exportCode)
	}

	exportData, err := os.ReadFile(exportDest)
	if err != nil {
		t.Fatalf("failed to read exported workflow file: %v", err)
	}

	var workflowObj map[string]any
	if err := json.Unmarshal(exportData, &workflowObj); err != nil {
		t.Fatalf("exported workflow is not valid JSON: %v", err)
	}

	if _, ok := workflowObj["nodes"]; !ok {
		t.Fatalf("expected 'nodes' in exported workflow JSON: %+v", workflowObj)
	}
}
