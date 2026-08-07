# MUSE — Multimodal Unified Synthesis Engine

<p align="center">
  <img src="https://img.shields.io/badge/Status-Spec%20Draft-yellow" alt="Status">
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go" alt="Go"></a>
</p>

**MUSE is an autonomous AI image generation agent.** You describe what you want in natural language — *"a dog jumping on the moon, Studio Ghibli style"* — and MUSE understands your request with an LLM, crafts the technical prompt (positive, negative, model, parameters), and renders the image through an interchangeable backend.

> 🎨 **MUSE** = *the muse* — the goddess of artistic inspiration. And the acronym: **M**ultimodal **U**nified **S**ynthesis **E**ngine.

## Status

🚧 **Spec in progress** — the project is being designed. See [SPEC.md](SPEC.md) for the full specification (SDD workflow).

## Planned Architecture

- **UI:** Desktop app (Wails v2 — Go backend + vanilla HTML/Tailwind/JS frontend)
- **Brain:** LLM-powered prompt engine (multi-provider: OpenAI, Anthropic, Ollama)
- **Muscle:** Interchangeable image backends (Pollinations free, ComfyUI local, paid APIs)
- **Memory:** Persistent preferences & history (SQLite/JSON)

## Roadmap

- [ ] **MVP:** desktop app, LLM prompt engine, Pollinations backend, memory, iteration
- [ ] **Fase 2:** Telegram/Discord gateways, ComfyUI local, img2img editing
- [ ] **Fase 3:** paid API backends, advanced editing (inpainting/outpainting)

---

MIT License — see [LICENSE](LICENSE).
