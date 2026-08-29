# Specification: ARIS LoRA & ControlNet Manager

## Purpose
Extend the ARIS core domain, image utilities, multi-backend adapters, and CLI interface to support LoRA (Low-Rank Adaptation) weight stacking and ControlNet structural conditioning across local (ComfyUI) and cloud (Fal.ai Flux) generation backends, featuring a pure-Go Canny edge detection preprocessor and robust prompt tag parsing.

---

## User Stories & Acceptance Criteria

### REQ-LORA-1: LoRA Prompt Tag Parsing & Multi-LoRA CLI Stacking
The system MUST support parsing inline LoRA tags within prompt strings as well as stacking multiple LoRAs via CLI flags.

#### Scenario: Parse inline LoRA tags with explicit scales
- **GIVEN** a prompt string containing one or more inline LoRA tags, such as `"a cyberpunk portrait of a hero <lora:neon_cyber:0.85> <lora:detail_booster:0.6> in Tokyo"`
- **WHEN** the prompt parser parses the raw prompt
- **THEN** the system MUST extract `LoRAConfig{Name: "neon_cyber", Scale: 0.85}` and `LoRAConfig{Name: "detail_booster", Scale: 0.6}`, and clean the prompt to `"a cyberpunk portrait of a hero in Tokyo"`.

#### Scenario: Parse inline LoRA tag with default scale
- **GIVEN** a prompt string containing `<lora:retro_anime>` without an explicit scale factor
- **WHEN** the prompt parser processes the string
- **THEN** the system MUST extract `LoRAConfig{Name: "retro_anime", Scale: 1.0}` and remove the tag from the sanitized prompt.

#### Scenario: Stacking multiple LoRAs via CLI flags
- **GIVEN** CLI command flags `--lora "retro_anime:0.75" --lora "studio_lighting:1.2"`
- **WHEN** the CLI constructs the `domain.ImageSpec`
- **THEN** the system MUST merge all declared LoRAs into `ImageSpec.LoRAs` in the specified sequence, preserving their relative stacking order.

#### Scenario: Merging inline prompt LoRAs and CLI flag LoRAs
- **GIVEN** a prompt with `<lora:character_v2:0.9>` and a CLI flag `--lora "style_vintage:0.5"`
- **WHEN** the generation request is assembled
- **THEN** the system MUST combine both LoRA configurations into `ImageSpec.LoRAs` without duplicates if identical names are used (resolving to the explicitly declared flag or highest-precedence definition).

---

### REQ-LORA-2: LoRA Scale Factor Clamping & Defaults
The system MUST validate and enforce scale bounds for all configured LoRAs to prevent model weights from destabilizing inference.

#### Scenario: LoRA scale within valid bounds
- **GIVEN** a LoRA configuration with scale `0.75` (within `[0.0, 2.0]`)
- **WHEN** validation is executed on `domain.LoRAConfig` or `domain.ImageSpec`
- **THEN** the system MUST accept the scale as valid and unmodified.

#### Scenario: LoRA scale exceeding upper bound
- **GIVEN** a LoRA specification with scale `3.5` or `<lora:art_style:3.5>`
- **WHEN** `ApplyDefaults` or parameter validation is invoked
- **THEN** the system MUST clamp the scale to `2.0` and emit a warning log indicating the clamp occurred.

#### Scenario: Negative LoRA scale clamping or validation
- **GIVEN** a LoRA specification with scale `-0.5`
- **WHEN** parameter validation runs
- **THEN** the system MUST clamp the scale to `0.0` or reject negative scales based on domain boundary rules.

#### Scenario: Empty or omitted LoRA scale defaults
- **GIVEN** a LoRA configuration with scale `0.0` or unset
- **WHEN** `ApplyDefaults` runs on `domain.LoRAConfig`
- **THEN** if the scale was uninitialized, the system MUST assign the default scale of `1.0`.

---

### REQ-CNET-1: ControlNet Type Validation & Parameter Bounds
The system MUST validate ControlNet structural conditioning types, reference image inputs, and conditioning strength parameters.

#### Scenario: Valid ControlNet types
- **GIVEN** a ControlNet configuration with type `canny`, `openpose`, `depth`, `lineart`, or `scribble`
- **WHEN** `domain.ControlNetConfig.Validate()` is invoked
- **THEN** the system MUST validate the type successfully regardless of case (e.g. `Canny` or `canny`).

#### Scenario: Unsupported ControlNet type
- **GIVEN** a ControlNet request with type `unknown_sketch` or `segmentation_v9`
- **WHEN** validation runs
- **THEN** the system MUST reject the configuration with a descriptive error listing supported types (`canny`, `openpose`, `depth`, `lineart`, `scribble`).

#### Scenario: ControlNet reference image verification
- **GIVEN** a ControlNet specification with a local image path `/path/to/pose.png`
- **WHEN** the pre-execution validation executes
- **THEN** the system MUST verify that the file exists and is a valid image format (PNG, JPEG, WEBP); if the file does not exist, it MUST return an immediate file error without calling the backend.

#### Scenario: ControlNet strength clamping
- **GIVEN** a ControlNet strength parameter outside `[0.0, 2.0]` (e.g. `2.5`)
- **WHEN** `ApplyDefaults()` runs
- **THEN** the system MUST clamp the strength to `2.0` (or default to `1.0` if omitted).

---

### REQ-CNET-2: Pure-Go Canny Edge Detection Preprocessor
The system MUST implement a zero-external-dependency, pure-Go Canny edge detection preprocessor in `pkg/imgutil/controlnet.go` to prepare images for Canny ControlNet models.

#### Scenario: Standard Canny preprocessor execution
- **GIVEN** a raw input `image.Image` of dimensions $W \times H$
- **WHEN** `imgutil.CannyEdgeDetection(img, lowThreshold, highThreshold)` is called
- **THEN** the system MUST execute:
  1. Grayscale luminance conversion ($Y = 0.299R + 0.587G + 0.114B$).
  2. Gaussian smoothing filter (5x5 kernel) to suppress high-frequency noise.
  3. Sobel gradient operators to compute gradient magnitude and direction ($\Theta$).
  4. Non-Maximum Suppression (NMS) along gradient directions to thin edges.
  5. Double thresholding and hysteresis edge tracking to link strong and weak edges.
- **AND** the returned edge map MUST be a binary or grayscale image of dimensions $W \times H$ suitable for Canny conditioning.

#### Scenario: Canny default threshold parameters
- **GIVEN** an invocation of `imgutil.PreprocessCanny(imgPath)` without explicit thresholds
- **WHEN** the preprocessor executes
- **THEN** the system MUST default `lowThreshold` to `100` and `highThreshold` to `200` (or standard normalized ratios `0.33` / `0.67`).

#### Scenario: Preprocessing non-Canny ControlNet types
- **GIVEN** a ControlNet type that does not require local edge processing (e.g. raw pre-extracted `openpose` or `depth` map)
- **WHEN** preconditioning runs
- **THEN** the system MUST pass through the validated image bytes directly without altering pixel data.

---

### REQ-CNET-3: Multi-Backend Dynamic Node Chaining & Execution
The system backend adapters MUST dynamically construct execution graphs and API payloads for LoRA and ControlNet pipelines according to backend capabilities.

#### Scenario: ComfyUI dynamic LoRA node chaining
- **GIVEN** a target backend `comfyui` with $N$ configured LoRAs ($N \ge 1$)
- **WHEN** `ComfyUIBackend` generates the workflow JSON graph
- **THEN** it MUST dynamically insert $N$ `LoraLoader` nodes in a sequential chain, piping `MODEL` and `CLIP` outputs from node $i$ into node $i+1$, terminating into the `KSampler` and `CLIPTextEncode` nodes.

#### Scenario: ComfyUI dynamic ControlNet node insertion
- **GIVEN** a target backend `comfyui` with one or more ControlNets configured
- **WHEN** `ComfyUIBackend` builds the workflow
- **THEN** it MUST:
  1. Upload the reference or preprocessed edge map image to ComfyUI via `/upload/image`.
  2. Insert `ControlNetLoader` and `ApplyControlNet` nodes into the graph.
  3. Wire the positive/negative conditioning from `CLIPTextEncode` through `ApplyControlNet` to `KSampler`.

#### Scenario: Fal.ai Flux LoRA and ControlNet payload construction
- **GIVEN** target backend `falai` with configured LoRAs and ControlNets
- **WHEN** `FalAIBackend` constructs the JSON payload for `fal-ai/flux-lora` or `fal-ai/flux-general/controlnet`
- **THEN** it MUST format the LoRA array as `loras: [{"path": "<lora_url_or_id>", "scale": <scale>}]` and map ControlNets with `control_image_url`, `controlnet_type`, and `conditioning_scale`.

#### Scenario: Unsupported backend graceful handling
- **GIVEN** target backend `pollinations` or `openai` with configured LoRA or ControlNet
- **WHEN** generation is initiated
- **THEN** the system MUST either return a descriptive error stating that LoRA/ControlNet conditioning is not supported on that backend, or issue a warning log and execute base text-to-image without corrupting the API request.

---

### REQ-LORA-3: CLI Subcommands & Flag Integration
The system CLI MUST provide dedicated flags on generation commands and dedicated management subcommands for LoRA and ControlNet discovery and preprocessing.

#### Scenario: CLI flag invocation on `aris gen`
- **GIVEN** invocation `aris gen "portrait of an astronaut" --lora "sci_fi_suit:0.8" --controlnet "canny:0.9:pose.png" --backend comfyui`
- **WHEN** the command runs
- **THEN** the CLI MUST parse all flags, invoke Canny preprocessing on `pose.png`, configure `domain.ImageSpec`, and dispatch the request to the backend.

#### Scenario: CLI subcommand `aris lora list`
- **GIVEN** invocation `aris lora list`
- **WHEN** the subcommand executes
- **THEN** the system MUST inspect local ComfyUI model directories or configured LoRA registries and print available LoRAs with their detected file names and default scales.

#### Scenario: CLI subcommand `aris controlnet preproc`
- **GIVEN** invocation `aris controlnet preproc canny input.png --output edges.png`
- **WHEN** the subcommand executes
- **THEN** the system MUST run `imgutil.CannyEdgeDetection` on `input.png` and save the resulting edge map image to `edges.png`.

#### Scenario: Invalid CLI flag syntax
- **GIVEN** invocation `aris gen "prompt" --lora "invalid_format_without_colon_or_name:"`
- **WHEN** flag validation runs
- **THEN** the CLI MUST report a syntax error describing the expected format (`<name>:<scale>` or `<name>`).

---

### REQ-LORA-4: Persistence & Metadata Recording
The system MUST record applied LoRAs and ControlNets into the generation metadata and history store for reproducibility.

#### Scenario: Recording LoRA & ControlNet metadata
- **GIVEN** a successful generation utilizing 2 LoRAs and 1 ControlNet
- **WHEN** `domain.ImageResult` and history records are written
- **THEN** `ImageResult.Metadata` MUST contain the exact LoRA names, scales, ControlNet types, and reference image hashes used during inference.
