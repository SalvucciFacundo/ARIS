# Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1250-1600 lines |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 -> PR 2 -> PR 3 -> PR 4 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

```text
Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High
```

---

# Tasks: ARIS Batch Generation, Prompt Matrix & A/B Benchmarking

This tasks document follows strict TDD principles (`RED -> GREEN -> REFACTOR`) and is structured into 4 independent, reviewable pull request work units (`stacked-to-main`).

---

## PR 1: Combinatorial Prompt Matrix Engine
**Goal**: Implement the pure Cartesian product prompt parser supporting bracketed alternate tokens, escaped brackets, and maximum size safety limits.

### Tasks
- [x] **RED**: Write comprehensive unit tests in `internal/core/services/matrix_test.go` covering simple single group expansion, multi-group Cartesian products, escaped literal brackets (`\[...\]`), and matrix size bound validation (asserting error when exceeding `--max-matrix-jobs`). <!-- sdd-owner: implementation -->
- [x] **GREEN**: Implement the `MatrixEngine` and AST node parser in `internal/core/services/matrix.go` to satisfy all tests in `matrix_test.go`. <!-- sdd-owner: implementation -->
- [x] **REFACTOR**: Refactor `matrix.go` and `matrix_test.go` for clarity, performance, clean memory allocation, and robust error messages. <!-- sdd-owner: implementation -->

---

## PR 2: Concurrency-Controlled Batch Runner & Worker Pool
**Goal**: Implement the core batch execution engine supporting seed sweeps, N-count batching, multi-backend dispatch, concurrency throttling, fail-soft resilience, and context cancellation.

### Tasks
- [x] **RED**: Write comprehensive concurrent tests and unit tests in `internal/core/services/batch_runner_test.go` using mock backends (simulating latency, successes, and simulated failures like HTTP 429), verifying bounded worker concurrency, backend VRAM rate limiting, fail-soft error recovery, and graceful `SIGINT` cancellation. <!-- sdd-owner: implementation -->
- [x] **GREEN**: Implement `BatchRunner`, worker pool dispatchers, backend semaphores, and telemetry collectors in `internal/core/services/batch_runner.go`. <!-- sdd-owner: implementation -->
- [x] **REFACTOR**: Ensure race-detector clean execution (`go test -race ./internal/core/services/...`), clean cancellation semantics, and clear error logging. <!-- sdd-owner: implementation -->

---

## PR 3: HTML & Markdown Contact Sheet Exporters
**Goal**: Implement the self-contained output bundle generator exporting `index.html` (responsive dark theme grid), `summary.md` (tables & aggregate stats), and `batch_meta.json`.

### Tasks
- [x] **RED**: Write unit tests in `internal/core/services/contact_sheet_test.go` verifying correct generation and schema compliance of `batch_meta.json`, structure of `summary.md`, and self-contained HTML structure of `index.html`. <!-- sdd-owner: implementation -->
- [x] **GREEN**: Implement `ArtifactExporter` in `internal/core/services/contact_sheet.go` with embedded CSS and templating support. <!-- sdd-owner: implementation -->
- [x] **REFACTOR**: Clean up HTML templates, verify CSS responsiveness in dark mode, and optimize JSON serialization. <!-- sdd-owner: implementation -->

---

## PR 4: CLI Subcommand `aris batch`, Progress Terminal Renderer, E2E Tests & Docs
**Goal**: Implement the Cobra CLI command `aris batch` with flag parsing, validation, dry-run mode, real-time terminal progress reporting, E2E integration tests, and user documentation.

### Tasks
- [x] **RED**: Write integration and E2E tests in `test/integration/batch_e2e_test.go` covering CLI flag validation (`--count` vs `--seed-sweep` conflicts), dry-run execution, and full batch execution flow. <!-- sdd-owner: implementation -->
- [x] **GREEN**: Implement Cobra command `aris batch` in `internal/adapters/ui/cli/batch.go`, integrating the matrix engine, batch runner, and contact sheet exporters with live TTY progress reporting. <!-- sdd-owner: implementation -->
- [x] **REFACTOR**: Update user documentation in `docs/cli.md`, verify CLI help text, ensure clean exit codes, and run full test suite with `-race`. <!-- sdd-owner: implementation -->
