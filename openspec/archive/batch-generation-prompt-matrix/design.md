# Design: Batch Generation, Prompt Matrix & A/B Benchmarking

## Hexagonal Architecture Placement & Component Decomposition

The implementation strictly follows Hexagonal Architecture principles, isolating core business logic (matrix parsing, concurrency rules) from adapters (CLI output, HTML generation).

### 1. Core Services (`internal/core/services/`)

*   **`matrix.go` (Combinatorial Expansion Engine)**
    *   **Responsibility**: Parse prompts containing bracketed options `[opt1|opt2]`, escape characters, and generate the full Cartesian product.
    *   **Key Types**:
        *   `PromptNode`: AST node representing either literal text or an option group.
        *   `MatrixEngine`: Service with a method `Expand(prompt string) ([]string, error)` which calculates the permutations.

*   **`batch_runner.go` (Execution Orchestrator)**
    *   **Responsibility**: Orchestrates the worker pool, applies concurrency and backend throttling rules, manages context cancellation (SIGINT), and collects job telemetry.
    *   **Key Types**:
        *   `BatchPlan`: Output of parsing count/sweep/backends into a distinct list of execution targets.
        *   `BatchJob`: Value object representing one planned execution (seed, prompt, backend).
        *   `BatchResult`: Result object (duration, path, status, error).
        *   `Runner`: The main coordinator exposing `Execute(ctx, plan) (*BatchSummary, error)`.

*   **`contact_sheet.go` (Artifact Generators)**
    *   **Responsibility**: Takes the finalized `BatchSummary` and renders the output directory artifacts.
    *   **Key Types**:
        *   `ArtifactExporter`: Interfaces mapping to `ExportHTML`, `ExportMarkdown`, `ExportJSON`.

### 2. UI Adapters (`internal/adapters/ui/cli/`)

*   **`batch.go` (Cobra Command)**
    *   **Responsibility**: Handle `aris batch` CLI inputs, flag validation, construct execution plans, execute `--dry-run`, and drive terminal progress bars.

---

## Concurrency Model & Error Resilience

1.  **Channel-based Worker Pool**:
    *   The `BatchRunner` creates a buffered `jobs <-chan BatchJob` channel and a `results chan<- BatchResult` channel.
    *   It spawns N goroutines based on `--concurrency <N>`.
2.  **Backend-Aware Throttling**:
    *   Inside the worker, before delegating to the backend adapter, it checks a backend-specific semaphore (e.g., local backends like `comfyui` are wrapped with a `sync.Mutex` or a bounded semaphore channel of size 1) to prevent VRAM OOM while allowing remote backends (e.g., `falai`) to saturate the global concurrency pool.
3.  **Fail-Soft Resilience**:
    *   Backend invocations are wrapped in a safe error boundary. If a backend returns an error (like HTTP 429), the worker catches it, creates a `BatchResult` with `Status: "FAILED"` and `Error: err.Error()`, sends it to the results channel, and immediately pulls the next job. The main pool never panics.
4.  **Graceful Shutdown**:
    *   The command listens to `SIGINT`. Upon cancellation, the root context is cancelled. Workers abort long-polling network requests, the `jobs` channel is closed, and the results aggregator flushes all completed (and partially failed) jobs into the contact sheet exporter before exiting.

---

## Contact Sheet & Manifest Formats

*   **`batch_meta.json`**:
    *   Strict schema defining `batch_id`, `config`, and an array of `jobs` with complete input variables and outputs. Suitable for programmatic parsing.
*   **`summary.md`**:
    *   A Markdown table: `| Job | Backend | Seed | Status | Duration | Size |`
*   **`index.html`**:
    *   A standalone HTML5 file with embedded responsive CSS and Dark Mode.
    *   Grid layout using CSS Flexbox/Grid for image thumbnails.
    *   Overlaid metadata tags (Backend, Seed, Time, Critic Score).
    *   Failed jobs render as empty cards with red error text block.

---

## Flow & Sequence Diagrams

### Batch Runner Concurrency Flow

```text
CLI (batch.go)
 │
 ├── 1. Parse & Validate Flags
 ├── 2. matrix.Expand(prompt) -> []prompts
 ├── 3. Build BatchPlan (Cartesian: Prompts × Seeds × Backends)
 │
 ├── 4. batch_runner.Execute(ctx, BatchPlan)
 │    │
 │    ├── Setup jobs channel (Buffer: total jobs)
 │    ├── Setup results channel (Buffer: total jobs)
 │    │
 │    ├── Spawn N Workers (--concurrency limit)
 │    │    │
 │    │    ├── Worker Loop
 │    │    │    ├── Job <- jobs
 │    │    │    ├── Acquire Backend Semaphore (e.g., VRAM lock)
 │    │    │    ├── Execute Generate()
 │    │    │    ├── Release Backend Semaphore
 │    │    │    └── Send Result -> results
 │    │
 │    ├── Aggregator Loop (Select on results channel)
 │    │    ├── Accumulate BatchSummary
 │    │    └── Update CLI Progress Bar
 │    │
 │    └── Close channels & WaitGroup
 │
 └── 5. contact_sheet.Export(BatchSummary) -> Writes to Disk
```

---

## Testing Strategy (Strict TDD)

Since this project has **Strict TDD Mode: enabled**, all tests must be written before implementation code.

1.  **Matrix Expansion Suite (`matrix_test.go`)**:
    *   Pure function tests. Input `"a [red|blue] car"` -> Output `["a red car", "a blue car"]`.
    *   Assert escaping mechanisms (`\[not an option\]`).
    *   Verify Cartesian multiplier limits.
2.  **Concurrency & Runner Suite (`batch_runner_test.go`)**:
    *   Must be executed with the `go test -race` flag to guarantee channel safety.
    *   Use Mock Backends that simulate delay and simulated failures (return errors half the time).
    *   Assert that `BatchSummary` successfully accounts for N successes and M failures.
    *   Test context cancellation ensuring early exit without hanging goroutines.
3.  **CLI Validation Suite (`batch_test.go`)**:
    *   Test `--dry-run` output formatting using buffer captures.
    *   Test mutually exclusive flags (`--count` vs `--seed-sweep`).
