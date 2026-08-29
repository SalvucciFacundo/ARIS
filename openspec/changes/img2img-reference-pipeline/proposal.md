# SDD Proposal: ARIS Img2Img & Visual Reference Pipeline

## Problem Statement
ARIS is currently restricted to text-to-image generation. Users cannot perform image-to-image transformations, style transfers based on visual references, or localized inpainting with masks, which limits the platform's utility for advanced image composition and editing workflows.

## Target Users and Situations
CLI users and automated workflows that need to modify existing images, apply targeted restyling, or inpaint missing/masked sections directly through the ARIS pipeline without relying on heavy external GUI tools.

## Proposed Solution
Extend the ARIS domain types, image backend adapters, `pkg/imgutil` helpers, `AgentService` / `SubagentManager`, and the CLI layer to support input images, masks, and denoise strength parameters natively across multiple generation backends.

## Key Features
1. **`domain.ImageSpec` extension**: Introduce new fields including `InputImagePath`, `MaskImagePath`, `DenoiseStrength`, and `ReferenceMode` (e.g., img2img, inpaint, style_transfer).
2. **Image loading & validation utility**: Enhance `pkg/imgutil` to verify file existence, validate supported formats (PNG, JPEG, WEBP), and handle conversions to base64 data URIs.
3. **Backend support integration**:
   - **Fal.ai**: Support for Flux Dev img2img/inpaint.
   - **ComfyUI**: Native LoadImage and Inpaint node generation.
   - **OpenAI & Pollinations**: Implement compatible image payload formatting or graceful fallbacks.
4. **Specialized subagent updates**: Add `@inpainter` and `@restyler` presets to leverage the new pipeline features.
5. **New CLI subcommand**: Introduce `aris edit <image_path> "<prompt>" [options]` with flags such as `--strength`, `--mask`, `--ratio`, and `--backend`.

## Alternatives Considered
- **Dedicated external image preprocessing service**: Rejected in favor of in-process Go utilities (`pkg/imgutil`) to keep the ARIS deployment architecture simple and self-contained, avoiding additional external service dependencies.

## Risks & Mitigations
- **Local file path resolution**: Mitigated by strict file validation and absolute path resolution within the CLI layer before passing control to the pipeline.
- **Memory overhead of large base64 strings**: Mitigated by limiting maximum input file sizes or implementing downscaling for massive images before base64 encoding.
- **Backend API discrepancies**: Mitigated by pushing API-specific payload construction (e.g., base64 vs. multipart uploads vs. CDN URLs) down to the individual adapter implementation (`ports.ImageBackend`).

## Out of Scope (Non-goals)
- Interactive graphical canvas lasso/brush selection functionality. This is strictly reserved for the future Wails v2 Desktop App frontend; the CLI will accept pre-authored image masks.

---

## Proposal Question Round
*Please review these assumptions before we finalize the specification and design:*

1. **Payload Limits**: Are there explicit file size or resolution limits we need to enforce locally before passing base64 payloads to backends like Fal.ai or OpenAI to prevent OOM errors or API timeouts?
2. **Dimension Matching**: When a user supplies both a reference image and a mask for inpainting, should the ARIS pipeline enforce exact dimension matching locally, or pass them as-is and let the backend API reject mismatches?
3. **Pollinations Fallback**: If Pollinations lacks robust support for advanced masked inpainting via its GET API, should we default to a structural failure, or silently fallback to basic text-to-image/img2img?
4. **Base64 vs Uploads**: Should we assume base64 data URIs are acceptable for all adapters for this iteration, or do we need an intermediate upload-to-CDN step for backends that only accept remote URLs?
