# SDD Design: ARIS Img2Img & Visual Reference Pipeline

## 1. Domain Updates (`internal/core/domain/types.go`)

### 1.1 `ReferenceMode` Enum
We introduce a new domain enum to represent the generation mode of the pipeline:
```go
type ReferenceMode string

const (
	ModeText2Img      ReferenceMode = "text2img"
	ModeImg2Img       ReferenceMode = "img2img"
	ModeInpaint       ReferenceMode = "inpaint"
	ModeStyleTransfer ReferenceMode = "style_transfer"
	ModeUpscale       ReferenceMode = "upscale"
)
```

### 1.2 `ImageSpec` Extension
The core configuration struct for an image request will be expanded to carry visual reference parameters:
```go
type ImageSpec struct {
	Prompt          string
	NegativePrompt  string
	AspectRatio     string
	Seed            int64
	
	// New Fields for Img2Img & Inpainting
	Mode            ReferenceMode
	InputImagePath  string  // Optional: Path to base reference image
	MaskImagePath   string  // Optional: Path to mask image
	DenoiseStrength float64 // [0.0, 1.0] - degree of divergence from source
}

// Helper method to resolve defaulting
func (s *ImageSpec) ApplyDefaults() {
    if s.Mode == "" {
        s.Mode = ModeText2Img
    }
    if s.DenoiseStrength == 0.0 && s.Mode != ModeText2Img {
        s.DenoiseStrength = 0.70 // default deviation
    }
}
```

## 2. Shared Utilities (`pkg/imgutil`)

The CLI/Adapter boundary requires strict file checking to prevent OOM/network failures.
Add a new file `pkg/imgutil/reference.go` (or expand `open.go`) with:

- `func LoadAndValidateImage(path string, maxSize int64) ([]byte, error)`
  Reads file, checks existence, errors if > maxSize (e.g. 25MB), checks standard magic bytes (PNG, JPEG, WEBP).
- `func GetDimensions(data []byte) (width, height int, err error)`
  Parses image headers to return dimensions.
- `func ToBase64DataURI(mimeType string, data []byte) string`
  Formats standard `data:image/png;base64,...` payloads.
- `func ValidateMatchingDimensions(imgData, maskData []byte) error`
  Ensures base and mask are strictly identical in width/height.

## 3. Backend Adapter Adaptations (`internal/adapters/image/`)

The core port `ports.ImageBackend` remains `Generate(ctx context.Context, spec domain.ImageSpec) (domain.ImageResult, error)`, but the backend implementations will parse the new spec fields.

### 3.1 Fal.ai (`falai.go`)
- Check `spec.Mode`. If `ModeImg2Img`, construct payload for `fal-ai/flux/dev/image-to-image`.
- If `ModeInpaint` (mask provided), switch target endpoint to `fal-ai/flux-general/inpainting` (or Flux Dev Inpaint API), attaching `image_url` (base64 data URI) and `mask_url`.
- Pass `spec.DenoiseStrength`.

### 3.2 ComfyUI (`comfyui.go`)
- Pre-step: Execute POST `/upload/image` for both `InputImagePath` and `MaskImagePath` using `pkg/imgutil` byte buffers. Retrieve filenames.
- Build dynamic JSON graph mapping:
  - Add `LoadImage` node pointing to base image.
  - If inpainting, add `LoadImage` for mask and `VAEEncodeForInpaint` or equivalent latent composition.
  - Pipe latent output to `KSampler`, setting `denoise` parameter mapped from `spec.DenoiseStrength`.

### 3.3 OpenAI (`openai.go`)
- Uses `POST https://api.openai.com/v1/images/edits`.
- Requires square PNGs. Adapter must validate dimensions or gracefully reject non-square inputs unless `pkg/imgutil` is extended to crop them automatically.
- Construct standard `multipart/form-data` with `image`, `mask`, and `prompt`.

### 3.4 Pollinations (`pollinations.go`)
- Predominantly text-to-image API (GET params).
- Does not have a robust inpaint/img2img masking endpoint.
- If `spec.Mode` is anything other than `ModeText2Img`, return a structural `domain.ErrUnsupportedCapability{Backend: "pollinations", Capability: "img2img"}`. Do not fallback to T2I (to avoid silently ignoring user references).

## 4. Service Layer Updates (`internal/core/services/`)

### 4.1 `AgentService` 
`AgentService.GenerateImage` will now accept the extended `domain.ImageSpec`.
It will pass the populated struct directly to the `ports.ImageBackend`.

### 4.2 `SubagentManager` 
`SubagentManager` intercepts prompts and constructs system parameters.
Add two new presets:
- `@inpainter`: Sets `Mode = ModeInpaint`. Injects system prompt logic optimizing seamless blending.
- `@restyler`: Sets `Mode = ModeStyleTransfer`. Forces `DenoiseStrength = 0.65` (overridable) and prepends stylistic framing to the prompt.

## 5. CLI Interface (`internal/adapters/ui/cli/cli.go`)

Introduce a new `aris edit` subcommand to decouple img2img semantics from the standard `aris gen`.

**Command Signature:**
`aris edit <image_path> "<prompt>" [options]`

**Flags:**
- `--mask <path>`: Local path to the mask image.
- `--strength <float>`: Sets denoise strength [0.0 - 1.0].
- `--backend <name>`: Explicit override of backend (e.g. `falai`).
- `--ratio <string>`: (Standard aspect ratio flag, but typically defaults to input image's native ratio in edit mode).

**Execution Flow in CLI:**
1. Parse flags and `<image_path>`.
2. Check `image_path` existence via `os.Stat`.
3. If `--mask` provided, ensure it exists and dimensions match base image (`imgutil.ValidateMatchingDimensions`).
4. Build `domain.ImageSpec`, inject into `AgentService.GenerateImage`.
5. Output final `image.jpg` as usual.

## 6. Sequence & Data Flow

```text
User CLI                  CLI Layer (aris edit)        AgentService             Image Backend (e.g. Fal.ai)
   |                              |                         |                               |
   | aris edit img.png "dog"      |                         |                               |
   | --mask m.png --strength 0.8  |                         |                               |
   |----------------------------->|                         |                               |
   |                              |-- Validate Input path   |                               |
   |                              |-- Validate Mask path    |                               |
   |                              |-- Check matching dims   |                               |
   |                              |                         |                               |
   |                              | build domain.ImageSpec  |                               |
   |                              |------------------------>|                               |
   |                              |                         | Resolve Backend Adapter       |
   |                              |                         |------------------------------>|
   |                              |                         |                               |-- Base64 Encode inputs
   |                              |                         |                               |-- Check Capabilities
   |                              |                         |                               |-- Construct HTTP request
   |                              |                         |                               |<-- Response (Image bytes)
   |                              |                         |<-- Return ImageResult         |
   |                              |<-- Return output path   |                               |
   |<-- Output saved to ~/.aris   |                         |                               |
```

## 7. Testing Strategy (Strict TDD)

Since Strict TDD mode is active, implement tests following the RED-GREEN-REFACTOR cycle.

1. **`pkg/imgutil` Unit Tests (`pkg/imgutil/reference_test.go`)**
   - *TestLoadAndValidateImage*: Mock a >25MB file, assert size error. Pass a real 1x1 PNG, assert correct bytes.
   - *TestValidateMatchingDimensions*: Supply a 512x512 PNG and a 512x512 mask, assert `nil`. Supply a 512x512 and a 256x256, assert dimension mismatch error.

2. **Backend Adapter Unit Tests** (`internal/adapters/image/falai_test.go`, etc.)
   - Mock HTTP server to capture payload shape.
   - Inject an `ImageSpec` with `ModeInpaint`, `InputImagePath`, `MaskImagePath`.
   - Assert that Fal.ai targets `fal-ai/flux-general/inpainting` instead of standard text-to-image.
   - Assert `pollinations` adapter returns `ErrUnsupportedCapability` when `ModeInpaint` is sent.

3. **CLI Command Tests** (`internal/adapters/ui/cli/cli_test.go`)
   - Simulate `aris edit` execution without positional args -> assert failure.
   - Simulate valid `aris edit` -> assert `ImageSpec` construction matches expected mode and paths.
