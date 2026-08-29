# ARIS CLI Reference

ARIS provides a comprehensive command-line interface for text-to-image synthesis, image editing/inpainting, terminal TUI, web server hosting, desktop window launching, messaging gateways, subagent management, and Knowledge Graph memory.

---

## Command Overview

```text
Usage:
  aris <command> [options]
  aris gen "<prompt>" [options]
  aris batch "<prompt>" [options]
  aris edit <image_path> "<prompt>" [options]
  aris gen "@<subagent> <prompt>"
  aris subagents [list|show|run]

Commands:
  gen, generate       Synthesize an image from natural language (or route via @name)
  batch               Batch generation, prompt matrix expansion & A/B benchmarking
  edit                Transform or inpaint reference images (img2img / inpaint)
  chat, tui           Launch interactive Cyberpunk TUI (split-screen chat & controls)
  serve, ui           Launch ARIS Web Interface & REST/SSE Server (headless VPS mode)
  gui, desktop        Launch ARIS Desktop GUI (local window or remote VPS client)
  gateway, gw         Launch remote messaging gateway (Telegram & Discord bots)
  subagents, sub      Inspect and run specialized visual subagents
  backends, backend   List and inspect available local & cloud image backends
  memory, mem         Manage 3-scope Knowledge Graph (list, add, search)
  history, hist       View past generations log
  version             Show version info
  help                Show help message
```

---

## 1. `aris gen` — Text-to-Image Generation

Synthesizes an image using the Art Director ReAct loop, Knowledge Graph memory recall, and the selected image backend.

### Syntax
```bash
aris gen "<prompt>" [options]
aris gen "@<subagent> <prompt>" [options]
```

### Options
| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `--ratio <val>` | `-r` | `1:1` | Aspect ratio: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `21:9` |
| `--backend <val>` | `-b` | `pollinations` | Backend: `pollinations`, `comfyui`, `falai`, `replicate`, `openai`, `huggingface` |
| `--model <val>` | `-m` | `flux` | Model name (e.g. `flux`, `flux-realism`, `dall-e-3`, `sd-3.5`) |
| `--seed <int>` | `-s` | `0` (random) | Seed for reproducible generation |
| `--negative <str>` | `-n` | `""` | Negative prompt keywords |
| `--critic` | — | `false` | Run VLM visual critique on output |
| `--auto-heal` | — | `false` | Automatically retry & refine prompt if critique score is below threshold |

### Examples
```bash
# Zero-config generation
aris gen "a cyberpunk cat in neo tokyo" --ratio 16:9

# Direct subagent routing
aris gen "@director cinematic shot of an ancient alien temple in a bioluminescent jungle"
aris gen "@anime young mech pilot standing in neo tokyo rain, makoto shinkai style"
aris gen "@photorealism 85mm portrait of a weary cybernetic detective, cinematic lighting"

# High-performance local GPU via ComfyUI
aris gen "hyperrealistic robotic tiger" --backend comfyui --model flux

# Generation with automated VLM critique & self-healing
aris gen "detailed futuristic cockpit with illuminated telemetry displays" --critic --auto-heal
```

---

## 2. `aris batch` — Batch Generation, Prompt Matrix & A/B Benchmarking

Executes high-throughput image generation batches with combinatorial prompt matrix expansion (`[opt1|opt2]`), deterministic seed sweeping (`--seed-sweep <start>-<end>`), multi-backend A/B benchmarking, concurrency-controlled worker pool, fail-soft resilience, and visual/tabular contact sheet exports (`index.html`, `summary.md`, `batch_meta.json`).

### Syntax
```bash
aris batch "<prompt>" [options]
```

### Options
| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `--count <int>` | `-c` | `1` | Number of image variants to generate |
| `--seed-sweep <str>` | `-s` | `""` | Sequential seed range in `<start>-<end>` format (e.g. `100-105`) |
| `--seed <int64>` | — | `0` | Base seed for deterministic count generation |
| `--matrix` | `-m` | `false` | Explicitly enable prompt matrix Cartesian expansion |
| `--benchmark` | `-b` | `false` | Benchmark prompt across all registered image backends |
| `--backends <list>` | — | `""` | Comma-separated list of image backends to execute |
| `--concurrency <int>`| `-j` | `2` | Number of concurrent worker goroutines |
| `--output-dir <path>`| `-o` | `./outputs/<batch_id>` | Custom output directory for batch bundle |
| `--max-matrix-jobs <int>` | — | `100` | Upper limit on matrix permutations before requiring `--force` |
| `--force` | — | `false` | Bypass matrix job limit safety check |
| `--dry-run` | — | `false` | Preview planned job combinations without rendering |
| `--eval` | — | `false` | Run VLM visual critic evaluation on output images |
| `--ratio <val>` | `-r` | `1:1` | Aspect ratio: `1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `21:9` |
| `--model <val>` | — | `flux` | Target model identifier |
| `--negative <str>` | `-n` | `""` | Negative prompt keywords |

### Examples
```bash
# Combinatorial prompt matrix with seed sweeping
aris batch "a [cyberpunk|steampunk] cat in [Tokyo|London]" --seed-sweep 1-4

# N-count generation with base seed
aris batch "neon sports car" --count 4 --seed 4200 --concurrency 4

# Multi-backend A/B benchmarking
aris batch "portrait of a space explorer" --benchmark --backends comfyui,falai,pollinations

# Preview job combinations with dry-run
aris batch "a [red|blue|gold] mecha robot in [space|desert]" --seed-sweep 10-12 --dry-run

# Matrix generation with VLM critic scoring and custom output folder
aris batch "alien flora [luminescent|crystal]" --eval -o ./outputs/alien_flora_batch
```

---

## 3. `aris edit` — Image-to-Image & Inpainting

Modifies existing images using visual reference conditioning, denoise strength calibration, and optional inpainting masks.

### Syntax
```bash
aris edit <image_path> "<prompt>" [options]
```

### Options
| Flag | Shorthand | Default | Description |
|---|---|---|---|
| `--strength <val>` | `-s` | `0.70` | Denoise strength `[0.0 - 1.0]` (0.3 = subtle restyle, 0.7 = heavy edit, 1.0 = repaint) |
| `--mask <path>` | — | `""` | Path to inpainting mask image (PNG/JPEG) |
| `--backend <val>` | `-b` | `falai` | Backend: `falai`, `comfyui`, `openai`, `pollinations` |
| `--model <val>` | `-m` | `flux` | Target model identifier |
| `--ratio <val>` | `-r` | `1:1` | Target aspect ratio override |
| `--critic` | — | `false` | Run VLM visual critique on edited output |
| `--auto-heal` | — | `false` | Automatically retry on low critique score |

### Examples
```bash
# Image-to-Image style overhaul
aris edit photo.png "cyberpunk neon style, raining street reflections" --strength 0.65 --backend falai

# Localized inpainting with a mask
aris edit character.png "remove sunglasses, realistic detailed blue eyes" --mask eye_mask.png --backend comfyui
```

---

## 4. `aris tui` / `aris chat` — Interactive Cyberpunk TUI

Launches a terminal user interface powered by Bubbletea and Lipgloss.

```bash
aris tui
```

### Keyboard Shortcuts
- `Enter`: Submit prompt / message.
- `Tab`: Cycle aspect ratio (`1:1` $\to$ `16:9` $\to$ `9:16` $\to$ `4:3` $\to$ `3:4` $\to$ `21:9`).
- `Ctrl+B`: Cycle through registered image backends.
- `Ctrl+O`: Open the latest rendered image in the operating system's default viewer.
- `Esc` / `Ctrl+C`: Exit TUI.

---

## 5. `aris serve` / `aris ui` — Headless Web Server (VPS Mode)

Launches the ARIS Web Interface and REST/SSE API server.

```bash
# Local development server
aris serve

# Production VPS server on all interfaces with token security
aris serve --host 0.0.0.0 --port 8080 --token "my-secret-token"
```

---

## 6. `aris gui` / `aris desktop` — Desktop Window & Remote Client

Launches the native OS desktop window connected either to a local in-process engine or to a remote ARIS VPS server.

```bash
# Local desktop window
aris gui

# Connect desktop client to a remote VPS
aris gui --remote https://vps.mydomain.com:8080 --token "my-secret-token"
```

---

## 7. `aris gateway` — Telegram & Discord Bots

Runs concurrent Telegram and Discord bot adapters with centralized job queue concurrency control.

```bash
export TELEGRAM_BOT_TOKEN="your-telegram-token"
export DISCORD_BOT_TOKEN="your-discord-token"
aris gateway
```

---

## 8. `aris subagents` — Visual Subagent Catalog

Inspects and tests built-in and user-defined visual subagents.

```bash
# List all registered subagents
aris subagents list

# Show detailed information and prompt recipe for a subagent
aris subagents show director

# Execute a direct subagent prompt
aris subagents run critic "evaluate anatomical accuracy and lighting coherence"
```

---

## 9. `aris memory` — Knowledge Graph Management

Manages persistent 3-scope memory (User, Style, Project) in SQLite FTS5.

```bash
# List memory facts by scope
aris memory list --scope style

# Search memory facts
aris memory search "cyberpunk lighting"

# Add a new fact manually
aris memory add --topic "style:cyberpunk" --concept "lighting" --fact "volumetric neon teal and magenta fog" --scope style
```

---

## 10. `aris history` — Past Generations Log

Inspects recent image generations, metadata, and local file paths.

```bash
aris history
```
