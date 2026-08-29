# Proposal: Batch Generation, Prompt Matrix & A/B Benchmarking

## Intent
Add concurrent batch generation, prompt matrix expansion, seed sweeps, and multi-backend A/B benchmarking to ARIS. This enables exploratory style testing, seed variance hunting, and comparative backend evaluation directly from the CLI.

## Problem Statement
Generating single images one by one makes exploratory style testing, seed variance hunting, and comparative backend benchmarking slow and tedious. Users currently have to manually run the CLI repeatedly, track seeds, and manually compare results across different providers (like Fal.ai vs ComfyUI).

## Proposed Solution
Build a concurrent Batch Engine with Prompt Matrix expansion, Seed Sweeps, Multi-Backend A/B benchmarking, and HTML/Markdown contact sheet exporters under `internal/core/services/` and `internal/adapters/ui/cli/`.

## Key Capabilities
1. **N-Count & Seed Sweep**: `--count <N>` (random seeds) & `--seed-sweep <start>-<end>` (sequential seeds) modes.
2. **Prompt Matrix Engine**: Combinatorial Cartesian expansion for bracketed syntax (e.g., `[cyberpunk|anime|photorealistic]`).
3. **Multi-Backend A/B Benchmark**: `--benchmark --backends pollinations,comfyui,falai` mode collecting duration, size, and optional VLM critic scores.
4. **Concurrency Control**: Bounded worker pool via `--concurrency <N>` (defaulting to a safe number like 2).
5. **Contact Sheet Exporter**: Auto-generates `index.html` (visual grid) and `summary.md` (table) inside `./outputs/batch_<timestamp>/`.

## Scope & Affected Areas
- **`internal/adapters/ui/cli/`**: New `batch` cobra command and flag definitions.
- **`internal/core/services/`**: 
  - `matrix.go`: pure Go Cartesian parser for prompt matrices.
  - `batch_runner.go`: worker pool, job orchestration, metric collection.
  - `contact_sheet.go`: HTML and Markdown generation logic.

## Non-goals
- A full web-based interactive dashboard (we rely on static HTML contact sheets for now).
- Distributing batch jobs across multiple physical machines (this is single-machine multi-threading).

## Risks
1. **OOM / VRAM Exhaustion**: Local backends like ComfyUI might crash if concurrency is too high.
2. **API Rate Limits**: Remote backends (Fal, Pollinations) might HTTP 429 if bursts are unmanaged.
3. **Matrix Explosion**: Cartesian products grow extremely fast. `[a|b]...` x 10 equals 1024 jobs.

## Rollback
- Revert the `batch` command registration from the CLI.
- Drop `internal/core/services/batch_runner.go` and its dependents.

## Success Criteria
- The CLI can run a prompt like `"a [cyberpunk|anime] cat"` with `--seed-sweep 1-2` and produce 4 images concurrently.
- An `index.html` file is generated showing the 4 variants clearly.
- Running `--benchmark --backends fal,pollinations` outputs a `summary.md` comparing execution times.

---

## Proposal Question Round
These questions help ensure the batch engine handles real-world constraints cleanly before we spec the exact CLI shape and concurrency model:

1. **Matrix Safety Limits**: Should the prompt matrix expansion have a hard upper limit (e.g., max 100 jobs) to prevent accidental Cartesian explosions that lock up the system or exhaust API credits? 
2. **Backend Concurrency Heterogeneity**: ComfyUI likely needs concurrency=1 to avoid VRAM OOM, while Fal.ai could comfortably handle concurrency=5. Should concurrency be defined *per backend*, or globally for the entire batch pool?
3. **Failure Handling**: If a remote backend rate-limits (HTTP 429) or fails mid-batch, should the batch engine halt completely, or log the error, mark the grid cell as "FAILED", and continue the remaining jobs?
4. **VLM Critic Scope**: The benchmark mentions "optional VLM critic scores". Does this require adding a completely new VLM adapter (like OpenAI/Anthropic vision), or do we defer the VLM scoring capability to a future PR and just stub the metric?
