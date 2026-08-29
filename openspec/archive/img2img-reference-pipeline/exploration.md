# Exploration: ARIS Img2Img & Visual Reference Pipeline

## Overview
This exploration assesses the integration of image-to-image (Img2Img) and visual reference (inpainting, style transfer) capabilities into the ARIS image generation pipeline.

## Current Architecture
- **Domain Layer (`domain.ImageSpec`)**: Currently handles prompt text, dimensions, and standard generation parameters.
- **Port Layer (`ports.ImageBackend`)**: Defines the `Generate` interface.
- **Adapters**:
  - `PollinationsBackend`: Simple GET-based requests. Limited support for complex img2img.
  - `FalAIBackend`: Structured POST-based JSON requests, well-suited for extended parameters like img2img and inpainting.
- **Service Layer (`SubagentManager`)**: Coordinates the pipeline via `PipelineExecute`.

## Proposed Changes
1.  **Domain Update**:
    - Extend `domain.ImageSpec` with:
        - `InputImagePath` (string)
        - `MaskImagePath` (string, for inpainting)
        - `DenoiseStrength` (float64, 0.0-1.0)
        - `ReferenceMode` (enum: `img2img`, `inpaint`, `style_transfer`, `upscale`)
2.  **Backend Updates**:
    - `FalAIBackend` is the primary candidate for img2img support. The `Generate` method needs to check for `InputImagePath` and include it in the `reqPayload` sent to Fal.ai.
    - `PollinationsBackend` may require additional API research or might remain limited to text-to-image.
3.  **Service/CLI Updates**:
    - `SubagentManager.PipelineExecute`: Propagate new `ImageSpec` fields from `PipelineOptions`.
    - `internal/adapters/ui/cli/cli.go`: Extend CLI commands to expose these new options (`--strength`, `--mask`, `--mode`).
4.  **Utilities**:
    - `pkg/imgutil`: Enhance to handle mask preprocessing and image normalization (to base64 or optimized CDN upload).

## Risks
- **API Variability**: Each backend API handles img2img differently (URL vs. base64 vs. multipart). Need to abstract this in `ImageBackend` if possible, or implement per-backend logic.
- **UI/UX**: Managing local file paths for images/masks through CLI requires robust validation.

## Conclusion
The architecture is well-suited for this extension. The `SubagentManager`'s `PipelineOptions` structure naturally supports adding these new parameters.
