# Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1250-1600 |
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

# Tasks: ARIS ComfyUI Workflow JSON Export & Metadata Interoperability

This document enumerates the atomic, dependency-ordered tasks required to implement ComfyUI workflow JSON export, pure-Go PNG chunk manipulation, multi-backend metadata injection, and CLI inspection/export subcommands under strict TDD (RED -> GREEN -> REFACTOR).

---

## PR 1: Pure-Go PNG Chunk Manipulation Pipeline
**Objective**: Implement pure-Go streaming utilities to read, write, inject, and extract `tEXt` metadata chunks in PNG byte streams with strict CRC-32 validation and zero image-raster memory overhead.

### Tasks
- [x] **RED: Write failing unit tests for PNG chunk injection and extraction** <!-- sdd-owner: implementation -->
  - Create `pkg/imgutil/png_chunks_test.go`.
  - Implement tests for valid 1x1 PNG round-trip metadata injection/extraction (`TestInjectAndExtractMetadata_RoundTrip`), invalid PNG signatures (`TestExtractMetadata_InvalidSignature`), corrupted chunk CRCs (`TestExtractMetadata_CorruptedCRC`), and missing metadata handling (`TestExtractMetadata_NoMetadata`).
  - Run `go test ./pkg/imgutil/...` and verify test failure.

- [x] **GREEN: Implement core PNG chunk reader, writer, and `tEXt` manipulator** <!-- sdd-owner: implementation -->
  - Create `pkg/imgutil/png_chunks.go`.
  - Implement signature validation (`\x89PNG\r\n\x1a\n`), chunk streaming header iteration (Length, Type, Data, CRC-32), `IEND` chunk interception, `tEXt` chunk formatting (`keyword\x00text`), and IEEE CRC-32 calculation using `hash/crc32`.
  - Implement `InjectMetadata`, `ExtractMetadata`, and companion stream readers/writers.
  - Run `go test ./pkg/imgutil/...` and verify all tests pass.

- [x] **TRIANGULATE & REFACTOR: Edge cases, performance optimization, and error wrapping** <!-- sdd-owner: implementation -->
  - Add edge-case test cases for multiple `tEXt` chunks, very large text payloads, and malformed chunk lengths.
  - Refactor stream copy operations to minimize heap allocations and ensure proper `io.Reader`/`io.Writer` cleanup.
  - Verify all tests pass cleanly under `go test -v -race ./pkg/imgutil/...`.

---

## PR 2: ComfyUI Backend Workflow Injection
**Objective**: Integrate `pkg/imgutil` into the ComfyUI image adapter to automatically capture and embed execution prompts (`prompt`) and UI node graphs (`workflow`) as valid PNG `tEXt` chunks.

### Tasks
- [x] **RED: Write failing integration/unit tests for ComfyUI workflow embedding** <!-- sdd-owner: implementation -->
  - Create or update `internal/adapters/image/comfyui_test.go` (or `backends_test.go`).
  - Mock the ComfyUI generation response to return a valid synthetic PNG image along with JSON prompt and workflow graphs.
  - Assert that images processed through the ComfyUI adapter successfully retain embedded `prompt` and `workflow` `tEXt` chunks.
  - Run `go test ./internal/adapters/image/...` and verify test failure.

- [x] **GREEN: Implement ComfyUI adapter metadata embedding** <!-- sdd-owner: implementation -->
  - Update `internal/adapters/image/comfyui.go` to intercept generated PNG output streams.
  - Marshal the active ComfyUI `prompt` map and `workflow` graph map into minified JSON strings.
  - Invoke `imgutil.InjectMetadata` with keywords `"prompt"` and `"workflow"` prior to writing the file to disk.
  - Run `go test ./internal/adapters/image/...` and verify tests pass.

- [x] **REFACTOR: Clean error propagation and resource management** <!-- sdd-owner: implementation -->
  - Ensure temporary byte buffers and readers are efficiently pooled or cleaned up.
  - Add structured logging for metadata injection events.
  - Run full test suite for adapters to ensure no regressions.

---

## PR 3: Universal Metadata Injection across Backends
**Objective**: Extend metadata embedding across all generation backends (Fal.ai, Pollinations, OpenAI, etc.) under the standardized `parameters` `tEXt` chunk.

### Tasks
- [x] **RED: Write failing tests for multi-backend parameter embedding** <!-- sdd-owner: implementation -->
  - Create or update `internal/core/services/agent_test.go` or backend-specific test files to test parameter serialization and embedding.
  - Verify that images generated via Fal.ai or Pollinations adapters include a valid `parameters` `tEXt` chunk containing prompt, model, seed, steps, and CFG scale.
  - Run tests and verify failure.

- [x] **GREEN: Implement central parameter metadata injection** <!-- sdd-owner: implementation -->
  - Update `internal/core/services/agent.go` and image saving pipelines to construct a unified generation parameters JSON structure.
  - Invoke `imgutil.InjectMetadata` with the `"parameters"` keyword for non-ComfyUI backends.
  - Run tests and verify success.

- [x] **REFACTOR: Standardize metadata keys and schema validation** <!-- sdd-owner: implementation -->
  - Ensure JSON schemas for parameters match Civitai/A1111 conventions.
  - Run `go test -race ./internal/core/services/...`.

---

## PR 4: CLI Subcommands, E2E Integration Tests & Docs
**Objective**: Deliver the `aris workflow inspect` and `aris workflow export` CLI subcommands, end-to-end integration tests, and user documentation.

### Tasks
- [x] **RED: Write failing CLI and E2E workflow integration tests** <!-- sdd-owner: implementation -->
  - Create `test/integration/workflow_e2e_test.go` and CLI command unit tests in `internal/adapters/ui/cli/workflow_test.go`.
  - Test `aris workflow inspect` (human-readable table vs `--json` output), `aris workflow export` (default basename, `-o` custom path, `-o -` stdout, overwrite protection without `--force`, and forced overwrite with `--force`).
  - Run tests and verify failure.

- [x] **GREEN: Implement CLI `workflow` subcommands (`inspect`, `export`)** <!-- sdd-owner: implementation -->
  - Create `internal/adapters/ui/cli/workflow.go`.
  - Implement Cobra subcommands `inspect` and `export` with appropriate flags (`--json`, `-o`, `--force`/`-f`).
  - Wire up error handling for non-PNG files (`ErrInvalidPNGSignature`), missing metadata, and existing file conflicts.
  - Register `workflow` command tree in the root CLI router.
  - Run tests and verify all pass.

- [x] **TRIANGULATE & REFACTOR: E2E verification, documentation, and final polish** <!-- sdd-owner: implementation -->
  - Create `docs/cli.md` detailing usage examples for `aris workflow inspect` and `aris workflow export`.
  - Run full integration test suite: `go test -v -race ./test/integration/... ./internal/adapters/ui/cli/...`.
  - Perform static analysis checks (`go vet`, `golangci-lint` if available).
