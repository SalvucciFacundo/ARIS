# Apply Progress: ARIS Batch Generation, Prompt Matrix & A/B Benchmarking

## Overview
- **Change**: `batch-generation-prompt-matrix`
- **Delivery Strategy**: `auto-chain` (Chained PRs recommended: Yes, stacked-to-main)
- **Status**: Completed (All 4 PR Work Units Implemented & Verified)
- **Mode**: Strict TDD (`RED -> GREEN -> REFACTOR`)

---

## TDD Cycle Evidence

| Phase / Work Unit | Red Test File | Green Implementation File | Refactor / Verification Evidence | Outcome |
|---|---|---|---|---|
| **PR 1: Combinatorial Prompt Matrix Engine** | `internal/core/services/matrix_test.go` | `internal/core/services/matrix.go` | Added multi-group, escaped brackets (`\[...\]`), safety limit tests. Race-detector clean. | ✅ PASS |
| **PR 2: Batch Runner & Worker Pool** | `internal/core/services/batch_runner_test.go` | `internal/core/services/batch_runner.go` | Concurrency worker pool, backend VRAM rate limiting (comfyui=1), fail-soft resilience, context cancellation. Tested with `go test -race`. | ✅ PASS |
| **PR 3: HTML & Markdown Contact Sheet Exporters** | `internal/core/services/contact_sheet_test.go` | `internal/core/services/contact_sheet.go` | Dark cyberpunk theme HTML, schema-compliant JSON (`batch_meta.json`), Markdown table with aggregate stats (`summary.md`). | ✅ PASS |
| **PR 4: CLI Subcommand `aris batch`, E2E & Docs** | `test/integration/batch_e2e_test.go`, `internal/adapters/ui/cli/batch_test.go` | `internal/adapters/ui/cli/batch.go`, `internal/adapters/ui/cli/cli.go` | Flag validation, mutual exclusion (`--count` vs `--seed-sweep`), dry-run preview, progress reporting, CLI docs updated. | ✅ PASS |

---

## Completed Tasks

- [x] **PR 1: Combinatorial Prompt Matrix Engine**
  - [x] **RED**: Unit tests in `internal/core/services/matrix_test.go` covering single group expansion, Cartesian products, escaped literal brackets (`\[...\]`), and safety limits.
  - [x] **GREEN**: AST node parser & `MatrixEngine.Expand` in `internal/core/services/matrix.go`.
  - [x] **REFACTOR**: Optimized allocations, boundary safety, empty string validation.
- [x] **PR 2: Concurrency-Controlled Batch Runner & Worker Pool**
  - [x] **RED**: Concurrent tests in `internal/core/services/batch_runner_test.go` using mock backends, rate limiting, and failure simulations.
  - [x] **GREEN**: `BatchRunner`, worker pools, backend semaphores, and telemetry aggregation in `internal/core/services/batch_runner.go`.
  - [x] **REFACTOR**: Bounded concurrency, race-free telemetry synchronization (`go test -race`).
- [x] **PR 3: HTML & Markdown Contact Sheet Exporters**
  - [x] **RED**: Schema tests in `internal/core/services/contact_sheet_test.go` for `batch_meta.json`, `summary.md`, and `index.html`.
  - [x] **GREEN**: `ContactSheetExporter` in `internal/core/services/contact_sheet.go`.
  - [x] **REFACTOR**: Embedded responsive CSS dark theme grid, error callout cards, Markdown backend aggregates.
- [x] **PR 4: CLI Subcommand `aris batch`, Progress Terminal Renderer, E2E Tests & Docs**
  - [x] **RED**: E2E integration tests in `test/integration/batch_e2e_test.go` and CLI unit tests in `internal/adapters/ui/cli/batch_test.go`.
  - [x] **GREEN**: `handleBatch` in `internal/adapters/ui/cli/batch.go` and routing in `cli.go`.
  - [x] **REFACTOR**: Updated `docs/cli.md`, verified exit codes, ran full test suite with `-race`.

---

## Files Changed

### Core Services:
- `internal/core/services/matrix.go` (new)
- `internal/core/services/matrix_test.go` (new)
- `internal/core/services/batch_runner.go` (new)
- `internal/core/services/batch_runner_test.go` (new)
- `internal/core/services/contact_sheet.go` (new)
- `internal/core/services/contact_sheet_test.go` (new)

### CLI & UI Adapters:
- `internal/adapters/ui/cli/batch.go` (new)
- `internal/adapters/ui/cli/batch_test.go` (new)
- `internal/adapters/ui/cli/cli.go` (updated routing & help text)

### Integration Tests & Documentation:
- `test/integration/batch_e2e_test.go` (new)
- `docs/cli.md` (updated with `aris batch` section and examples)
- `openspec/changes/batch-generation-prompt-matrix/tasks.md` (updated checkboxes)

---

## Verification Commands & Evidence
- `go test -v ./internal/core/services/...`: All tests pass.
- `go test -v ./internal/adapters/ui/cli/...`: All CLI unit tests pass.
- `go test -v ./test/integration/... -run TestBatch`: All E2E integration tests pass.
- `go test -race ./...`: Clean execution, zero data races, 100% test pass across all packages.
