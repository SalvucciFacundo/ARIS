# Exploration: Batch Generation, Prompt Matrix & A/B Benchmarking (ARIS)

## 1. Overview
This exploration investigates the requirements and technical strategy for implementing batch generation, combinatorial prompt matrix expansion, seed sweeps, and multi-backend A/B benchmarking in ARIS.

## 2. Key Capabilities Explored

### 2.1 Batch Generation & Seed Sweeps
- **N-Count Mode**: Executing $N$ runs of a single prompt with distinct random seeds (`--count 4`).
- **Seed Sweep Mode**: Explicit sequential range of seeds (`--seed-sweep 100-104`).
- **Concurrency Management**: Utilizing a bounded worker pool (default concurrency = 2) to prevent GPU VRAM exhaustion or API rate limit triggers.

### 2.2 Prompt Matrix Engine (Combinatorial Cartesian Expansion)
- Parsing bracketed variant syntax within prompt strings:
  `"a [cyberpunk|anime|photorealistic] cat in [neo-tokyo|space station] with [neon lighting|cinematic haze]"`
- Computing the Cartesian product to produce all combinatorial variations (e.g. $3 \times 2 \times 2 = 12$ distinct generation tasks).

### 2.3 Multi-Backend A/B Benchmark
- Executing identical prompt blueprints across multiple target backends:
  `aris batch "a majestic dragon" --benchmark --backends pollinations,comfyui,falai`
- Aggregating metrics per run: duration (ms), image byte size, resolution, and optional VLM Vision Critic quality score.

### 2.4 Output Packaging & Contact Sheets
- Dedicated batch output folder: `./outputs/batch_<timestamp>/`.
- JSON metadata summary (`batch_meta.json`).
- HTML Contact Sheet (`index.html`) featuring side-by-side thumbnail comparisons, timing badges, VLM scores, and parameter inspect modals.
- Markdown summary table (`summary.md`).

## 3. Architecture & Integration Points
- **Service Layer**: Introduce `BatchRunner` or `BatchService` in `internal/core/services/` that leverages `AgentService.Generate` across a worker pool.
- **Matrix Parser**: Implement a pure Go combinatorial parser in `internal/core/services/matrix.go`.
- **CLI Adapter**: Add Cobra command `aris batch` in `internal/adapters/ui/cli/batch.go`.

## 4. Risks & Mitigations
- **API Rate Limiting**: Centralized concurrency throttle and retry backoff per backend.
- **Memory Pressure**: Writing image assets directly to disk during streaming rather than holding large raw byte buffers in RAM.
