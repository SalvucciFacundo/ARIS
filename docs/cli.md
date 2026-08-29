# ARIS CLI Reference

ARIS provides a comprehensive command-line interface for text-to-image synthesis, image editing/inpainting, terminal TUI, web server hosting, desktop window launching, messaging gateways, subagent management, and Knowledge Graph memory.

---

## Command Overview

```text
Usage:
  aris <command> [options]
  aris gen "<prompt>" [options]
  aris edit <image_path> "<prompt>" [options]
  aris gen "@<subagent> <prompt>"
  aris subagents [list|show|run]

Commands:
  gen, generate       Synthesize an image from natural language (or route via @name)
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

## 2. `aris edit` — Image-to-Image & Inpainting

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

## 3. `aris tui` / `aris chat` — Interactive Cyberpunk TUI

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

## 4. `aris serve` / `aris ui` — Headless Web Server (VPS Mode)

Launches the ARIS Web Interface and REST/SSE API server.

```bash
# Local development server
aris serve

# Production VPS server on all interfaces with token security
aris serve --host 0.0.0.0 --port 8080 --token "my-secret-token"
```

---

## 5. `aris gui` / `aris desktop` — Desktop Window & Remote Client

Launches the native OS desktop window connected either to a local in-process engine or to a remote ARIS VPS server.

```bash
# Local desktop window
aris gui

# Connect desktop client to a remote VPS
aris gui --remote https://vps.mydomain.com:8080 --token "my-secret-token"
```

---

## 6. `aris gateway` — Telegram & Discord Bots

Runs concurrent Telegram and Discord bot adapters with centralized job queue concurrency control.

```bash
export TELEGRAM_BOT_TOKEN="your-telegram-token"
export DISCORD_BOT_TOKEN="your-discord-token"
aris gateway
```

---

## 7. `aris subagents` — Visual Subagent Catalog

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

## 8. `aris memory` — Knowledge Graph Management

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

## 9. `aris history` — Past Generations Log

Inspects recent image generations, metadata, and local file paths.

```bash
aris history
```
