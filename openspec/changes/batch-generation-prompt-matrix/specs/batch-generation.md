# Specification: Batch Generation, Prompt Matrix & A/B Benchmarking

## Purpose

Define the functional and technical requirements for the ARIS Batch Generation subsystem. This subsystem provides high-throughput image generation workflows, combinatorial prompt matrix expansion, deterministic seed sweeping, multi-backend A/B benchmarking, concurrency-controlled worker execution, fail-soft resilience, and rich visual/tabular contact sheet export (`index.html`, `summary.md`, and `batch_meta.json`).

---

## Requirements

### Requirement: REQ-BATCH-1 - N-Count Generation Mode

The system MUST support generating a batch of $N$ image variants from a single base prompt using the `--count <N>` flag.

#### Scenario: Default N-Count Batch with Random Seeds
- GIVEN a user specifies an image prompt `"a neon cyberpunk alleyway in the rain"`
- AND passes the flag `--count 4` without an explicit seed
- WHEN the batch runner executes
- THEN the system MUST schedule and execute exactly 4 distinct image generation jobs
- AND each job MUST be assigned a uniquely generated random 64-bit non-negative integer seed
- AND all 4 resulting images MUST be saved to the batch output directory with unique sequential filenames.

#### Scenario: N-Count Batch with Base Seed
- GIVEN a user specifies a prompt and `--count 3`
- AND provides a base seed `--seed 4200`
- WHEN the batch runner schedules jobs
- THEN job 1 MUST use seed `4200`, job 2 MUST use seed `4201`, and job 3 MUST use seed `4202`
- AND all jobs MUST execute with deterministic seed assignment.

#### Scenario: Boundary and Invalid Count Validation
- GIVEN a user passes `--count 0` or `--count -5`
- WHEN the CLI validates the command flags
- THEN the system MUST reject the command with a validation error stating that `--count` MUST be an integer greater than or equal to 1
- AND the batch runner MUST NOT initialize worker pools or invoke image backends.

#### Scenario: Single Count Default
- GIVEN a user runs `aris batch "a serene mountain lake"` without `--count` or `--seed-sweep`
- WHEN the batch runner initializes
- THEN `--count` MUST default to `1`.

---

### Requirement: REQ-BATCH-2 - Seed Sweep Mode

The system MUST support deterministic sequential seed exploration via the `--seed-sweep <start>-<end>` flag.

#### Scenario: Inclusive Sequential Seed Range Parsing
- GIVEN a user specifies `--seed-sweep 100-105`
- WHEN the batch runner parses the sweep range
- THEN the system MUST generate a seed sequence containing `[100, 101, 102, 103, 104, 105]` (6 seeds total)
- AND schedule exactly 6 jobs using the specified prompt and respective seeds in ascending order.

#### Scenario: Single Seed Range
- GIVEN a user specifies `--seed-sweep 500-500`
- WHEN the batch runner parses the sweep range
- THEN the system MUST treat the range as valid containing exactly 1 seed (`500`)
- AND schedule 1 generation job.

#### Scenario: Inverted and Malformed Range Validation
- GIVEN a user passes `--seed-sweep 200-100` (inverted) or `--seed-sweep foo-bar` (non-numeric) or `--seed-sweep 100` (missing delimiter)
- WHEN the CLI validates the input
- THEN the system MUST reject the command with an error message detailing the invalid format or condition (`start` seed MUST be $\le$ `end` seed, and both MUST be valid non-negative integers)
- AND exit with a non-zero status code.

#### Scenario: Precedence and Mutual Exclusion
- GIVEN a user provides both `--count 5` and `--seed-sweep 10-15`
- WHEN the CLI flag validator runs
- THEN the system MUST reject the combination with a conflict error stating `--count` and `--seed-sweep` are mutually exclusive
- AND prompt the user to specify either a random count or an explicit seed sweep range.

---

### Requirement: REQ-BATCH-3 - Combinatorial Prompt Matrix Engine

The system MUST provide a pure Cartesian product parser that expands bracketed alternate tokens `[option1|option2|...]` into all combinatorial prompt permutations.

#### Scenario: Single Group Matrix Expansion
- GIVEN a prompt template `"a [cyberpunk|anime|oil painting] portrait of an astronaut"`
- WHEN the prompt matrix engine parses the template
- THEN it MUST generate exactly 3 distinct prompt variants:
  1. `"a cyberpunk portrait of an astronaut"`
  2. `"a anime portrait of an astronaut"`
  3. `"a oil painting portrait of an astronaut"`
  - AND maintain the original spacing and non-matrix prompt tokens.

#### Scenario: Multi-Group Cartesian Expansion
- GIVEN a prompt template `"a [cyberpunk|steampunk] [cat|fox] in [Tokyo|London]"`
- WHEN the prompt matrix engine parses the template
- THEN it MUST compute the full Cartesian product ($2 \times 2 \times 2 = 8$ variants)
- AND emit the exact permutations in deterministic order matching the order of option declarations.

#### Scenario: Escaped Brackets and Literal Preservation
- GIVEN a prompt containing escaped brackets `r"a cat wearing \[cyberpunk\] goggles in [neon|dark] alley"`
- WHEN the prompt matrix engine parses the template
- THEN the escaped brackets `\[cyberpunk\]` MUST be preserved as literal text `"a cat wearing [cyberpunk] goggles"`
- AND only `[neon|dark]` MUST be expanded into 2 prompt variants.

#### Scenario: Matrix Safety Upper Bound Enforcement
- GIVEN a prompt template with multiple groups yielding $128$ variants
- AND `--max-matrix-jobs` is set to the default safety limit of `100`
- AND `--force` is NOT provided
- WHEN the batch runner validates the planned job matrix
- THEN the system MUST abort execution BEFORE scheduling any jobs
- AND emit an error message indicating that the planned matrix size (128) exceeds the maximum allowed limit (100), advising the user to reduce variants or supply `--force` / `--max-matrix-jobs 150`.

#### Scenario: Matrix Multiplied by Seed Sweep
- GIVEN a prompt template with 3 matrix variants
- AND a seed sweep `--seed-sweep 1-4` (4 seeds)
- WHEN the batch runner compiles the job execution plan
- THEN the total number of scheduled jobs MUST be $3 \times 4 = 12$ jobs
- AND each prompt variant MUST be paired with all 4 seeds.

---

### Requirement: REQ-BATCH-4 - Multi-Backend A/B Benchmark Execution

The system MUST support executing batch jobs across multiple image generation backends simultaneously via `--benchmark --backends <backend1,backend2,...>`.

#### Scenario: Multi-Backend Parallel Dispatch
- GIVEN a user specifies `--benchmark --backends pollinations,comfyui,falai`
- AND a prompt generating 2 variants
- WHEN the batch runner compiles the execution plan
- THEN the system MUST duplicate each prompt-seed pair across all 3 registered backends ($2 \times 3 = 6$ total jobs)
- AND execute each job using its targeted backend implementation.

#### Scenario: Backend Registration Validation
- GIVEN a user requests `--backends comfyui,unknown_backend`
- WHEN the batch runner resolves the backend names against the `BackendRegistry`
- THEN the system MUST detect that `unknown_backend` is not registered
- AND abort the batch execution immediately, listing the valid available backends.

#### Scenario: Performance and Telemetry Capture
- GIVEN a running benchmark job
- WHEN the image generation completes (or fails)
- THEN the system MUST record:
  - `Backend`: Name of the executing backend
  - `DurationMS`: Total execution time in milliseconds
  - `ImageSizeBytes`: Output file size in bytes (if successful)
  - `Resolution`: Width $\times$ Height dimensions
  - `Status`: `"SUCCESS"` or `"FAILED"`
  - `ErrorMessage`: Error string if status is `"FAILED"`
  - `Seed`: Exact seed utilized
  - `Prompt`: Exact expanded prompt text.

#### Scenario: Optional Vision Critic Evaluation
- GIVEN a benchmark run with `--eval` enabled
- AND a registered `VisionCritic` service is available
- WHEN an image job succeeds
- THEN the batch runner MUST pass the rendered image and prompt spec to `VisionCritic.Evaluate`
- AND record the numeric score ($0.0 - 10.0$) and critique text in the job metadata
- AND IF the critic evaluation fails or is disabled, the batch runner MUST record score as `null` without failing the overall image generation job.

---

### Requirement: REQ-BATCH-5 - Concurrency Worker Pool, Throttling & Fail-Soft Error Resilience

The system MUST manage execution through a bounded goroutine worker pool with backend-aware concurrency throttling and fail-soft error recovery.

#### Scenario: Bounded Global Concurrency
- GIVEN a batch with 20 jobs and flag `--concurrency 4`
- WHEN the worker pool executes
- THEN at most 4 image generation goroutines MUST be executing concurrently at any given instant.

#### Scenario: Backend-Specific Rate and VRAM Throttling
- GIVEN a benchmark run combining local backend `comfyui` and cloud backend `falai`
- AND global `--concurrency 4` is set
- WHEN the worker pool dispatches jobs
- THEN the system MUST enforce a local backend concurrency limit (default $1$ for `comfyui` to prevent GPU/VRAM OOM)
- WHILE allowing remote HTTP backends (`falai`, `pollinations`) to utilize remaining worker pool capacity up to the global limit.

#### Scenario: Fail-Soft Error Handling on Mid-Batch Failures
- GIVEN a batch of 10 jobs where job 3 receives an HTTP 429 Rate Limit and job 7 encounters an upstream timeout
- WHEN the worker pool processes the jobs
- THEN the worker pool MUST NOT panic or abort the entire batch
- AND the runner MUST mark jobs 3 and 7 as `"FAILED"` with their respective error details
- AND continue processing the remaining 8 jobs to completion.

#### Scenario: Graceful Cancellation on Interruption Signal
- GIVEN an active batch running 50 jobs
- WHEN the user sends an interrupt signal (`SIGINT` / `Ctrl+C`)
- THEN the batch runner MUST cancel the root context
- AND stop dispatching new jobs from the queue
- AND await completion of currently in-flight network/local generation requests up to a grace period (default 5 seconds)
- AND finalize and export partial contact sheet artifacts (`index.html`, `summary.md`, `batch_meta.json`) for all completed jobs before exiting.

---

### Requirement: REQ-BATCH-6 - Artifacts & Contact Sheet Exporter

The system MUST automatically export a self-contained output bundle for every batch run inside a dedicated timestamped directory.

#### Scenario: Batch Output Directory Structure
- GIVEN a completed batch run
- WHEN the exporter writes the results
- THEN it MUST create an output directory at `./outputs/batch_<YYYYMMDD_HHMMSS>_<short_id>/` (or custom `--output-dir`) containing:
  - `images/`: Directory containing all rendered PNG/JPEG files named `job_<seq>_<backend>_seed<seed>.<ext>`
  - `index.html`: Self-contained HTML5 visual contact sheet
  - `summary.md`: Markdown summary table
  - `batch_meta.json`: Complete JSON manifest.

#### Scenario: Responsive HTML Visual Contact Sheet (`index.html`)
- GIVEN a batch with both successful and failed jobs
- WHEN `index.html` is generated
- THEN it MUST be a standalone HTML5 file with embedded responsive CSS and dark theme
- AND display a summary header (total jobs, duration, success count, failure count, concurrency, timestamp)
- AND render a responsive CSS grid of cards for each job containing:
  - Image thumbnail with link to full-resolution image
  - Prompt text with copy-to-clipboard button
  - Metadata badges: Backend, Seed, Aspect Ratio, Generation Duration (e.g. `1.84s`), and Critic Score (if evaluated)
  - Visual error callout box for any failed job displaying the error message.

#### Scenario: Markdown Summary Report (`summary.md`)
- GIVEN a completed benchmark or batch run
- WHEN `summary.md` is generated
- THEN it MUST contain a Markdown table with columns: `Job`, `Backend`, `Seed`, `Status`, `Duration (ms)`, `Size (KB)`, `Score`, `Prompt`
- AND include an aggregate summary section detailing Average Duration per Backend, Success Rate, and Total Batch Duration.

#### Scenario: Machine-Readable JSON Manifest (`batch_meta.json`)
- GIVEN a batch run
- WHEN `batch_meta.json` is exported
- THEN it MUST conform to a structured JSON schema containing:
  - `batch_id`: Unique string ID
  - `created_at`: ISO8601 timestamp
  - `total_jobs`: Integer count
  - `success_count`: Integer count
  - `failed_count`: Integer count
  - `total_duration_ms`: Integer milliseconds
  - `config`: Object containing flags and options used
  - `jobs`: Array of objects detailing each job's spec, result path, backend, duration, status, and error message.

---

### Requirement: REQ-BATCH-7 - CLI `aris batch` Flag Syntax, Validation & Progress Reporting

The system MUST provide a comprehensive Cobra CLI command `aris batch [prompt] [flags]` with full input validation and real-time terminal feedback.

#### Scenario: Command Syntax and Help Description
- GIVEN a user runs `aris batch --help`
- WHEN Cobra renders the command documentation
- THEN the system MUST display usage `aris batch "<prompt>" [flags]`
- AND list all supported flags:
  - `-c, --count <int>`: Number of image variants to generate (default 1)
  - `-s, --seed-sweep <string>`: Seed sweep range in `<start>-<end>` format
  - `--seed <int64>`: Base seed for deterministic count generation
  - `-m, --matrix`: Explicitly enable prompt matrix Cartesian expansion
  - `-b, --benchmark`: Enable multi-backend benchmarking mode
  - `--backends <string>`: Comma-separated list of backends to evaluate (default active backend)
  - `-j, --concurrency <int>`: Number of concurrent generation workers (default 2)
  - `-o, --output-dir <string>`: Custom directory path for batch outputs
  - `--max-matrix-jobs <int>`: Maximum allowed Cartesian matrix jobs before requiring `--force` (default 100)
  - `--force`: Bypass matrix job limit safety check
  - `--dry-run`: Preview planned jobs without generating images
  - `--eval`: Enable VLM visual critic evaluation on output images.

#### Scenario: Dry-Run Mode Execution
- GIVEN a prompt `"a [retro|futuristic] car"` with `--seed-sweep 1-3` and `--backends comfyui,falai`
- AND the `--dry-run` flag is passed
- WHEN the CLI executes
- THEN the system MUST calculate the planned execution plan ($2 \text{ prompts} \times 3 \text{ seeds} \times 2 \text{ backends} = 12 \text{ jobs}$)
- AND print a formatted terminal preview table listing all 12 planned job combinations
- AND exit with status 0 WITHOUT creating output directories or invoking image backends.

#### Scenario: Interactive Real-Time Terminal Progress Reporting
- GIVEN a batch run executing in a TTY environment
- WHEN jobs are being processed by the worker pool
- THEN the CLI MUST display a real-time progress indicator showing:
  - Progress counter: `[Completed/Total]` (e.g. `[7/12] 58%`)
  - Active worker status line indicating currently generating backend and job ID
  - Real-time elapsed time counter
  - Completed job outcome indicators (e.g. green `✓` for success, red `✗` for failure)
- AND on completion, print the path to `index.html` and `summary.md`.

#### Scenario: Empty Prompt Validation Error
- GIVEN a user invokes `aris batch ""` without a prompt argument
- WHEN the command validates arguments
- THEN the system MUST return a non-zero exit code with an error message: `"prompt cannot be empty"`.
