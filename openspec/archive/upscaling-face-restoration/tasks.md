## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~650 lines |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 -> PR 2 -> PR 3 -> PR 4 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

---

## Task Plan & Breakdown (Stacked PR Chain)

### PR 1: Domain Entities, Scaling Models & Validation (`upscaling-face-restoration-pr1`)
*Focus: Core domain primitives, scale factor validation, face restoration bounds, and unit tests adhering to strict TDD.*

- [x] **Task 1.1 (RED)**: Write comprehensive unit tests in `internal/core/domain/types_test.go` covering `ImageSpec` scaling (`ScaleFactor` 2, 4, 8 vs invalid scales), face restoration defaults, `FaceFidelity` clamping (`[0.0, 1.0]`), and `IsUpscale()` helper behavior. <!-- sdd-owner: implementation -->
- [x] **Task 1.2 (GREEN)**: Implement domain extensions in `internal/core/domain/types.go`: add `ScaleFactor`, `RestoreFaces`, `FaceFidelity`, `UpscalerModel` fields, `ModeUpscale` constant, `IsUpscale()`, and update `ApplyDefaults()` and `Validate()`. Ensure PR 1 unit tests pass successfully. <!-- sdd-owner: implementation -->
- [x] **Task 1.3 (REFACTOR)**: Clean up domain validation error messages and ensure thread-safe struct copy behaviors for `ImageSpec` extensions. Run `go test -race ./internal/core/domain/...`. <!-- sdd-owner: implementation -->

---

### PR 2: Multi-Backend Upscaling Adapters (`upscaling-face-restoration-pr2`)
*Focus: Adapters for Fal.ai super-resolution endpoints and ComfyUI node graph generation with CodeFormer/GFPGAN face restoration.*

- [x] **Task 2.1 (RED)**: Write mock-based unit tests in `internal/adapters/image/backends_test.go` (and/or `falai_test.go`, `comfyui_test.go`) validating that `FalAIBackend` constructs valid super-resolution request payloads and that `ComfyUIBackend` dynamically assembles `UpscaleModelLoader`, `ImageUpscaleWithModel`, and `FaceRestoreCFWithModel` workflow nodes. <!-- sdd-owner: implementation -->
- [x] **Task 2.2 (GREEN)**: Implement upscale/face-restoration handling in `internal/adapters/image/falai.go` (routing to esrgan/aura-sr endpoints with face enhancement toggles) and `internal/adapters/image/comfyui.go` (image upload, node graph assembly, and prompt submission). Also implement graceful rejections for unsupported backends (OpenAI and Pollinations). <!-- sdd-owner: implementation -->
- [x] **Task 2.3 (REFACTOR)**: Refactor adapter HTTP error handling, payload serialization, and mock server tests. Run `go test -race ./internal/adapters/image/...`. <!-- sdd-owner: implementation -->

---

### PR 3: `@upscaler` Subagent & Service Layer (`upscaling-face-restoration-pr3`)
*Focus: Subagent registration, natural language intent parsing, and service-level routing for upscaling workflows.*

- [x] **Task 3.1 (RED)**: Write unit tests in `internal/core/services/agent_test.go` and subagent registry tests verifying `@upscaler` registration in `DefaultSubagents()`, system prompt correctness, and natural language intent extraction (mapping prompts like `"upscale to 4x with high fidelity face restoration"` into a valid `ImageSpec`). <!-- sdd-owner: implementation -->
- [x] **Task 3.2 (GREEN)**: Implement `@upscaler` subagent definition in `internal/core/domain/subagent.go` and update `internal/core/services/agent.go` / `subagent_manager.go` to support automatic visual specialist handoff and parameter population. <!-- sdd-owner: implementation -->
- [x] **Task 3.3 (REFACTOR)**: Refactor prompt parser matching logic and verify clean subagent interaction logs. Run `go test -race ./internal/core/services/...`. <!-- sdd-owner: implementation -->

---

### PR 4: CLI Subcommand `aris upscale`, E2E Integration Suite & Documentation (`upscaling-face-restoration-pr4`)
*Focus: CLI user-facing subcommand, comprehensive E2E integration test suite, and CLI documentation.*

- [x] **Task 4.1 (RED)**: Write CLI command unit tests in `internal/adapters/ui/cli/upscale_test.go` and E2E integration tests in `test/integration/upscale_e2e_test.go` verifying flag parsing (`--scale`, `--restore-faces`, `--fidelity`, `-b`), input image validation checks (non-existent file or unsupported format), and metadata output summaries. <!-- sdd-owner: implementation -->
- [x] **Task 4.2 (GREEN)**: Implement CLI subcommand `aris upscale` in `internal/adapters/ui/cli/upscale.go` (integrating image utility validation, backend generation, and console progress reporting). Write `docs/cli.md` updates detailing upscale usage and flags. <!-- sdd-owner: implementation -->
- [x] **Task 4.3 (REFACTOR)**: Run full test suite across the repository (`go test -race ./...`), verify zero race conditions or broken builds, and prepare final artifact closeout. <!-- sdd-owner: implementation -->
