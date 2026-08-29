# Archive Report: ARIS LoRA & ControlNet Manager

## Metadata
- **Change Name**: `lora-controlnet-manager`
- **Milestone**: Milestone 4 (v1.3.0)
- **Date**: 2026-08-29
- **Status**: Completed & Verified (100% PASS, 0 race conditions)
- **Architecture**: Hexagonal (Ports & Adapters)
- **Artifact Store**: Hybrid (OpenSpec + Engram)

---

## Executive Summary
The ARIS LoRA Weight Stacking & ControlNet Structural Conditioning subsystem has been designed, implemented under Strict TDD, and verified with race detection across all 4 planned PR work units.

### Key Capabilities Delivered:
1. **LoRA Weight Stacking & Prompt Tag Parsing (`pkg/prompt/parser.go`)**:
   - Extraction and sanitization of inline prompt tags `<lora:name:scale>`.
   - Support for stacking multiple LoRAs via CLI flags (`--lora "name:scale"`).
   - Scale factor validation and clamping `[0.0 - 2.0]` with default `1.0`.

2. **Pure-Go Canny Edge Detection Preprocessor (`pkg/imgutil/controlnet.go`)**:
   - Zero-dependency 5-stage Canny preprocessor (Grayscale $\to$ 5x5 Gaussian blur $\to$ Sobel operators $\to$ Non-Maximum Suppression $\to$ Double Thresholding & Hysteresis).
   - Generates preprocessed edge maps locally without external Python/CGO dependencies.

3. **Multi-Backend Dynamic Node Chaining**:
   - **ComfyUI**: Dynamically chains Model & CLIP inputs through sequential `LoraLoader` nodes, and pipes conditioning through `ApplyControlNet` + `ControlNetLoader` + `LoadImage` nodes.
   - **Fal.ai**: Formats `loras: [{ path, scale }]` and `controlnet: [{ type, image_url, strength }]` payloads.
   - **Pollinations & OpenAI**: Graceful fallbacks and metadata recording.

4. **CLI Subcommands & Flags**:
   - `--lora` and `--controlnet` flags available on `aris gen`, `aris edit`, and `aris batch`.
   - Subcommands `aris lora list` and `aris controlnet preproc <image>`.
   - Documentation in `docs/cli.md` and `docs/roadmap.md`.
