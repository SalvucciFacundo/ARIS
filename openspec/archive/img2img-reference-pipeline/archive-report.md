# Archive Report: ARIS Img2Img & Visual Reference Pipeline

## Metadata
- **Change Name**: `img2img-reference-pipeline`
- **Date**: 2026-08-29
- **Status**: Completed & Verified (100% PASS)
- **Architecture**: Hexagonal (Ports & Adapters)
- **Artifact Store**: Hybrid (OpenSpec + Engram)

---

## Executive Summary
The ARIS Img2Img & Visual Reference Pipeline has been successfully designed, implemented following Strict TDD, and verified with race detection.

### Key Capabilities Delivered:
1. **Domain Extensions**:
   - `ReferenceMode` enum (`text2img`, `img2img`, `inpaint`, `style_transfer`, `upscale`).
   - `ImageSpec` extended with `InputImagePath`, `MaskImagePath`, `DenoiseStrength`, and `ReferenceMode`.
   - Domain defaults (`0.70` strength) and out-of-bounds clamping.

2. **Image & Mask Utilities (`pkg/imgutil`)**:
   - Pure-Go MIME detection and dimension extraction for PNG, JPEG, and WEBP.
   - Validation for file existence, size limits ($\le 25\text{MB}$), Base64 data URI formatting, and strict $W \times H$ mask dimension matching.

3. **Multi-Backend Adapters**:
   - **Fal.ai**: Automatic endpoint routing to `fal-ai/flux/dev/image-to-image` and `fal-ai/flux-general/inpainting`.
   - **ComfyUI**: Two-stage multipart image/mask upload with `LoadImage` and `VAEEncodeForInpaint` node graph generation.
   - **OpenAI**: Multipart `POST /v1/images/edits` support.
   - **Pollinations**: Parameterized reference URL support with graceful fallback.

4. **Visual Subagents**:
   - Registered `@inpainter` (inpainting and seamless blending specialist) and `@restyler` (style transfer specialist with `0.65` strength default).

5. **CLI & Documentation**:
   - Added `aris edit <image_path> "<prompt>" [options]` with `--strength`, `--mask`, `--ratio`, `--backend`, `--model`, `--critic`, and `--auto-heal`.
   - Comprehensive technical and user guide in `docs/img2img.md`.
