# Specification: ARIS Img2Img & Visual Reference Pipeline

## Purpose
Extend the ARIS core architecture, image processing utilities, multi-backend adapters, visual subagents, and CLI interface to support image-to-image (img2img), style reference transfer, and masked inpainting workflows across multiple backends (Fal.ai, ComfyUI, Pollinations, OpenAI).

---

## User Stories & Acceptance Criteria

### REQ-IMG2IMG-1: Reference Image Input Validation
The system MUST support and validate reference images supplied as local file paths or remote HTTP/HTTPS URLs.

#### Scenario: Valid local reference image
- **GIVEN** a local file path to an existing image in PNG, JPEG, or WEBP format within acceptable size bounds (<= 25 MB)
- **WHEN** the image is validated and prepared by `pkg/imgutil`
- **THEN** the system MUST successfully detect its MIME type, resolve absolute file path, and convert it to a valid base64 data URI or raw byte stream as required by downstream adapters.

#### Scenario: Non-existent or invalid local reference image path
- **GIVEN** a file path pointing to a non-existent file or an unsupported file format (e.g. `.gif`, `.bmp`, `.exe`)
- **WHEN** validation is executed
- **THEN** the system MUST reject the input and return a descriptive validation error without invoking the generation backend.

#### Scenario: Valid remote image URL
- **GIVEN** a valid HTTP/HTTPS URL pointing to an image
- **WHEN** the reference image is processed
- **THEN** the system MUST either pass the URL directly to backends supporting remote URLs (e.g. Fal.ai) or fetch and buffer the image bytes with a 15-second timeout and 25 MB payload limit before encoding.

#### Scenario: Image exceeding size bounds
- **GIVEN** an input image exceeding the 25 MB size threshold
- **WHEN** the validation step runs
- **THEN** the system MUST reject the image with an error indicating the file size limit has been exceeded.

---

### REQ-IMG2IMG-2: Inpainting Mask Input Validation & Dimension Matching
The system MUST validate optional mask image inputs for inpainting workflows and verify dimensional alignment with the base reference image.

#### Scenario: Valid mask with identical dimensions
- **GIVEN** a valid reference image of dimension $W \times H$ and a mask image of dimension $W \times H$
- **WHEN** inpainting parameters are validated locally
- **THEN** the system MUST mark the spec mode as `inpaint` and prepare both the reference image and mask image payloads for the target backend.

#### Scenario: Dimension mismatch between base image and mask
- **GIVEN** a reference image of dimension $1024 \times 1024$ and a mask image of dimension $512 \times 512$
- **WHEN** dimension validation is executed
- **THEN** the system MUST reject the request locally with an error stating the dimension mismatch and expected dimensions.

#### Scenario: Mask supplied without reference image
- **GIVEN** a `--mask` input provided without a primary reference image
- **WHEN** command or spec validation executes
- **THEN** the system MUST return a validation error stating that a mask requires a base reference image.

---

### REQ-IMG2IMG-3: Denoise Strength Parameter Validation
The system MUST accept a denoise strength parameter governing the degree of divergence from the source image.

#### Scenario: Default denoise strength
- **GIVEN** an img2img or edit request without an explicit `--strength` flag
- **WHEN** the `domain.ImageSpec` is constructed
- **THEN** the system MUST default `DenoiseStrength` to `0.70`.

#### Scenario: Explicit valid denoise strength
- **GIVEN** an explicit strength value between `0.0` and `1.0` (e.g. `0.45`)
- **WHEN** the parameter is parsed
- **THEN** `DenoiseStrength` MUST be set to the exact provided value.

#### Scenario: Out-of-bounds denoise strength clamping or validation
- **GIVEN** a strength value less than `0.0` or greater than `1.0` (e.g. `1.5` or `-0.2`)
- **WHEN** the parameter is validated
- **THEN** the system MUST either clamp the value to the `[0.0, 1.0]` boundary or return an actionable validation error indicating the valid range.

---

### REQ-IMG2IMG-4: Multi-Backend Payload Formatting & Execution
The system image adapters MUST construct backend-compliant payloads and handle img2img / inpaint capabilities according to backend protocol constraints.

#### Scenario: Fal.ai Flux Dev img2img and inpainting execution
- **GIVEN** target backend `falai` with an img2img or inpaint request
- **WHEN** the adapter constructs the API request
- **THEN** it MUST route to `fal-ai/flux/dev/image-to-image` or `fal-ai/flux-general/inpainting` (or equivalent endpoint), supplying `image_url`, `mask_url`, `strength`, and `prompt`.

#### Scenario: ComfyUI LoadImage and Inpaint node graph dispatch
- **GIVEN** target backend `comfyui` with local image and optional mask
- **WHEN** the adapter constructs the workflow graph
- **THEN** it MUST upload the image/mask via ComfyUI `/upload/image` API, insert `LoadImage` and `VAEEncodeForInpaint` or `SetLatentNoiseMask` nodes, and connect the latent denoise strength to `KSampler`.

#### Scenario: OpenAI DALL-E 2 edit execution
- **GIVEN** target backend `openai` with model `dall-e-2`, an input image, and optional mask
- **WHEN** the adapter submits the generation request
- **THEN** it MUST format the payload as multipart/form-data to `POST https://api.openai.com/v1/images/edits` with square PNG images.

#### Scenario: Pollinations graceful fallback or image URL query
- **GIVEN** target backend `pollinations` with an img2img request
- **WHEN** the adapter formats the GET request
- **THEN** if image-to-image is supported via URL parameter (or image reference parameter), it MUST append the reference URI; otherwise it MUST return an explicit error stating inpainting is unsupported on Pollinations rather than silently corrupting output.

---

### REQ-IMG2IMG-5: Subagent Routing (`@inpainter` and `@restyler`)
The system MUST support specialized subagents `@inpainter` and `@restyler` that automatically configure `domain.ImageSpec` parameters and prompt refinements.

#### Scenario: `@inpainter` subagent execution
- **GIVEN** user prompt dispatched with `@inpainter` containing target prompt, base image, and mask
- **WHEN** `SubagentManager` processes the request
- **THEN** it MUST assign system prompt instructions specialized in seamless background/foreground blending, set `ReferenceMode` to `inpaint`, and preserve high denoise strength on masked regions.

#### Scenario: `@restyler` subagent execution
- **GIVEN** user prompt dispatched with `@restyler` containing target prompt and base reference image
- **WHEN** `SubagentManager` processes the request
- **THEN** it MUST assign system prompt instructions for artistic style transformation, set `ReferenceMode` to `style_transfer` / `img2img`, and default denoise strength to `0.65` unless overridden.

---

### REQ-IMG2IMG-6: CLI Command `aris edit` Execution & Flag Parsing
The system CLI MUST provide a dedicated subcommand `aris edit` to invoke the reference pipeline.

#### Scenario: Standard `aris edit` invocation
- **GIVEN** command line invocation `aris edit input.png "cyberpunk neon overhaul" --strength 0.65 --backend falai`
- **WHEN** the command executes
- **THEN** the CLI MUST resolve `input.png`, validate flags, construct `domain.ImageSpec`, invoke the pipeline, and output the generated image to `~/.aris/outputs/YYYY-MM-DD/aris_*.jpg`.

#### Scenario: Inpaint `aris edit` invocation with mask
- **GIVEN** command line invocation `aris edit portrait.png "remove glasses" --mask mask.png --backend comfyui`
- **WHEN** the command executes
- **THEN** the CLI MUST validate both `portrait.png` and `mask.png`, confirm dimension compatibility, and dispatch the inpaint task to `comfyui`.

#### Scenario: Missing required positional arguments
- **GIVEN** command line invocation `aris edit` without image path or prompt
- **WHEN** flag parsing runs
- **THEN** the CLI MUST display usage instructions and exit with non-zero exit code.

---

### REQ-IMG2IMG-7: Error Handling & Diagnostic Messages
The system MUST provide clear, actionable diagnostic messages for file, backend, or parameter failures.

#### Scenario: Unsupported backend for inpainting
- **GIVEN** a request with `--mask` targeting a backend that does not support inpainting
- **WHEN** adapter capability check runs
- **THEN** the system MUST return an error: `backend '<name>' does not support masked inpainting; please use falai, comfyui, or openai`.

#### Scenario: Backend API failure during img2img execution
- **GIVEN** a network timeout or rejected image from the remote backend
- **WHEN** the backend returns an error response
- **THEN** the system MUST log the failure context, capture error details, and return a user-friendly error specifying the backend failure reason.
