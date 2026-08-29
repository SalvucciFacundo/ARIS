# Archive Report: ARIS Upscaling & Face Restoration Pipeline

## Metadata
- **Change Name**: `upscaling-face-restoration`
- **Milestone**: Milestone 3 (v1.2.0)
- **Date**: 2026-08-29
- **Status**: Completed & Verified (100% PASS, 0 race conditions)
- **Architecture**: Hexagonal (Ports & Adapters)
- **Artifact Store**: Hybrid (OpenSpec + Engram)

---

## Executive Summary
The ARIS Upscaling & Face Restoration Pipeline has been designed, implemented under Strict TDD, and verified with race detection across all 4 planned PR work units.

### Key Capabilities Delivered:
1. **Domain Extensions**:
   - `domain.ImageSpec`: `ScaleFactor int` (2x, 4x, 8x), `RestoreFaces bool`, `FaceFidelity float64` (default `0.75`), `UpscalerModel string`, and `IsUpscale() bool`.
   - `ReferenceMode`: Added `ModeUpscale`.

2. **Multi-Backend Super-Resolution & Face Restoration**:
   - **Fal.ai**: Endpoints `fal-ai/esrgan`, `fal-ai/aura-sr`, and `fal-ai/creative-upscaler` with face enhancer payload integration.
   - **ComfyUI**: Dynamic node graph construction chaining `UpscaleModelLoader` $\to$ `ImageUpscaleWithModel` $\to$ `ApplyFaceRestoreModel` (CodeFormer) $\to$ `SaveImage`.
   - **OpenAI & Pollinations**: Explicit rejection and graceful degradation.

3. **Subagent `@upscaler`**:
   - Registered in `domain.DefaultSubagents()` as the *"Super-Resolution & Face Restoration Specialist"*.

4. **CLI Subcommand `aris upscale`**:
   - Syntax: `aris upscale <image_path> [options]` with `--scale 2|4|8`, `--restore-faces`, `--fidelity <0.0-1.0>`, `-b/--backend`, `-m/--model`, `-o/--output`.
   - Full documentation in `docs/cli.md` and `docs/roadmap.md`.
