# SDD Design: Upscaling & Face Restoration Pipeline

## 1. Domain Architecture Extensions

### 1.1 `ImageSpec` Modifications (`internal/core/domain/types.go`)
The `ImageSpec` struct will be extended to natively support scaling and face restoration properties as part of the core visual blueprint.

```go
type ImageSpec struct {
    // ... existing fields ...
    ScaleFactor     int     `json:"scale_factor,omitempty"`    // e.g. 2, 4, 8
    RestoreFaces    bool    `json:"restore_faces,omitempty"`   // Toggles face reconstruction
    FaceFidelity    float64 `json:"face_fidelity,omitempty"`   // [0.0, 1.0] fidelity preservation
    UpscalerModel   string  `json:"upscaler_model,omitempty"`  // Specific model (e.g. CodeFormer/GFPGAN)
}
```

### 1.2 `ReferenceMode` Addition
We will add a new constant to explicitly mark upscaling workloads:
```go
const ModeUpscale ReferenceMode = "upscale"
```

### 1.3 Helper & Lifecycle Methods
- **`IsUpscale() bool`**: 
  ```go
  func (s *ImageSpec) IsUpscale() bool {
      return s.Mode == ModeUpscale || s.ScaleFactor > 1
  }
  ```
- **`ApplyDefaults()` Updates**:
  If `IsUpscale()` or `RestoreFaces` is true, explicitly set `Mode = ModeUpscale` if not defined.
  Default `ScaleFactor` to `4` if 0 but in upscale mode.
  If `RestoreFaces` is true, default `FaceFidelity` to `0.75`. Clamp `FaceFidelity` strictly between `0.0` and `1.0`.
- **`Validate()` Updates**:
  If `IsUpscale()` is true, ensure `ScaleFactor` is exactly `2`, `4`, or `8`. Return an error otherwise.

---

## 2. Backend Adapter Implementations

### 2.1 Fal.ai (`internal/adapters/image/falai.go`)
When `spec.IsUpscale()` is true:
- Bypass standard Text2Img endpoint generation.
- **Routing**: Use `fal-ai/esrgan` or `fal-ai/aura-sr` based on `spec.UpscalerModel` or default fallback.
- **Payload**: Provide `image_url` (or base64 URI encoded), `scale` matching `spec.ScaleFactor`. If `spec.RestoreFaces` is true, enable the corresponding face enhancement toggle provided by the endpoint (e.g., `face_enhancer: true` or mapping `spec.FaceFidelity`).

### 2.2 ComfyUI (`internal/adapters/image/comfyui.go`)
When `spec.IsUpscale()` is true:
- Use `uploadImage(ctx, spec.InputImagePath)` to upload the target file to the local ComfyUI instance.
- Construct an upscaling node graph (Workflow API JSON):
  - `LoadImage` node pointing to the uploaded image.
  - `UpscaleModelLoader` node (loading RealESRGAN_x4plus or similar).
  - `ImageUpscaleWithModel` connecting the image and the model.
  - If `spec.RestoreFaces` is true, pipe the upscaled output into a `FaceRestoreCFWithModel` (CodeFormer) node, applying `fidelity: spec.FaceFidelity`.
  - Terminate into `SaveImage` node.

### 2.3 OpenAI & Pollinations
- **OpenAI (`openai.go`)**: Gracefully reject by returning `fmt.Errorf("OpenAI DALL-E backend does not support standalone image upscaling or face restoration")`.
- **Pollinations (`pollinations.go`)**: Similar rejection, or if the API gateway ever natively supports upscaling query parameters in the future, map them to URL parameters. For now, explicitly reject with a clean backend limit error.

---

## 3. Service & Subagent Orchestration

### 3.1 `@upscaler` Subagent (`internal/core/domain/subagent.go`)
Replace the current generic `enhancer` subagent definition with a strict `@upscaler` definition mapped to the spec:

```go
{
    Name:        "upscaler",
    DisplayName: "Super-Resolution & Face Restoration Specialist",
    Role:        "Image Enhancement & Artifact Restoration",
    Description: "Coordinates super-resolution upscaling (2x, 4x, 8x) and face restoration (GFPGAN / CodeFormer) workflows.",
    SystemPrompt: `You are @upscaler, the Super-Resolution & Face Restoration Specialist of ARIS.
You handle:
- Super-resolution scaling factors (2x, 4x, 8x)
- Face restoration and eye sharpening (GFPGAN / CodeFormer setups)
- Natural language parsing of scale requirements and fidelity weighting.`,
    Personality:  "Polished, efficiency-focused, and obsessed with high-resolution clarity.",
    Temperature:  0.2,
    AllowedTools: []string{"upscale", "restore_faces", "evaluate_fidelity"},
}
```

### 3.2 `AgentService.Generate()` Integration
If the prompt starts with `@upscaler` (or the LLM determines upscaling intent), the LLM extracts the local image path, requested scale, and face preferences, mapping them directly into the `ImageSpec` fields. The standard pipeline routes this via the `registry.GetBackend()` which now inherently handles `ModeUpscale`.

---

## 4. CLI Subcommand `aris upscale` Design

**Target File**: `internal/adapters/ui/cli/upscale.go`
We introduce a direct action bypassing standard prompt generation.

```text
Usage: aris upscale <image_path> [options]

Flags:
  -s, --scale int          Scale factor: 2, 4, or 8 (default 4)
  --restore-faces          Enable face artifact reconstruction (default false)
  -f, --fidelity float     Face fidelity weight from 0.0 to 1.0 (default 0.75)
  -b, --backend string     Target backend: falai, comfyui (default: user configured)
  -o, --output string      Output path for the upscaled image
```

**Lifecycle**:
1. Validate command syntax (requires exactly one positional argument: `image_path`).
2. Run `imgutil.LoadAndValidateImage` to ensure the local file exists, is an image, and fits memory/size limits.
3. Build the `ImageSpec`: `InputImagePath: image_path`, `Mode: domain.ModeUpscale`, `ScaleFactor: scale`, `RestoreFaces: restore-faces`, `FaceFidelity: fidelity`.
4. Call `spec.ApplyDefaults()` and `spec.Validate()`.
5. Check backend compatibility (prevent routing to OpenAI).
6. Execute `backend.Generate()` and save the output. Emit a final console summary showing time taken, original size, and final dimension multiplier.

---

## 5. Flow & Sequence Diagrams

### Execution Flow
```ascii
[User / CLI] 
      │ (aris upscale <path> --scale 4)
      ▼
[CLI Parser (upscale.go)] -> Validates local file -> Builds ImageSpec
      │
      ▼
[Agent / Core] -> Runs ImageSpec.ApplyDefaults() -> Checks Validate()
      │
      ▼
[Backend Registry] -> Selects Image Backend (e.g. FalAI / ComfyUI)
      │
      ├──> [FalAIAdapter] -> Converts to Base64 -> POST /fal-ai/esrgan
      │
      └──> [ComfyUIAdapter] -> POST /upload/image 
                             -> Assembles Workflow JSON (LoadImage -> Upscale -> FaceRestore -> Save)
                             -> POST /prompt
      │
      ▼
[Image Output] -> Returns domain.ImageResult -> Saved to disk
```

---

## 6. Testing Strategy

Strict TDD (Red -> Green -> Refactor) using Go's standard `testing` package with table-driven tests.

### 6.1 Unit Tests
- `internal/core/domain/types_test.go`:
  - **ApplyDefaults**: Asserts `ScaleFactor` gets `4`, `Mode` becomes `upscale`, and `FaceFidelity` defaults/clamps correctly.
  - **Validate**: Validates boundary rules (e.g., rejecting a `ScaleFactor` of 5).
  - **IsUpscale**: Validates detection based on mode and scale factor flags.

### 6.2 Integration Tests
- `internal/adapters/image/falai_test.go` / `comfyui_test.go`:
  - Launch internal test HTTP servers (`httptest.NewServer`) mocking Fal.ai and ComfyUI endpoints.
  - Assert that an upscale `ImageSpec` generates the correct backend payload shapes (e.g. checks that the ComfyUI workflow graph properly injects `FaceRestoreCFWithModel` only when `RestoreFaces` is true).
  - Run all adapter tests with `go test -race` to ensure concurrent safety across the HTTP clients and payloads.

### 6.3 CLI Tests
- `internal/adapters/ui/cli/upscale_test.go`:
  - Validate flag parser defaults and argument bindings (e.g. `-s 8` mapping accurately to `ScaleFactor: 8`).
  - Validate graceful failure outputs when a non-existent image path is passed.
