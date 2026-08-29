# ARIS — Autonomous Reasoner for Image System

<p align="center">
  <img src="https://img.shields.io/badge/Status-Architecture%20Design-blue" alt="Status">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go" alt="Go"></a>
</p>

**ARIS** is an autonomous AI visual generation agent written in pure **Go**. You describe what you want in natural language — *"a cyberpunk samurai cat in neo-tokyo, raining, cinematic lighting"* — and ARIS acts as an **Art Director & Prompt Architect**: recalls user aesthetic preferences, engineers positive and negative prompts, selects optimal parameters, synthesizes the image across local or cloud backends, and optionally evaluates the result with a Vision Critic.

---

## ✨ Key Features

- 🧠 **Autonomous Prompt Architect**: Understands natural language requests and converts them into technical prompts with style tokens, composition, lighting, aspect ratios, seeds, and CFG parameters.
- 💾 **GAIA Knowledge Graph Memory**: Persistent 3-scope memory (User, Style, Project) in SQLite with FTS5 full-text search. Recalls your favorite styles, negative triggers, and character consistency across sessions.
- 🔌 **Interchangeable Image Backends**:
  - **Pollinations.ai API** (Default out-of-the-box, zero configuration, free & fast).
  - **ComfyUI Local** (High-performance local GPU generation via WebSocket/REST).
  - **Cloud APIs** (Fal.ai, Replicate, OpenAI DALL-E 3, HuggingFace).
- 👁️ **VLM Self-Correction Loop**: Optional vision language model critique (Ollama Qwen2.5-VL / Cloud Vision) that inspects generated images against user constraints before final delivery.
- 🔄 **Conversational Iteration & Img2Img**: Refine images iteratively (*"make it darker"*, *"remove the helmet"*) or supply reference images.
- 🖥️ **Multiple Interfaces**: Headless CLI, Cyberpunk Bubbletea TUI (with Kitty/Sixel/iTerm2 image previews), Wails v2 Desktop App, and Telegram/Discord gateways.
- 📦 **Single Binary**: Pure Go, zero external system runtime dependencies.

---

## 🏛️ Architecture

ARIS follows strict **Hexagonal Architecture (Ports & Adapters)** in Go:

```
[User Input] ──► [Reason & Recall (Knowledge Graph)]
                       │
                       ▼
                 [Prompt Architect]
                       │
                       ▼
                 [Image Backend Dispatcher (Pollinations / ComfyUI / Fal.ai)]
                       │
                       ▼
                 [VLM Critic (Evaluation & Self-Correction)]
                       │
                       ▼
                 [Persist to SQLite & Return Output]
```

See [SPEC.md](SPEC.md) for the full architectural specification.

---

## 🚀 Quick Start (Coming Soon)

```bash
# Generate an image directly via CLI
aris gen "a majestic dragon perched on a crystal mountain at sunset" --ratio 16:9

# Launch interactive chat session
aris chat

# Manage Knowledge Graph memories
aris memory list --scope style
```

---

## 🗺️ Roadmap

- [x] Technical Specification & Hexagonal Architecture (`SPEC.md`)
- [ ] Core Domain Models & Ports
- [ ] SQLite Knowledge Graph with FTS5 (Port from GAIA)
- [ ] Multi-provider LLM Reasoner (OpenAI, Anthropic, Ollama)
- [ ] Zero-config Pollinations Backend
- [ ] CLI & Interactive TUI (Bubbletea + terminal image rendering)
- [ ] Wails v2 Desktop App & Gateways

---

## 📄 License

MIT License — see [LICENSE](LICENSE).
