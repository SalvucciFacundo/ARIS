# Apply Progress: ARIS Upscaling & Face Restoration Pipeline

**Change Name:** `upscaling-face-restoration`  
**Artifact Store Mode:** `hybrid`  
**Delivery Strategy:** `auto-chain`  
**Chain Strategy:** `stacked-to-main`  
**Status:** `all_done`

---

## Completed Tasks & Checkbox Status

### PR 1: Domain Entities, Scaling Models & Validation (`upscaling-face-restoration-pr1`)
- [x] **Task 1.1 (RED)**: Unit tests for `ImageSpec` scaling (`ScaleFactor` 2, 4, 8 vs invalid scales), face restoration defaults, `FaceFidelity` clamping (`[0.0, 1.0]`), and `IsUpscale()` helper in `internal/core/domain/types_test.go`.
- [x] **Task 1.2 (GREEN)**: Domain extensions in `internal/core/domain/types.go` (`ScaleFactor`, `RestoreFaces`, `FaceFidelity`, `UpscalerModel`, `ModeUpscale`, `IsUpscale()`, `ApplyDefaults()`, `Validate()`).
- [x] **Task 1.3 (REFACTOR)**: Domain validation error message cleanups and thread-safe validations verified with `go test -race ./internal/core/domain/...`.

### PR 2: Multi-Backend Upscaling Adapters (`upscaling-face-restoration-pr2`)
- [x] **Task 2.1 (RED)**: Mock-based unit tests in `internal/adapters/image/backends_test.go` covering Fal.ai and ComfyUI super-resolution and face restoration payload structures, and unsupported backend rejections.
- [x] **Task 2.2 (GREEN)**: Multi-backend adapter upscaling implementations in `internal/adapters/image/falai.go`, `internal/adapters/image/comfyui.go`, `internal/adapters/image/openai.go`, and `internal/adapters/image/pollinations.go`.
- [x] **Task 2.3 (REFACTOR)**: Refactored HTTP payload serialization and mock server tests verified with `go test -race ./internal/adapters/image/...`.

### PR 3: `@upscaler` Subagent & Service Layer (`upscaling-face-restoration-pr3`)
- [x] **Task 3.1 (RED)**: Unit tests in `internal/core/services/agent_test.go` verifying `@upscaler` subagent registration, properties, and `GenerateOptions` scaling mappings.
- [x] **Task 3.2 (GREEN)**: Registered `@upscaler` subagent in `internal/core/domain/subagent.go` and updated `internal/core/services/agent.go` and `internal/core/services/subagent_manager.go` with scaling options and pipeline routing.
- [x] **Task 3.3 (REFACTOR)**: Refactored pipeline post-processing fallback and verified with `go test -race ./internal/core/services/...`.

### PR 4: CLI Subcommand `aris upscale`, E2E Integration Suite & Documentation (`upscaling-face-restoration-pr4`)
- [x] **Task 4.1 (RED)**: CLI command tests in `internal/adapters/ui/cli/upscale_test.go` and E2E integration test suite in `test/integration/upscale_e2e_test.go`.
- [x] **Task 4.2 (GREEN)**: Implemented `aris upscale` subcommand in `internal/adapters/ui/cli/upscale.go`, wired into `internal/adapters/ui/cli/cli.go`, and updated `docs/cli.md` & `docs/roadmap.md`.
- [x] **Task 4.3 (REFACTOR)**: Ran full repository test suite (`go test -race ./...`) confirming 100% test pass with zero race conditions.

---

## Files Changed

- `internal/core/domain/types.go`
- `internal/core/domain/types_test.go`
- `internal/core/domain/subagent.go`
- `internal/adapters/image/falai.go`
- `internal/adapters/image/comfyui.go`
- `internal/adapters/image/openai.go`
- `internal/adapters/image/pollinations.go`
- `internal/adapters/image/backends_test.go`
- `internal/core/services/agent.go`
- `internal/core/services/subagent_manager.go`
- `internal/core/services/agent_test.go`
- `internal/adapters/ui/cli/cli.go`
- `internal/adapters/ui/cli/upscale.go`
- `internal/adapters/ui/cli/upscale_test.go`
- `test/integration/upscale_e2e_test.go`
- `docs/cli.md`
- `docs/roadmap.md`
- `openspec/changes/upscaling-face-restoration/tasks.md`
- `openspec/changes/upscaling-face-restoration/apply-progress.md`

---

## Strict TDD Cycle Evidence

| Phase / Work Unit | Target File | Test File | Test Status (RED -> GREEN -> REFACTOR) |
|---|---|---|---|
| **PR 1**: Domain Entities & Validation | `internal/core/domain/types.go` | `internal/core/domain/types_test.go` | Compilation failure (RED) -> Implement types & methods (GREEN) -> Pass `go test -race ./internal/core/domain/...` (REFACTOR) |
| **PR 2**: Multi-Backend Adapters | `internal/adapters/image/*.go` | `internal/adapters/image/backends_test.go` | Assertion failures on upscale endpoints & payload (RED) -> Implement Fal.ai/ComfyUI/OpenAI/Pollinations (GREEN) -> Pass `go test -race ./internal/adapters/image/...` (REFACTOR) |
| **PR 3**: `@upscaler` Subagent & Service | `internal/core/domain/subagent.go`, `internal/core/services/*.go` | `internal/core/services/agent_test.go` | Compilation failure on options (RED) -> Implement subagent def & manager/agent options (GREEN) -> Pass `go test -race ./internal/core/services/...` (REFACTOR) |
| **PR 4**: CLI & E2E Integration Suite | `internal/adapters/ui/cli/upscale.go`, `cli.go` | `internal/adapters/ui/cli/upscale_test.go`, `test/integration/upscale_e2e_test.go` | Failures on missing subcommand (RED) -> Implement `handleUpscale` & wire in `cli.go` (GREEN) -> Full suite pass `go test -race ./...` (REFACTOR) |

---

## Test Verification Commands

```bash
cd "/run/media/kuno/Disco local/Kuno/GO/ARIS"
go test -count=1 -race ./...
```
Output: All packages compiled cleanly and passed 100% of unit and integration tests with zero race conditions.

---

## Workload & PR Boundary

- **Delivery Strategy:** `auto-chain` (stacked-to-main)
- **Suggested PR Slices:**
  1. `PR 1`: Core domain primitives, scale factor validation, face restoration defaults (`types.go`, `types_test.go`)
  2. `PR 2`: Multi-backend upscaling adapters (`falai.go`, `comfyui.go`, `openai.go`, `pollinations.go`, `backends_test.go`)
  3. `PR 3`: `@upscaler` subagent and service layer orchestration (`subagent.go`, `agent.go`, `subagent_manager.go`, `agent_test.go`)
  4. `PR 4`: CLI subcommand `aris upscale`, E2E integration test suite and documentation (`upscale.go`, `cli.go`, `upscale_test.go`, `upscale_e2e_test.go`, `docs/cli.md`, `docs/roadmap.md`)
