# Exploration: Upscaling & Face Restoration Pipeline

## Overview
This exploration outlines the necessary architectural and domain changes to integrate a Super-Resolution and Face Restoration pipeline into ARIS. This enables users to perform high-fidelity upscaling (2x, 4x, 8x) and restorative operations (CodeFormer/GFPGAN) as a standalone mode or post-processing step.

## Domain Model Changes (`internal/core/domain/types.go`)
- **`ImageSpec` Updates**:
  - Add `ScaleFactor float64` (e.g., 2.0, 4.0, 8.0).
  - Add `RestoreFaces bool` (flag for restoration pipeline).
  - Add `FaceFidelity float64` (0.0–1.0, e.g., 0.75).
  - Add `UpscalerModel string` (to specify model: RealESRGAN, 4x-UltraSharp, etc.).
- **Helper Extensions**:
  - Implement `IsUpscale() bool` on `ImageSpec`.
  - Ensure `ApplyDefaults()` correctly initializes upscaling defaults (e.g., if `RestoreFaces` is true, default `FaceFidelity` to 0.75).

## Backend Integration (`internal/adapters/image/`)
- **Fal.ai**: Extend `falai.go` to invoke upscaling endpoints (`fal-ai/esrgan`, `fal-ai/aura-sr`, etc.) when `ModeUpscale` is set.
- **ComfyUI**: Update `comfyui.go` to handle node graphs that incorporate `UpscaleImageUsingModel` and `ApplyFaceRestoreModel` (CodeFormer/GFPGAN).
- **Registry**: Ensure the `Registry` is aware of models capable of restoration so the orchestrator can filter or route appropriately.

## Orchestration Logic (`internal/core/services/agent.go`)
- **`AgentService`**:
  - Update `Generate` to pass the new upscale/restoration fields down to the backend `Generate()` call.
  - Registration of the `@upscaler` subagent in `domain.DefaultSubagents()`.
  - Logic to handle `ModeUpscale` routing if a user requests a direct upscale of an existing image.

## CLI Subcommand (`internal/adapters/ui/cli/`)
- **`aris upscale`**:
  - Implement a new CLI command `aris upscale <image_path>`.
  - Support flags:
    - `--scale`: 2, 4, 8.
    - `--restore-faces`: Toggle for CodeFormer/GFPGAN.
    - `--fidelity`: Set restoration fidelity (0.0–1.0).
  - Delegate the request to the `AgentService` with `ModeUpscale`.

## Risks
- **Memory/Bandwidth**: High-resolution upscaling (8x) can significantly impact memory usage on GPUs/local workers.
- **Backend Heterogeneity**: Not all backends (e.g., OpenAI vs ComfyUI) support the exact same restoration fidelity parameters.
- **Subagent Routing**: Ensuring `@upscaler` correctly interprets complex intent (e.g., "upscale with face restoration") vs. simple scaling.
