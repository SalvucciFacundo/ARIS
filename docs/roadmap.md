# ARIS Product & Technical Roadmap

---

## 🏁 Milestone 1: Core Foundations & Multi-Platform v1.0.0 (DONE)

- [x] **Hexagonal Architecture (Ports & Adapters)**: Pure Go domain core decoupled from providers.
- [x] **Multi-Backend Registry**: Pollinations (default free), ComfyUI, Fal.ai, OpenAI DALL-E 3, Replicate, HuggingFace.
- [x] **Knowledge Graph Memory**: 3-scope memory (User, Style, Project) in SQLite with FTS5 and auto-learning loop.
- [x] **VLM Vision Critic & Self-Healing Loop**: Automated visual constraint evaluation and single-reroll healing.
- [x] **Specialized Visual Subagents (@name)**: `@director`, `@promptsmith`, `@photorealism`, `@anime`, `@concept-art`, `@cyberpunk`, `@pixelart`, `@critic`, `@inpainter`, `@restyler`.
- [x] **Cyberpunk Bubbletea TUI (`aris tui`)**: Split-screen chat, status stream, ANSI 24-bit image preview.
- [x] **Remote Messaging Gateways (`aris gateway`)**: Concurrent Telegram and Discord bots with `JobQueue` concurrency limiting.
- [x] **Img2Img & Inpainting Pipeline (`aris edit`)**: Reference image loader, denoise strength calibration, and inpainting mask validation.
- [x] **Desktop GUI & Remote Web Interface (`aris serve` / `aris gui`)**: Single-binary Templ + Templ Islands + HTMX + Tailwind UI with remote VPS mode.
- [x] **Enterprise Distribution**: Multi-distro Linux packages (`.deb`, `.rpm`, `.apk`, `.pkg.tar.zst`), macOS, Windows, one-line curl installer, CI/CD GitHub Actions.

---

## 🚀 Milestone 2: Batch Generation & Prompt Matrix (v1.1.0) — [DONE]

- [x] **Batch Generator (`aris batch`)**: Generate $N$ image variations concurrently with worker pool management.
- [x] **Seed Sweep Mode (`--seed-sweep 1-10`)**: Sweep across seeds with an identical prompt to explore the latent space.
- [x] **Prompt Matrix Engine (`--matrix`)**: Combine multiple prompt modifiers and styles into a combinatorial comparison grid.
- [x] **Multi-Backend A/B Benchmark**: Execute identical prompts across backends (e.g. ComfyUI vs Fal.ai vs Pollinations) and output side-by-side comparisons with timing and VLM quality scores.
- [x] **HTML / Markdown Grid Exporter**: Export batch generations as HTML contact sheets or Markdown grids.

---

## 🔍 Milestone 3: Upscaling & Face Restoration (v1.2.0) — [DONE]

- [x] **Super-Resolution Engine (`aris upscale`)**: 2x, 4x, 8x upscaling using Real-ESRGAN, SUPIR, and Flux Upscaler.
- [x] **Face Restoration Loop (`--restore-faces`)**: CodeFormer and GFPGAN face and eye fidelity corrections.
- [x] **Subagent `@upscaler`**: Dedicated prompt and resolution enhancer.

---

## 🎨 Milestone 4: LoRA & ControlNet Manager (v1.3.0) — [DONE]

- [x] **LoRA Loader (`aris lora`, `--lora`)**: Dynamic inline `<lora:name:scale>` prompt tag parsing and multi-LoRA stacking for ComfyUI and Fal.ai Flux.
- [x] **ControlNet Structural Guidance (`aris controlnet`, `--controlnet`)**: Zero-dependency pure-Go Canny edge detection preprocessor, OpenPose, LineArt, Depth Map conditioning, and dynamic graph/payload assembly.

---

## 📦 Milestone 5: ComfyUI Workflow JSON Export (v1.4.0) — [DONE]

- [x] **Reproducible Workflow Serialization**: Embed full ComfyUI JSON node graphs (`prompt` & `workflow`) inside generated PNG metadata (`tEXt`/`iTXt` chunks).
- [x] **Drag-and-Drop Workflow Import**: Drag any ARIS-generated image directly into ComfyUI Web GUI to reload the exact visual node graph.
- [x] **CLI Workflow Tooling (`aris workflow`)**: Subcommands `aris workflow inspect` and `aris workflow export` for extracting generation parameters and raw JSON graphs.
