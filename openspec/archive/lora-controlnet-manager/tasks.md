## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1200-1600 |
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

# Implementation Tasks: ARIS LoRA & ControlNet Manager

This tasks document defines the step-by-step implementation plan for adding LoRA weight stacking and ControlNet structural conditioning across ARIS adapters and core domain, adhering strictly to **Strict TDD (RED -> GREEN -> REFACTOR)** and structured into 4 cohesive PR work units.

---

## PR 1: Domain Entities, Scaling Models, Tag Parser & Validation
*Scope:* Core domain updates (`internal/core/domain/types.go`), inline prompt parser (`pkg/prompt/parser.go`), unit tests, and validation.

- [x] **RED:** Write table-driven unit tests in `internal/core/domain/types_test.go` and `pkg/prompt/parser_test.go` for LoRA tag parsing, scale clamping bounds `[0.0, 2.0]`, default scale assignments, and ControlNet type whitelist validation. <!-- sdd-owner: implementation -->
- [x] **GREEN:** Implement `LoRAConfig`, `ControlNetConfig`, and extend `ImageSpec` with `ApplyDefaults()`, `Validate()`, `HasLoRA()`, and `HasControlNet()` in `internal/core/domain/types.go`. <!-- sdd-owner: implementation -->
- [x] **GREEN:** Implement regex-based inline LoRA tag extractor and cleaner in `pkg/prompt/parser.go`. <!-- sdd-owner: implementation -->
- [x] **REFACTOR & VERIFY:** Run unit tests with `-race` to ensure zero data races and correct parsing edge cases. <!-- sdd-owner: implementation -->

---

## PR 2: Pure-Go Canny Preprocessor & Image Utilities
*Scope:* Zero-dependency pure-Go Canny edge detection preprocessor in `pkg/imgutil/controlnet.go` and comprehensive unit tests.

- [x] **RED:** Write unit tests in `pkg/imgutil/controlnet_test.go` verifying synthetic image grayscale conversion, Gaussian blur convolution, Sobel gradients, Non-Maximum Suppression, and hysteresis linking. <!-- sdd-owner: implementation -->
- [x] **GREEN:** Implement the full pure-Go Canny edge detection pipeline in `pkg/imgutil/controlnet.go` without external dependencies. <!-- sdd-owner: implementation -->
- [x] **REFACTOR & VERIFY:** Execute `go test -v -race ./pkg/imgutil/...` to confirm mathematical validity and execution speed. <!-- sdd-owner: implementation -->

---

## PR 3: Multi-Backend Dynamic Graph Chaining
*Scope:* Backend adapter updates for ComfyUI dynamic LoRA/ControlNet graph generation (`internal/adapters/image/comfyui.go`), Fal.ai payload mapping (`internal/adapters/image/falai.go`), and graceful degradation for Pollinations/OpenAI.

- [x] **RED:** Write unit tests in `internal/adapters/image/comfyui_test.go` and `falai_test.go` to verify dynamic node chaining for $N$ LoRAs and ControlNets. <!-- sdd-owner: implementation -->
- [x] **GREEN:** Implement dynamic node chaining for `ComfyUIBackend` in `internal/adapters/image/comfyui.go` and payload structuring for `FalAIBackend` in `falai.go`. <!-- sdd-owner: implementation -->
- [x] **GREEN:** Implement graceful fallback / warning warnings for unsupported backends (`pollinations.go`, `openai.go`). <!-- sdd-owner: implementation -->
- [x] **REFACTOR & VERIFY:** Run backend adapter unit tests and race detector. <!-- sdd-owner: implementation -->

---

## PR 4: CLI Subcommands & Flags, E2E Integration Tests & Docs
*Scope:* CLI flag bindings (`--lora`, `--controlnet`), subcommands (`aris lora`, `aris controlnet`), E2E tests, and documentation.

- [x] **RED:** Write integration tests in `test/integration/lora_controlnet_e2e_test.go` simulating end-to-end command parsing, preprocessor triggering, and mock backend generation. <!-- sdd-owner: implementation -->
- [x] **GREEN:** Implement CLI flags and subcommands in `internal/adapters/ui/cli/lora.go` and `controlnet.go`. <!-- sdd-owner: implementation -->
- [x] **GREEN:** Update user documentation in `docs/cli.md` covering LoRA stacking and ControlNet usage examples. <!-- sdd-owner: implementation -->
- [x] **REFACTOR & VERIFY:** Execute full test suite `go test ./...` and verify clean build. <!-- sdd-owner: implementation -->
