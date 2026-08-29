# Design: ARIS LoRA & ControlNet Manager

## 1. Domain Architecture Extensions

### 1.1 Data Structures

Introduce the new configuration structures in the core domain package (`domain/image.go`).

```go
package domain

// LoRAConfig represents a Low-Rank Adaptation model and its weight scale.
type LoRAConfig struct {
    Name  string  `json:"name"`
    Scale float64 `json:"scale"`
}

// ControlNetConfig represents a structural conditioning configuration.
type ControlNetConfig struct {
    Type             string  `json:"type"`             // e.g., "canny", "depth", "openpose"
    Strength         float64 `json:"strength"`         // The conditioning scale
    ReferenceImage   string  `json:"reference_image"`  // Path to the local reference image
    PreprocessedHash string  `json:"preprocessed_hash,omitempty"`
}
```

Extend the existing `ImageSpec` to support collections of these configurations.

```go
type ImageSpec struct {
    // ... existing fields (Prompt, NegativePrompt, Width, Height, etc.)
    LoRAs       []LoRAConfig       `json:"loras,omitempty"`
    ControlNets []ControlNetConfig `json:"controlnets,omitempty"`
}
```

### 1.2 Lifecycle Methods

Add evaluation, default injection, and validation logic for `ImageSpec`.

```go
func (s *ImageSpec) HasLoRA() bool {
    return len(s.LoRAs) > 0
}

func (s *ImageSpec) HasControlNet() bool {
    return len(s.ControlNets) > 0
}

func (s *ImageSpec) ApplyDefaults() {
    // Apply defaults to LoRAs
    for i := range s.LoRAs {
        if s.LoRAs[i].Scale == 0 {
            s.LoRAs[i].Scale = 1.0
        }
        // Clamp bounds [0.0, 2.0]
        if s.LoRAs[i].Scale > 2.0 {
            s.LoRAs[i].Scale = 2.0
        } else if s.LoRAs[i].Scale < 0.0 {
            s.LoRAs[i].Scale = 0.0
        }
    }
    
    // Apply defaults to ControlNets
    for i := range s.ControlNets {
        if s.ControlNets[i].Strength == 0 {
            s.ControlNets[i].Strength = 1.0
        }
        if s.ControlNets[i].Strength > 2.0 {
            s.ControlNets[i].Strength = 2.0
        } else if s.ControlNets[i].Strength < 0.0 {
            s.ControlNets[i].Strength = 0.0
        }
    }
}

func (s *ImageSpec) Validate() error {
    validTypes := map[string]bool{"canny": true, "depth": true, "openpose": true, "lineart": true, "scribble": true}
    for _, cn := range s.ControlNets {
        if !validTypes[strings.ToLower(cn.Type)] {
            return fmt.Errorf("unsupported controlnet type: %s", cn.Type)
        }
        if cn.ReferenceImage != "" {
            if _, err := os.Stat(cn.ReferenceImage); os.IsNotExist(err) {
                return fmt.Errorf("reference image does not exist: %s", cn.ReferenceImage)
            }
        }
    }
    return nil
}
```

## 2. Prompt Tag Extraction & Image Preprocessor

### 2.1 Prompt Parser (Regex)
Location: `pkg/prompt/parser.go`
- **Regex Pattern:** `(?i)<lora:([^:>]+)(?::([0-9.]+))?>`
- **Logic:** 
  1. Find all matches in the raw prompt.
  2. Extract Name (group 1).
  3. Extract Scale (group 2, parse to float64; if missing, use 1.0).
  4. Replace matched blocks with empty spaces, trim redundant spaces, and return the cleaned prompt alongside a slice of `domain.LoRAConfig`.

### 2.2 Pure-Go Canny Edge Detection Preprocessor
Location: `pkg/imgutil/controlnet.go`

Implemented entirely in standard Go to avoid external bindings (e.g., OpenCV, Python).

```go
package imgutil

import (
    "image"
)

// PreprocessCanny executes a Canny edge detection pipeline on the input image.
func PreprocessCanny(img image.Image, lowThresh, highThresh float64) (image.Image, error) {
    // Standard Defaults if inputs are 0
    if lowThresh == 0 && highThresh == 0 {
        lowThresh, highThresh = 100.0, 200.0
    }
    
    // 1. Grayscale Conversion: Y = 0.299R + 0.587G + 0.114B
    gray := convertToGrayscale(img)
    
    // 2. Gaussian Filter: Apply 5x5 convolution to suppress high-frequency noise
    blurred := applyGaussianBlur(gray)
    
    // 3. Sobel Operator: Compute Gx and Gy, derive gradient magnitude and direction (theta)
    magnitudes, directions := applySobel(blurred)
    
    // 4. Non-Maximum Suppression (NMS): Thin edges based on local gradient direction
    suppressed := nonMaximumSuppression(magnitudes, directions)
    
    // 5. Hysteresis Thresholding: Link edges using highThresh and lowThresh
    finalEdges := applyHysteresis(suppressed, lowThresh, highThresh)
    
    return finalEdges, nil
}
```

## 3. Multi-Backend Dynamic Chaining

### 3.1 ComfyUI Dynamic Graph Wiring
Location: `internal/adapters/image/comfyui.go`

Dynamically assemble the JSON graph for the `/prompt` API endpoint.

**LoRA Chaining:**
Initialize `CheckpointLoaderSimple` (Model & CLIP outputs). For $N$ LoRAs, dynamically construct sequential `LoraLoader` nodes:
- Node $L_1$: Model IN = Checkpoint Model, CLIP IN = Checkpoint CLIP.
- Node $L_{i}$: Model IN = $L_{i-1}$ Model, CLIP IN = $L_{i-1}$ CLIP.
- Connect final node $L_N$ outputs to `CLIPTextEncode` and `KSampler`.

**ControlNet Chaining:**
If ControlNets exist, inject `ControlNetLoader` and `ApplyControlNet` nodes:
- Image API Push: Automatically upload reference images to ComfyUI's `/upload/image`.
- Add a `LoadImage` node to pull the uploaded image.
- Wire `CLIPTextEncode` (Positive/Negative) sequentially through the $M$ `ApplyControlNet` nodes.
- Feed the terminal `ApplyControlNet` output into `KSampler`.

### 3.2 Fal.ai Flux Payload Mapping
Location: `internal/adapters/image/falai.go`

Map the parsed arrays directly into the JSON POST payload:
```json
{
  "prompt": "...",
  "loras": [
    { "path": "<lora_url_or_id>", "scale": 1.2 }
  ],
  "controlnets": [
    { 
      "control_image_url": "data:image/jpeg;base64,...", 
      "controlnet_type": "canny", 
      "conditioning_scale": 0.8 
    }
  ]
}
```
Requires base64 encoding the final preprocessed image.

### 3.3 Graceful Degradation (Pollinations & OpenAI)
Location: `internal/adapters/image/pollinations.go` and `openai.go`

Check `HasLoRA()` and `HasControlNet()`. If true, log warnings indicating these features are unsupported on the selected backend, and proceed cleanly using only base configurations.

## 4. CLI Subcommand & Flag Design

### 4.1 CLI Flags
Integrate with `aris gen`, `aris edit`, and `aris batch`:
- `--lora` (StringSlice): Accepts format `<name>:<scale>` or `<name>`. (e.g., `--lora "neon_cyber:0.85"`).
- `--controlnet` (StringSlice): Accepts format `<type>:<scale>:<path>`. (e.g., `--controlnet "canny:0.75:pose.png"`).

### 4.2 LoRA & ControlNet Subcommands
- `aris lora list`: Scans configured paths for local LoRAs and displays them in a table.
- `aris controlnet types`: Emits supported types (canny, depth, openpose, lineart, scribble).
- `aris controlnet preproc <type> <input> --output <out>`: Invokes `imgutil.PreprocessCanny()` (or others) and outputs to the specified path for debugging or manual caching.

## 5. ASCII Flow & Node Graph Diagrams

### 5.1 System Data Flow
```text
[ CLI Flags / Raw Prompt ]
           |
           v
[ Prompt Parser regex  ] --> Strips <lora:name:scale> tags
           |
           v
[ Domain.ImageSpec     ] --> Merges & Clamps scales (0.0 - 2.0)
           |
           v
[ imgutil Preprocessor ] --> If Canny + Image exists, generates edges map
           |
   +-------+-------+---------------+
   |               |               |
[ ComfyUI ]     [ Fal.ai ]     [ Pollinations ]
   |               |               |
Node Wiring    JSON Mapping    Fallback warn
```

### 5.2 ComfyUI Graph Linkage
```text
CheckpointLoaderSimple
  ├── MODEL ───────────────> LoraLoader (1)
  └── CLIP ───────────────>    ├── MODEL ──────────> LoraLoader (2)
                               └── CLIP ──────────>   ├── MODEL ──────> KSampler
                                                      └── CLIP ──────> CLIPTextEncode
                                                                           │
ControlNetLoader ───────────> ApplyControlNet                              │
                                   ├── CONDITIONING (Positive) <───────────┘
LoadImage (Canny Map) ────────────>├── CONDITIONING (Negative) <───── [Negative Prompt]
                                   │
                                   └── CONDITIONING OUT ──────────────> KSampler (Positive IN)
```

## 6. Testing Strategy

Following Strict TDD principles using `-race`:

1. **Unit - Parsers:** `pkg/prompt/parser_test.go`
   - Table-driven tests validating regex extraction: explicit scales, default scales, missing scales, multiple tags, and tags without colons. Ensure resulting string has tags perfectly stripped.
2. **Unit - Domain:** `domain/image_test.go`
   - Assert `ApplyDefaults()` correctly sets lower/upper bounds (0.0 to 2.0).
   - Assert `Validate()` denies unapproved string types (`"unknown_cn_type"`).
3. **Unit - Preprocessor:** `pkg/imgutil/controlnet_test.go`
   - Test synthetic grids against `PreprocessCanny` to prove matrix mutations inside `applySobel` and `nonMaximumSuppression` are correct and memory-safe (`go test -race`).
4. **Integration - ComfyUI:** `internal/adapters/image/comfyui_test.go`
   - Validate dynamically generated JSON nodes check out topologically (sequential IN -> OUT connections for N LoRAs).
