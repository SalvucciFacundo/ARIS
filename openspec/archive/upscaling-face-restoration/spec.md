# Specification: ARIS Upscaling & Face Restoration Pipeline

## Purpose
Extend the ARIS core domain, image processing pipeline, multi-backend adapters, visual subagents, and CLI interface to support super-resolution image upscaling (2x, 4x, 8x) and deep face restoration (CodeFormer, GFPGAN) workflows across supported image backends (Fal.ai, ComfyUI, Pollinations, OpenAI).

---

## User Stories & Acceptance Criteria

### REQ-UPSCALE-1: Scale Factor Parameter Bounds & Defaults
The system MUST support discrete integer super-resolution scaling multipliers (2x, 4x, 8x) for image upscaling requests and calculate expected output dimensions deterministically.

#### Scenario: Default scale factor assignment
- **GIVEN** an upscaling request with no explicit `--scale` parameter specified
- **WHEN** `ImageSpec.ApplyDefaults()` is executed
- **THEN** the system MUST set `ScaleFactor` to default `4` and `Mode` to `ModeUpscale` (`"upscale"`).

#### Scenario: Valid explicit scale factor selection (2x, 4x, 8x)
- **GIVEN** a user request with scale factor set to `2`, `4`, or `8`
- **WHEN** `ImageSpec.Validate()` is executed
- **THEN** the system MUST accept the scale factor and compute the target dimensions as $\text{Width}_{\text{target}} = \text{Width}_{\text{src}} \times \text{ScaleFactor}$ and $\text{Height}_{\text{target}} = \text{Height}_{\text{src}} \times \text{ScaleFactor}$.

#### Scenario: Rejection of invalid or out-of-bounds scale factor
- **GIVEN** an upscaling request with an unsupported scale factor (e.g., `3`, `5`, `0`, `-2`, `16`)
- **WHEN** `ImageSpec.Validate()` is executed
- **THEN** the system MUST reject the specification and return a validation error stating that supported scale factors are `2`, `4`, and `8`.

#### Scenario: ImageSpec mode helper detection
- **GIVEN** an `ImageSpec` configured with `Mode: ModeUpscale` or `ScaleFactor > 1` with a valid `InputImagePath`
- **WHEN** `ImageSpec.IsUpscale()` is evaluated
- **THEN** the system MUST return `true`.

---

### REQ-UPSCALE-2: Face Restoration & Fidelity Weight Blending
The system MUST provide dedicated facial reconstruction controls (`RestoreFaces`, `FaceFidelity`, `UpscalerModel`) to sharpen and reconstruct facial anatomy while preserving source identity.

#### Scenario: Enabling face restoration with default fidelity
- **GIVEN** an upscaling or image processing request with `--restore-faces` enabled but no explicit `--fidelity` provided
- **WHEN** `ImageSpec.ApplyDefaults()` is executed
- **THEN** the system MUST set `RestoreFaces` to `true` and default `FaceFidelity` to `0.75`.

#### Scenario: Custom face fidelity weighting within valid bounds
- **GIVEN** a face restoration request with an explicit `--fidelity` value between `0.0` and `1.0` (e.g., `0.85`)
- **WHEN** `ImageSpec.ApplyDefaults()` and `ImageSpec.Validate()` are executed
- **THEN** the system MUST assign `FaceFidelity` to `0.85` without clamping or validation errors.

#### Scenario: Out-of-bounds face fidelity clamping
- **GIVEN** a face restoration request with `--fidelity` outside the $[0.0, 1.0]$ range (e.g., `-0.2` or `1.5`)
- **WHEN** `ImageSpec.ApplyDefaults()` is executed
- **THEN** the system MUST clamp values $< 0.0$ to `0.0` and values $> 1.0$ to `1.0`.

#### Scenario: Face restoration disabled by default
- **GIVEN** a standard upscaling or text2img request without `--restore-faces`
- **WHEN** the spec is evaluated
- **THEN** `RestoreFaces` MUST be `false` and face restoration nodes/endpoints MUST NOT be invoked.

---

### REQ-UPSCALE-3: Multi-Backend Upscaling Payload Construction
The system's image generation adapters MUST construct backend-compliant payloads and workflows corresponding to the target backend's API constraints and capabilities.

#### Scenario: Fal.ai super-resolution and face restoration endpoint dispatch
- **GIVEN** the target backend `falai` with an upscale request (`ScaleFactor: 4`, `RestoreFaces: true`, `FaceFidelity: 0.8`)
- **WHEN** `FalAIBackend.Generate()` is called
- **THEN** the adapter MUST route to the appropriate Fal.ai upscaling model endpoint (e.g., `fal-ai/esrgan`, `fal-ai/aura-sr`, or `fal-ai/creative-upscaler`), supplying the input image URL/data URI, scale multiplier `4`, and face enhancement parameters.

#### Scenario: ComfyUI node graph assembly with Upscale and Face Restore models
- **GIVEN** the target backend `comfyui` with `ScaleFactor: 4` and `RestoreFaces: true`
- **WHEN** `ComfyUIBackend.Generate()` builds the execution graph
- **THEN** the adapter MUST upload the source image via `/upload/image`, inject `LoadImage`, `UpscaleModelLoader` / `UpscaleImageUsingModel` nodes, chain an `ApplyFaceRestoreModel` (CodeFormer or GFPGAN) node with fidelity weight, and terminate with `SaveImage`.

#### Scenario: OpenAI backend graceful rejection for standalone upscaling
- **GIVEN** the target backend `openai` with an upscaling request (`Mode: ModeUpscale`)
- **WHEN** `OpenAIBackend.Generate()` is called
- **THEN** the adapter MUST return a descriptive error indicating that OpenAI DALL-E does not support standalone super-resolution upscaling or face restoration.

#### Scenario: Pollinations backend URL query formatting or graceful fallback
- **GIVEN** the target backend `pollinations` with an upscaling request
- **WHEN** `PollinationsBackend.Generate()` is called
- **THEN** if image-to-image/upscaling parameters are supported by the gateway, it MUST append the required query parameters; otherwise, it MUST return an actionable error stating backend limitation.

---

### REQ-UPSCALE-4: Subagent `@upscaler` Natural Language Processing & Routing
The system MUST register a specialized `@upscaler` subagent in `domain.DefaultSubagents()` capable of parsing natural language upscaling and face-enhancement instructions into structured `ImageSpec` configurations.

#### Scenario: Subagent registration in default visual specialists
- **GIVEN** the ARIS subagent registry initialized via `domain.DefaultSubagents()`
- **WHEN** the list of subagents is inspected
- **THEN** it MUST contain an entry for `upscaler` with role `"Super-Resolution & Face Restoration Specialist"`, appropriate system prompt, and tools `["upscale", "restore_faces", "evaluate_fidelity"]`.

#### Scenario: Natural language upscale intent parsing
- **GIVEN** a user prompt: `"@upscaler please upscale /tmp/portrait.png to 4k resolution and fix the face artifacts with high fidelity"`
- **WHEN** the `@upscaler` subagent or LLM reasoner interprets the request
- **THEN** the resulting `ImageSpec` MUST populate `InputImagePath: "/tmp/portrait.png"`, `Mode: ModeUpscale`, `ScaleFactor: 4`, `RestoreFaces: true`, and `FaceFidelity: 0.75` (or higher).

#### Scenario: Multi-agent pipeline handoff to `@upscaler`
- **GIVEN** an image generation pass orchestrated by `SubagentManager` where `@critic` detects low face clarity or sub-2K resolution
- **WHEN** post-processing steps are determined
- **THEN** the manager MUST route the intermediate image result to `@upscaler` to perform face restoration and 2x/4x super-resolution before delivering the final asset.

---

### REQ-UPSCALE-5: CLI Subcommand `aris upscale` Flag Parsing & Output Metadata
The system MUST provide an intuitive CLI subcommand `aris upscale <image_path> [options]` that parses command line flags and displays rich progress and metadata summaries.

#### Scenario: Successful execution of `aris upscale` command
- **GIVEN** a valid local image file `photo.png`
- **WHEN** the user executes `aris upscale photo.png --scale 4 --restore-faces --fidelity 0.80 -b falai`
- **THEN** the CLI MUST:
  1. Parse `--scale 4`, `--restore-faces`, `--fidelity 0.80`, and `-b falai`.
  2. Validate `photo.png` existence and format.
  3. Dispatch generation to the selected backend.
  4. Save the upscaled output to disk.
  5. Print a metadata summary including original dimensions, new dimensions, elapsed time, and output path.

#### Scenario: CLI argument parsing with shorthand flags
- **GIVEN** shorthand arguments: `aris upscale input.jpg -s 2 -f 0.9 -b comfyui -o /tmp/output.png`
- **WHEN** the flag set is parsed
- **THEN** `-s 2` MUST map to `ScaleFactor = 2`, `-f 0.9` MUST map to `FaceFidelity = 0.9` with `RestoreFaces = true`, `-b comfyui` MUST map to `Backend = "comfyui"`, and `-o` MUST set the explicit target path.

#### Scenario: CLI invocation with missing positional image argument
- **GIVEN** the user runs `aris upscale` without an image path argument
- **WHEN** `handleUpscale` executes
- **THEN** it MUST exit with return code `1` and print usage instructions for `aris upscale <image_path> [options]`.

---

### REQ-UPSCALE-6: Input Image Validation & Error Handling
The system MUST validate input image paths, formats, dimensions, and file sizes prior to dispatching upscale jobs to prevent backend errors and resource exhaustion.

#### Scenario: Valid input image file validation
- **GIVEN** a local file path pointing to a readable `.png`, `.jpg`, `.jpeg`, or `.webp` file under 25 MB
- **WHEN** `pkg/imgutil` validation is performed
- **THEN** the validation MUST succeed, returning detected dimensions and MIME type.

#### Scenario: Non-existent or unreadable input image file
- **GIVEN** an image path pointing to a non-existent path `/tmp/missing.png`
- **WHEN** validation is executed
- **THEN** the system MUST reject the request immediately with `image file not found: /tmp/missing.png` without invoking remote APIs.

#### Scenario: Unsupported image file format
- **GIVEN** an input file with an unsupported extension or MIME type (e.g. `.gif`, `.bmp`, `.tiff`, `.exe`)
- **WHEN** validation is executed
- **THEN** the system MUST return a validation error stating the format is unsupported.

#### Scenario: Dimension and memory protection on 8x scale requests
- **GIVEN** an input image with dimensions exceeding $2048 \times 2048$ and a requested `ScaleFactor` of `8`
- **WHEN** `ImageSpec.Validate()` or adapter pre-flight runs
- **THEN** the system MUST warn or reject if estimated memory allocation exceeds safety thresholds, preventing out-of-memory crashes on local backends.
