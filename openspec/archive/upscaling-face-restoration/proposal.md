# SDD Proposal: Upscaling & Face Restoration Pipeline

## 1. Intent & Business Problem
Standard diffusion model outputs often result in limited native resolutions (typically 1024x1024) and frequently introduce facial artifacts—especially on complex, small, or distant characters in the scene. 

To achieve production-ready quality, users need a dedicated **super-resolution** engine (capable of 2x, 4x, and 8x upscaling) combined with a **face fidelity restoration** pipeline (utilizing models like CodeFormer or GFPGAN) seamlessly integrated into the ARIS ecosystem.

## 2. Proposed Solution
We propose introducing a native Upscaling & Face Restoration engine into ARIS. This entails extending our domain entities to support upscale parameters, upgrading image backends (Fal.ai, ComfyUI, Pollinations) to execute restoration models, introducing a specialized subagent (`@upscaler`) for natural language routing, and adding a dedicated CLI subcommand (`aris upscale`).

### Key Capabilities
1. **Scale Factor Support**: Deterministic multipliers for upscaling: 2x, 4x, 8x (e.g., `--scale 2|4|8`).
2. **Face Restoration Pipeline**: Dedicated toggles for facial reconstruction (`--restore-faces`) with configurable fidelity strength (`--fidelity 0.75`).
3. **Backend Adapters**: 
   - **Fal.ai**: Integration with endpoints like `fal-ai/esrgan` or `fal-ai/aura-sr`.
   - **ComfyUI**: Execution of node workflows containing `UpscaleImageUsingModel` and `ApplyFaceRestoreModel`.
4. **Specialized Subagent**: Registration of `@upscaler` within `domain.DefaultSubagents()` to process natural language upscaling requests.
5. **CLI Subcommand**: A new terminal command, `aris upscale <image_path> [options]`, providing direct access to the pipeline without needing a full generation pass.

## 3. Scope & Affected Areas

### In Scope
* Extending `ImageSpec` in `internal/core/domain/types.go` (`ScaleFactor`, `RestoreFaces`, `FaceFidelity`, `UpscalerModel`).
* Updating Backend Adapters (`internal/adapters/image/`) for Fal.ai and ComfyUI to handle upscaling nodes/endpoints.
* Adding the new `@upscaler` subagent logic in `internal/core/services/agent.go`.
* Creating the `aris upscale` CLI command in `internal/adapters/ui/cli/`.
* Updating `ApplyDefaults` to ensure `FaceFidelity` defaults safely (e.g., `0.75`) when `RestoreFaces` is activated.

### Out of Scope (Non-Goals)
* Building or hosting our own custom upscale/restoration models natively.
* Video upscaling or temporal consistency passes.
* Full integration of local standalone ESRGAN CLI binaries (we rely on existing adapter backends like ComfyUI and Fal.ai).

## 4. Risks, Tradeoffs & Alternatives

* **Risk: Backend Heterogeneity**: Not all backends interpret fidelity parameters the same way. (e.g., Fal.ai vs ComfyUI GFPGAN implementations).
  * *Mitigation*: We map the generic 0.0-1.0 `FaceFidelity` domain parameter to the closest sensible default per adapter.
* **Risk: Memory & Compute Spikes**: 8x scaling significantly increases latency and VRAM usage on local ComfyUI workers.
  * *Mitigation*: Fallback mechanisms or explicit warnings for 8x workloads on constrained hardware.
* **Alternative Considered**: Instead of extending `ImageSpec`, create a new distinct `UpscaleSpec`. 
  * *Decision*: Rejected. Upscaling is often desired as an immediate post-processing step to generation. Merging them into `ImageSpec` allows single-pass generation-and-upscale workflows in the future.

## 5. Rollback Strategy
If the integration breaks standard generation paths, the new fields (`ScaleFactor`, `RestoreFaces`) can be ignored by setting their defaults to 0/false, isolating the legacy generation path. The `@upscaler` subagent can be quickly deregistered from `domain.DefaultSubagents()`.

## 6. Success Criteria
* [ ] The `aris upscale <path> --scale 4 --restore-faces` command executes successfully against at least one backend (e.g., Fal.ai).
* [ ] The `@upscaler` agent parses natural language intent ("upscale this to 4k and fix the faces") correctly into the new domain fields.
* [ ] The ComfyUI adapter can correctly assemble and execute a node graph containing CodeFormer/GFPGAN when requested.
* [ ] Existing standard image generation tests pass without regression.
