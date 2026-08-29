# Proposal: Autonomous Agent Core with Multi-Backend Image Generation

## 1. Problem Statement
Users wanting to generate AI images are currently forced to either:
1. Manually write complex model-specific technical tags (seeds, CFG, samplers, negative keywords, camera settings), requiring deep prompt engineering expertise.
2. Use restrictive single-vendor proprietary GUIs that lock them into one paid cloud provider or require heavy Python/ComfyUI manual setups.

There is no single lightweight, pure-Go autonomous agent that provides:
- Natural language intent translation to multi-model technical prompt specs.
- Flexible choice between **Local GPU generation (ComfyUI / Diffusers / Ollama)** and **Multiple Cloud APIs (Pollinations, Fal.ai, Replicate, OpenAI DALL-E, HuggingFace)**.
- Long-term memory of user aesthetics, character consistency, and negative defaults via a persistent 3-scope Knowledge Graph.

## 2. Proposed Solution
Build the complete **ARIS Core Engine** in Go following strict Hexagonal Architecture:
- **Autonomous Reasoner Loop**: Recall -> Reason -> Prompt Architect -> Dispatch -> Critic -> Persist.
- **Pluggable Multi-Backend Image Architecture**:
  - `pollinations`: Zero-config free cloud backend (Flux / Turbo).
  - `comfyui`: High-performance local GPU generation via WebSocket + REST with workflow JSONs.
  - `falai`: Ultra-fast managed Flux Pro / Realism inference.
  - `replicate`: Community models & Flux Schnell.
  - `openai`: DALL-E 3 API.
  - `huggingface`: Inference API for Stable Diffusion 3.5.
- **Backend Selector & Configuration**: Interactive and CLI flags allowing users to switch backend per prompt or persist a default in `~/.aris/config.yaml`.
- **Knowledge Graph Integration**: 3-scope SQLite + FTS5 memory recall (User, Style, Project).

## 3. Scope Boundaries
- **In Scope**:
  - Hexagonal ports & domain models for multi-backend image generation.
  - Implementations for Pollinations, ComfyUI, Fal.ai, Replicate, OpenAI, HuggingFace adapters.
  - CLI commands to select backends dynamically (`--backend comfyui`, `--backend falai`, etc.).
  - Config management with environment variable overrides.
  - Unit tests with HTTP mocks and full test coverage.
- **Out of Scope for this Change**:
  - Full Desktop Wails GUI frontend (scheduled for next phase).
  - Telegram/Discord remote gateways (scheduled for next phase).
