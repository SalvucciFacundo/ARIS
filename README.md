# ARIS — Autonomous Reasoner for Image System

<p align="center">
  <img src="assets/hero_banner.jpg" alt="ARIS — Autonomous Reasoner for Image System" width="100%">
</p>

<p align="center">
  <a href="https://go.dev"><img src="https://img.shields.io/badge/Go-1.23+-00ADD8?style=for-the-badge&logo=go" alt="Go Version"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge" alt="License"></a>
  <img src="https://img.shields.io/badge/Architecture-Hexagonal%20(Ports%20%26%20Adapters)-blueviolet?style=for-the-badge" alt="Hexagonal Architecture">
  <img src="https://img.shields.io/badge/Tests-Strict%20TDD%20%26%20Race%20Clean-success?style=for-the-badge" alt="Test Status">
  <img src="https://img.shields.io/badge/UI-Templ%20%2B%20Islands%20%2B%20HTMX-orange?style=for-the-badge" alt="UI Stack">
</p>

<p align="center">
  <b>Autonomous AI Art Director, Visual Synthesizer & Image Reasoner built in pure Go.</b><br>
  <i>From natural language intent to model-tailored diffusion, VLM self-correction, masked inpainting, and multi-platform interfaces.</i>
</p>

---

## 🏷️ Topics & Keywords
`golang` • `ai-agent` • `image-generation` • `art-director` • `flux` • `comfyui` • `fal-ai` • `dall-e-3` • `inpainting` • `img2img` • `templ` • `htmx` • `templ-islands` • `tailwindcss` • `bubbletea` • `tui` • `telegram-bot` • `discord-bot` • `hexagonal-architecture` • `sqlite-fts5`

---

## 🌟 Executive Summary & Vision

Unlike naive prompt wrappers or heavy GUI frontends, **ARIS (Autonomous Reasoner for Image System)** behaves as an **Art Director + Autonomous Engineer**:

1. **Understands Natural Intent**: Deconstructs vague or complex user prompts into technical parameters (lighting, composition, lenses, aspect ratios, CFG scales, seeds, and negative triggers).
2. **Consults Memory**: Recalls user aesthetic preferences, character/style consistency, and negative defaults from a persistent 3-scope SQLite Knowledge Graph (FTS5).
3. **Multi-Backend Synthesis**: Dispatches jobs across zero-config cloud backends (Pollinations, Fal.ai, Replicate, OpenAI DALL-E) or local GPU setups (ComfyUI via WebSocket/REST).
4. **VLM Critic & Self-Healing**: Optionally inspects rendered outputs with Vision Language Models (Qwen2.5-VL / GPT-4o) to verify anatomical accuracy and constraint satisfaction before final delivery.
5. **Image-to-Image & Masked Inpainting**: Supports iterative visual editing (`aris edit`) with denoise strength control and interactive canvas mask drawing.
6. **Unified Multi-Interface Ecosystem**: Single binary providing a Cyberpunk Bubbletea TUI, Telegram & Discord bots, and a zero-Node Desktop/Web GUI (Templ + Templ Islands + HTMX + Tailwind).

---

## 🏛️ Hexagonal Architecture (Ports & Adapters)

```text
┌─────────────────────────────────────────────────────────────────────────────┐
│                                  ARIS CORE                                  │
│                                                                             │
│   ┌─────────────────────────────────────────────────────────────────────┐   │
│   │                        ORCHESTRATOR / AGENT                         │   │
│   │  • ReAct Loop: Reason -> Plan -> Query Memory -> Synthesize -> Critic│  │
│   │  • Specialized Visual Subagents (@director, @promptsmith, etc.)     │   │
│   │  • Knowledge Graph Auto-Learning & Style Decomposer                 │   │
│   └──────────────────────────────────┬──────────────────────────────────┘   │
│                                      │                                      │
│   ┌──────────────────────────────────▼──────────────────────────────────┐   │
│   │                                PORTS                                │   │
│   │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │   │
│   │  │   LLMProvider    │  │   ImageBackend   │  │   VisionCritic   │   │   │
│   │  └──────────────────┘  └──────────────────┘  └──────────────────┘   │   │
│   │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐   │   │
│   │  │  KnowledgeGraph  │  │   HistoryStore   │  │  GatewayAdapter  │   │   │
│   │  └──────────────────┘  └──────────────────┘  └──────────────────┘   │   │
│   └──────────────────────────────────┬──────────────────────────────────┘   │
└──────────────────────────────────────┼──────────────────────────────────────┘
                                       │
┌──────────────────────────────────────▼──────────────────────────────────────┐
│                                  ADAPTERS                                   │
│                                                                             │
│  [LLM]           OpenAI, Anthropic, Ollama, DeepSeek, OpenRouter, Groq      │
│  [Image Backend] Pollinations (Default/Free), ComfyUI (Local), Fal.ai,      │
│                  Replicate, OpenAI DALL-E 3, HuggingFace                    │
│  [Vision Critic] Ollama (Qwen2.5-VL), OpenAI / Claude Vision                │
│  [Storage / KG]  SQLite3 + FTS5 Full-Text Search (GAIA 3-Scope Model)       │
│  [Gateways]      Telegram Bot & Discord Bot with JobQueue concurrency       │
│  [Presentation]  • Interactive Cyberpunk TUI (Bubbletea + Lipgloss)         │
│                  • Desktop GUI & Remote VPS Web (Templ + Islands + HTMX)    │
│                  • Headless CLI (`aris gen`, `aris edit`, `aris serve`)     │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## 🤖 Specialized Visual Subagents

ARIS features built-in expert visual subagents that can be targeted via direct `@name` prefix in any interface:

| Subagent | Role | Specialty |
|---|---|---|
| `@director` | Art Director & Master Stylist | High-concept cinematography, lighting composition, scene coherence. |
| `@promptsmith` | Prompt Optimization Engineer | Converting keywords into model-specific positive and negative syntax. |
| `@photorealism` | Optical & Camera Specialist | Shutter speed, focal length, bokeh, ISO, RAW photography realism. |
| `@anime` | Japanese Animation Director | Cel-shading, vibrant palettes, Makoto Shinkai / Studio Ghibli aesthetics. |
| `@concept-art` | Production Concept Artist | Worldbuilding, matte painting, sci-fi/fantasy environment staging. |
| `@cyberpunk` | Neo-Tokyo & Sci-Fi Specialist | Volumetric neon lighting, holograms, chrome reflections, gritty tech. |
| `@pixelart` | Retro Pixel Specialist | Isometric pixel art, 16-bit palettes, clean dithering and sprite design. |
| `@critic` | Quality & Constraint Evaluator | Anatomical checks, text placement, color balance, lighting verification. |
| `@inpainter` | Inpainting & Blending Artist | Seamless mask repairs, object insertion/removal, edge blending. |
| `@restyler` | Style Transfer Specialist | Image-to-image restyling with precise denoise strength calibration. |

---

## 🚀 Installation & Quick Start

### ⚡ One-Line Install (Linux & macOS)
```bash
curl -fsSL https://raw.githubusercontent.com/SalvucciFacundo/ARIS/main/install.sh | bash
```

### 🐹 Install via Go (Go 1.23+)
```bash
go install github.com/SalvucciFacundo/ARIS/cmd/aris@latest
```

### 📦 Manual Binary Download
Download the pre-compiled binary for your OS (Linux, macOS Apple Silicon/Intel, Windows) from the [GitHub Releases](https://github.com/SalvucciFacundo/ARIS/releases) page.

### 🛠️ Build from Source
```bash
git clone https://github.com/SalvucciFacundo/ARIS.git
cd ARIS
go build -o aris ./cmd/aris
```

### 1. Text-to-Image Generation (`aris gen`)
```bash
# Zero-config generation via Pollinations (default)
./aris gen "a cyberpunk samurai cat in neo-tokyo, raining, cinematic lighting" --ratio 16:9

# Route directly to a specialized subagent
./aris gen "@director cinematic shot of an astronaut discovering ancient alien monolith"

# Use high-performance local GPU via ComfyUI
./aris gen "hyperrealistic portrait of a cyborg explorer" --backend comfyui --model flux

# Enable Vision Critic self-correction loop
./aris gen "futuristic supercar in futuristic desert" --critic --auto-heal
```

### 2. Image-to-Image & Inpainting (`aris edit`)
```bash
# Transform an existing image with denoise strength control
./aris edit input.png "cyberpunk neon overhaul, raining streets" --strength 0.65 --backend falai

# Localized inpainting with a mask image
./aris edit portrait.png "remove sunglasses, realistic eyes" --mask mask.png --backend comfyui
```

### 3. Interactive Cyberpunk TUI (`aris tui` / `aris chat`)
```bash
./aris tui
```
- **Left Panel**: Conversational chat stream with live subagent thought accordion.
- **Right Panel**: Real-time inspector with keyboard shortcuts (`Tab`: Ratio, `Ctrl+B`: Backend, `Ctrl+O`: Open full-res image).

### 4. Remote Messaging Gateways (`aris gateway`)
Run concurrent Telegram and Discord bots with centralized concurrency protection:
```bash
export TELEGRAM_BOT_TOKEN="your-telegram-token"
export DISCORD_BOT_TOKEN="your-discord-token"
./aris gateway
```

### 5. Desktop GUI & Remote VPS Server (`aris serve` / `aris gui`)
```bash
# Launch Headless Web Server on VPS with token security
./aris serve --host 0.0.0.0 --port 8080 --token "your-secret-token"

# Open Desktop Window connected to local in-process engine
./aris gui

# Open Desktop Window connected to your remote VPS
./aris gui --remote https://vps.yourdomain.com:8080 --token "your-secret-token"
```

---

## ⚙️ Configuration (`~/.aris/config.yaml`)

```yaml
general:
  default_backend: "pollinations"
  default_model: "flux"
  default_aspect_ratio: "16:9"
  save_dir: "./outputs"

llm:
  provider: "ollama"
  model: "qwen2.5-coder"
  base_url: "http://localhost:11434"

backends:
  comfyui:
    url: "http://localhost:8188"
  falai:
    api_key: "your-fal-key"
  openai:
    api_key: "your-openai-key"

gateway:
  concurrency: 2
  max_queue: 10
  telegram:
    enabled: true
    bot_token: "tg-token"
    allowed_chat_ids: [12345678]
  discord:
    enabled: true
    bot_token: "dc-token"
    allowed_channel_ids: ["987654321"]

ui:
  host: "127.0.0.1"
  port: 8080
  auth_token: ""
```

---

## 🤝 Contributing & Community

We welcome contributions from the community! Whether you want to add a new image backend, design a new visual subagent, improve the TUI/GUI, or report a bug:

### How to Contribute

1. **Check Existing Issues**: Search open [Issues](https://github.com/SalvucciFacundo/ARIS/issues) before opening a new one to avoid duplicates.
2. **Open an Issue**:
   - 🐛 **Bug Reports**: Include OS/Arch, Go version, backend used, reproduction steps, and error logs.
   - 💡 **Feature Requests**: Describe the proposed capability, motivation, and architectural fit.
3. **Fork & Branch**: Create a feature branch (`git checkout -b feat/my-new-backend`).
4. **Follow Strict TDD & Code Standards**:
   - Write unit tests first (RED -> GREEN).
   - Verify zero data races:
     ```bash
     go test -count=1 -race ./...
     ```
   - Ensure clean formatting:
     ```bash
     go vet ./...
     ```
5. **Commit with Conventional Commits**:
   - `feat: add Stability AI SD3.5 backend adapter`
   - `fix: resolve aspect ratio calculation in 21:9 ultrawide`
   - `docs: update gateway configuration guide`
6. **Submit a Pull Request**: Provide a clear summary of your changes, reference any related issues, and ensure CI passes.

---

## 📄 License

Distributed under the **MIT License**. See [`LICENSE`](LICENSE) for details.
