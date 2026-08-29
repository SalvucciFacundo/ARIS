# ARIS: Autonomous Reasoner for Image System
## Technical Specification & Architecture Design (v1.0)

---

## 1. Executive Summary & Vision

**ARIS (Autonomous Reasoner for Image System)** is an autonomous, AI-driven visual generation agent built in **pure Go**.

Unlike naive prompt wrappers or rigid GUI frontends, ARIS behaves as an **Art Director + Autonomous Engineer**:
1. **Understands Intent**: Analyzes vague or complex user descriptions in natural language.
2. **Consults Memory**: Recalls user aesthetic preferences, character/style consistency, camera setups, and negative defaults from a persistent SQLite Knowledge Graph (inherited from the GAIA architecture).
3. **Engineers Prompts & Parameters**: Deconstructs requests into model-specific positive prompts, negative triggers, lighting/composition parameters, aspect ratios, seeds, and samplers (optimized for Flux, SDXL, or DALL-E).
4. **Executes Image Synthesis**: Dispatches jobs across interchangeable local (ComfyUI/Diffusers) and cloud (Pollinations, Fal.ai, Replicate, OpenAI) image backends.
5. **Evaluates & Self-Corrects (VLM Critic Loop)**: Optionally inspects rendered outputs with a Vision Language Model to verify constraint satisfaction (e.g., correct anatomy, element count, lighting match) before final delivery.
6. **Iterative Refinement & Img2Img**: Supports conversational image editing ("make it darker", "remove the helmet", or inputting a reference image).
7. **Learns & Retains**: Records successful prompt recipes, style discoveries, and user feedback back into the Knowledge Graph.

---

## 2. Core Architecture: Hexagonal (Ports & Adapters)

ARIS follows strict Hexagonal Architecture principles in Go to guarantee zero vendor lock-in, high testability, and single-binary deployment.

```
+-------------------------------------------------------------------------------+
|                                  ARIS CORE                                    |
|                                                                               |
|   +-----------------------------------------------------------------------+   |
|   |                        ORCHESTRATOR / AGENT                           |   |
|   |  • ReAct Loop: Reason -> Plan -> Query Memory -> Synthesize -> Critic |   |
|   |  • Style Decomposer & Prompt Architect (Text2Img & Img2Img)           |   |
|   |  • Context Compactor & Memory Nudge                                   |   |
|   +-----------------------------------+-----------------------------------+   |
|                                       |                                       |
|   +-----------------------------------v-----------------------------------+   |
|   |                             PORTS                                     |   |
|   |  +-------------------+  +-------------------+  +-------------------+  |   |
|   |  |   LLMProvider     |  |   ImageBackend    |  |    VisionCritic   |  |   |
|   |  +-------------------+  +-------------------+  +-------------------+  |   |
|   |  +-------------------+  +-------------------+  +-------------------+  |   |
|   |  |  KnowledgeGraph   |  |    HistoryStore   |  |     UIPresenter   |  |   |
|   |  +-------------------+  +-------------------+  +-------------------+  |   |
|   +-----------------------------------+-----------------------------------+   |
+---------------------------------------|---------------------------------------+
                                        |
+---------------------------------------v---------------------------------------+
|                                ADAPTERS                                       |
|                                                                               |
|  [LLM]           OpenAI, Anthropic, Ollama, DeepSeek, OpenRouter, Groq        |
|  [Image Backend] Pollinations (Default/Free), ComfyUI (Local), Fal.ai,        |
|                  Replicate, OpenAI DALL-E, HuggingFace                       |
|  [Vision Critic] Ollama (Qwen2.5-VL/Granite-Vision), OpenAI/Claude Vision     |
|  [Storage/KG]    SQLite3 + FTS5 Full-Text Search (GAIA 3-Scope Model)         |
|  [Presentation]  • Cyberpunk TUI (Bubbletea + Kitty/Sixel/iTerm2 protocol)    |
|                  • Desktop GUI (Wails v2 + TailwindCSS + Vanilla JS)          |
|                  • CLI Headless Mode (`aris gen "..."`)                       |
|                  • Gateways: Telegram & Discord Bot adapters                  |
+-------------------------------------------------------------------------------+
```

---

## 3. The Autonomous Generation & Reasoning Lifecycle

Every user request undergoes a 5-stage autonomous lifecycle:

```
[User Input (Text / Image)]
           │
           ▼
1. REASON & RECALL ──► Query Knowledge Graph (User style, artist facts, negative rules)
           │
           ▼
2. PROMPT ARCHITECT ─► Deconstruct into Positive, Negative, Aspect Ratio, Seed, CFG
           │
           ▼
3. DISPATCH & RENDER ─► Invoke Image Backend (Pollinations / ComfyUI / Fal.ai / Replicate)
           │
           ▼
4. VLM EVALUATION ───► (Optional) Vision Critic verifies constraints -> Pass or Refine Loop
           │
           ▼
5. PERSIST & LEARN ──► Save Image, Metadata, Generation Recipe, and new Facts to SQLite
           │
           ▼
[Deliver Result + Display Preview / Stream to UI / Gateway]
```

### 3.1. Stage 1: Reason & Recall
- Search SQLite Knowledge Graph using FTS5 for topics related to the user prompt (e.g. style tokens like `cyberpunk`, `watercolor`, user preferences like `always use 16:9 for landscapes`, negative defaults like `no text, no watermark`).
- Inject recalled facts into the system context.

### 3.2. Stage 2: Prompt Architect
- LLM acts as an expert Art Director.
- Maps raw concepts to model-specific syntax (e.g. Flux natural language vs. SDXL comma-separated weight tags `(masterpiece:1.2), volumetric lighting, 8k octane render`).
- Computes resolution, aspect ratio (1:1, 16:9, 9:16, 4:5), sampling steps, guidance scale (CFG), and seed.
- Handles conversational iterations (e.g. "make it darker", "remove the helmet") by modifying prompt deltas and maintaining seed or setting a new one.

### 3.3. Stage 3: Dispatch & Render
- Calls the active `ImageBackend`.
- **Default Out-of-the-Box**: Pollinations API (`image.pollinations.ai`) — Zero configuration, free, fast (~1-2s).
- **Local Powerhouse**: ComfyUI API via WebSocket / REST for users with local GPUs.
- **Managed APIs**: Fal.ai, Replicate, OpenAI DALL-E 3 for production quality.
- Downloads the resulting asset to local cache (`~/.aris/outputs/YYYY-MM-DD/`).

### 3.4. Stage 4: VLM Evaluation & Critic (Self-Correction Loop)
- If enabled, the generated image is sent to a Vision model (`VisionCritic`).
- Evaluates against the prompt:
  - Are all requested subjects present?
  - Are there visible artifacts or style mismatches?
- If score < threshold, agent adjusts the prompt/seed and initiates a single targeted correction attempt.

### 3.5. Stage 5: Persist & Learn
- Persists image path, raw seed, final prompt, and render duration to `generations` table.
- If the user provides positive or corrective feedback, ARIS extracts a new `KnowledgeFact` and saves it to SQLite.

---

## 4. GAIA-Inherited Memory System (SQLite Knowledge Graph)

ARIS adopts GAIA's 3-scope memory model adapted for visual synthesis:

### 4.1. The Three Scopes
1. **User Scope (`scope: "user"`)**:
   - Universal preferences: favorite aspect ratios, default color tones, negative triggers (e.g. "never add text watermark", "prefers cinematic 35mm film grain").
2. **Style Scope (`scope: "style"`)**:
   - Curated artist recipes, lighting presets, rendering engine prompts, camera focal lengths, and model-specific nuances (e.g., "Flux.1 handles text in quotes natively without trigger words").
3. **Session / Project Scope (`scope: "project"`)**:
   - Character sheets, visual consistency tokens, seed references, palette locks for a specific image series or campaign.

### 4.2. Database Schema (`internal/adapters/db/schema.sql`)

```sql
-- Knowledge Graph Facts Table
CREATE TABLE IF NOT EXISTS knowledge_facts (
    id TEXT PRIMARY KEY,
    topic TEXT NOT NULL,          -- e.g. "style:cyberpunk", "artist:moebius", "pref:aspect_ratio"
    concept TEXT NOT NULL,        -- e.g. "lighting", "negative_prompt", "seed_stability"
    fact TEXT NOT NULL,           -- e.g. "Use volumetric neon fog with teal and orange chromatic aberration"
    source_agent TEXT NOT NULL,   -- "aris:reasoner", "user:feedback", "critic:vlm"
    labels TEXT NOT NULL,         -- JSON array: ["cyberpunk", "lighting", "neon"]
    created_at DATETIME NOT NULL,
    project TEXT DEFAULT '',      -- project / collection name
    scope TEXT NOT NULL           -- "user", "style", "project"
);

-- Full-Text Search (FTS5) for fast semantic keyword matching
CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_facts_fts USING fts5(
    topic,
    concept,
    fact,
    labels,
    content='knowledge_facts',
    content_rowid='rowid'
);

-- Generation History Table
CREATE TABLE IF NOT EXISTS generations (
    id TEXT PRIMARY KEY,
    prompt_raw TEXT NOT NULL,
    prompt_enhanced TEXT NOT NULL,
    negative_prompt TEXT,
    backend TEXT NOT NULL,
    model TEXT NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    steps INTEGER,
    cfg_scale REAL,
    seed INT64,
    image_path TEXT NOT NULL,
    thumb_path TEXT,
    duration_ms INTEGER,
    rating INTEGER DEFAULT 0,     -- -1 (thumbs down), 0 (neutral), 1 (thumbs up)
    feedback TEXT,
    created_at DATETIME NOT NULL
);
```

---

## 5. Domain Models & Port Interfaces in Go

### 5.1. Domain Models (`internal/core/domain/`)

```go
package domain

import (
	"time"
)

type AspectRatio string

const (
	RatioSquare    AspectRatio = "1:1"   // 1024x1024
	RatioLandscape AspectRatio = "16:9"  // 1344x768
	RatioPortrait  AspectRatio = "9:16"  // 768x1344
	RatioPhoto     AspectRatio = "4:3"   // 1152x864
	RatioPoster    AspectRatio = "3:4"   // 864x1152
)

type ImageSpec struct {
	ID             string            `json:"id"`
	RawPrompt      string            `json:"raw_prompt"`
	EnhancedPrompt string            `json:"enhanced_prompt"`
	NegativePrompt string            `json:"negative_prompt"`
	AspectRatio    AspectRatio       `json:"aspect_ratio"`
	Width          int               `json:"width"`
	Height         int               `json:"height"`
	Steps          int               `json:"steps"`
	CFGScale       float64           `json:"cfg_scale"`
	Seed           int64             `json:"seed"`
	Backend        string            `json:"backend"`
	Model          string            `json:"model"`
	InputImagePath string            `json:"input_image_path,omitempty"` // For img2img
	ExtraParams    map[string]any    `json:"extra_params"`
	CreatedAt      time.Time         `json:"created_at"`
}

type ImageResult struct {
	ID           string        `json:"id"`
	SpecID       string        `json:"spec_id"`
	LocalPath    string        `json:"local_path"`
	RemoteURL    string        `json:"remote_url,omitempty"`
	Format       string        `json:"format"` // png, webp, jpg
	SizeInBytes  int64         `json:"size_in_bytes"`
	Duration     time.Duration `json:"duration"`
	Metadata     map[string]any `json:"metadata"`
}

type KnowledgeFact struct {
	ID          string    `json:"id"`
	Topic       string    `json:"topic"`
	Concept     string    `json:"concept"`
	Fact        string    `json:"fact"`
	SourceAgent string    `json:"source_agent"`
	Labels      []string  `json:"labels"`
	Project     string    `json:"project"`
	Scope       string    `json:"scope"` // "user", "style", "project"
	CreatedAt   time.Time `json:"created_at"`
}
```

### 5.2. Core Ports (`internal/core/ports/`)

```go
package ports

import (
	"context"
	"aris/internal/core/domain"
)

// LLMProvider generates reasoning, prompt engineering, and parameter selection.
type LLMProvider interface {
	Name() string
	Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	ReasonPrompt(ctx context.Context, input string, facts []domain.KnowledgeFact) (*domain.ImageSpec, error)
}

// ImageBackend handles actual image rendering.
type ImageBackend interface {
	Name() string
	SupportsModels() []string
	Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error)
}

// VisionCritic evaluates generated images against quality and prompt constraints.
type VisionCritic interface {
	Evaluate(ctx context.Context, imagePath string, spec *domain.ImageSpec) (score float64, critique string, err error)
}

// KnowledgeGraphStore handles persistent 3-scope memory and full-text search.
type KnowledgeGraphStore interface {
	AddFact(ctx context.Context, fact domain.KnowledgeFact) (string, error)
	SearchFacts(ctx context.Context, query string, scope string, limit int) ([]domain.KnowledgeFact, error)
	GetFactsByTopic(ctx context.Context, topic string) ([]domain.KnowledgeFact, error)
	DeleteFact(ctx context.Context, id string) error
}

// HistoryStore records generation runs, specs, outputs, and ratings.
type HistoryStore interface {
	SaveGeneration(ctx context.Context, spec *domain.ImageSpec, result *domain.ImageResult) error
	UpdateRating(ctx context.Context, id string, rating int, feedback string) error
	GetHistory(ctx context.Context, limit, offset int) ([]domain.ImageResult, error)
}
```

---

## 6. Image Generation Backends (The Muscle)

ARIS supports multiple interchangeable backends via the `ImageBackend` interface:

| Backend | Type | Default Model | Cost / Auth | Notes |
|---|---|---|---|---|
| **Pollinations** | Free HTTP API | `flux`, `turbo` | Zero-Config, Free | Default out-of-the-box backend with no API keys required. |
| **ComfyUI** | Local HTTP/WS | Custom (SDXL / Flux) | Local Hardware | Connects to local ComfyUI instance with custom workflow JSONs. |
| **Fal.ai** | Managed Cloud API | `flux-pro`, `flux-realism` | API Key | Ultra-fast enterprise cloud rendering. |
| **Replicate** | Managed Cloud API | `black-forest-labs/flux-schnell` | API Key | Diverse community models. |
| **OpenAI** | Managed Cloud API | `dall-e-3` | API Key | High prompt coherence. |
| **HuggingFace**| Cloud / Local | `stable-diffusion-3.5` | Optional Token | HF Inference API integration. |

### Pollinations Endpoint Specification:
```
GET https://image.pollinations.ai/prompt/{prompt}?width={w}&height={h}&seed={seed}&nologo=true&model={model}&negative={negative}
```

---

## 7. Package Layout in Go

```
aris/
├── cmd/
│   └── aris/
│       └── main.go                 # Application entrypoint & CLI dispatcher
├── internal/
│   ├── core/
│   │   ├── domain/                 # Core entities (ImageSpec, ImageResult, KnowledgeFact)
│   │   ├── ports/                  # Interface contracts (LLM, Backend, Memory, Critic)
│   │   └── services/               # Core business logic & autonomous reasoner loop
│   │       ├── agent.go            # Master ARIS agent loop
│   │       ├── prompt_architect.go # Art Director & Prompt decomposition
│   │       ├── critic_service.go   # VLM Evaluation & Refinement loop
│   │       └── memory_service.go   # Fact indexing & recall orchestration
│   ├── adapters/
│   │   ├── llm/                    # OpenAI, Anthropic, Ollama, DeepSeek adapters
│   │   │   ├── client.go
│   │   │   ├── openai.go
│   │   │   └── ollama.go
│   │   ├── image/                  # Image generation adapters
│   │   │   ├── pollinations.go     # Zero-config default backend
│   │   │   ├── comfyui.go          # Local ComfyUI websocket/REST adapter
│   │   │   ├── falai.go            # Fal.ai cloud adapter
│   │   │   └── replicate.go        # Replicate cloud adapter
│   │   ├── vision/                 # Vision critique adapters (Qwen-VL, GPT-4o-vision)
│   │   │   └── vision_critic.go
│   │   ├── db/                     # SQLite Knowledge Graph & History
│   │   │   ├── sqlite.go           # DB connection & migrations
│   │   │   ├── knowledge.go        # GAIA-based Knowledge Graph implementation
│   │   │   └── history.go          # Generation logs
│   │   ├── ui/                     # Presentation layers
│   │   │   ├── cli/                # Terminal commands (aris gen "...", aris memory, etc.)
│   │   │   ├── tui/                # Interactive Bubbletea TUI with terminal image preview
│   │   │   └── wails/              # Wails v2 Desktop App bindings (Go <-> JS)
│   │   └── gateway/                # Optional gateways
│   │       ├── telegram/           # Telegram Bot adapter
│   │       └── discord/            # Discord Bot adapter
│   └── config/
│       └── config.go               # YAML / Env configuration loader (~/.aris/config.yaml)
├── pkg/
│   └── imgutil/                    # Image format conversion, thumbnailing, terminal protocols
├── frontend/                       # Optional Wails Desktop App UI (Tailwind + Vanilla JS/Svelte)
├── go.mod
├── go.sum
└── SPEC.md
```

---

## 8. User Interfaces & Gateways

### 8.1. CLI Headless Mode
```bash
# Quick single generation (uses default Pollinations or configured backend)
aris gen "a cyberpunk samurai cat in neo-tokyo, raining, cinematic lighting" --ratio 16:9

# Interactive session with continuous chat, memory recall, and iterative refinement
aris chat

# Manage Knowledge Graph memory
aris memory list --scope style
aris memory add --topic "style:anime" --concept "palette" --fact "Vibrant saturated pastel colors with clean lineart"

# Inspect & browse past generations
aris history --limit 10
```

### 8.2. Interactive TUI (Bubbletea + Lipgloss)
- Split screen: Chat & reasoning stream on the left, rendered image metadata & preview on the right.
- Supports native terminal graphic protocols (**Kitty Graphics Protocol**, **Sixel**, and **iTerm2 inline images**).
- Fallback to automatic OS image viewer (`xdg-open`, `open`, or Windows default viewer) when graphics protocols are unavailable.

### 8.3. Desktop UI (Wails v2)
- 3-Panel Layout:
  - **Left**: Conversation history & image gallery.
  - **Center**: High-res Image Canvas with drag & drop support for img2img reference inputs.
  - **Right**: Chat with ARIS, prompt inspector, parameter sliders (Steps, CFG, Seed, Model).

### 8.4. Gateways (Telegram / Discord)
- Receive image generation requests via Telegram/Discord bots and reply with rendered image attachments in real time.

---

## 9. Implementation Roadmap

### Phase 1: Foundations & MVP (The "Instant" Pipeline)
- [ ] Setup Hexagonal architecture structure (`internal/core`, `internal/adapters`).
- [ ] Implement SQLite Knowledge Graph with FTS5 search (porting GAIA's `knowledge.go`).
- [ ] Implement LLM Adapter (Ollama + OpenAI/OpenRouter compatible).
- [ ] Implement `Pollinations` Image Backend (zero-config, free image generation).
- [ ] Build basic CLI `aris gen "<prompt>"` that executes: `Reason -> Prompt -> Render -> Save to Disk`.

### Phase 2: Autonomous Reasoner & Memory Learning
- [ ] Build `PromptArchitect` service with structured JSON output parsing for positive/negative prompts and model parameters.
- [ ] Implement automated style recall and memory extraction from user prompts.
- [ ] Add `aris memory` command suite (add, search, list, prune facts).
- [ ] Add support for `Fal.ai` and `ComfyUI` backends.

### Phase 3: Vision Critic & Self-Healing Loop
- [ ] Implement `VisionCritic` port with local Ollama VLM (Qwen2.5-VL) and Cloud Vision.
- [ ] Add threshold-based automated refinement loop (re-roll with adjusted prompt/seed if critical flaw detected).
- [ ] Add interactive rating system (`aris rate <id> +1 / -1 --feedback "..."`).

### Phase 4: Interactive TUI, Wails Desktop & Gateways
- [ ] Implement Bubbletea TUI with streaming reasoning, parameter knobs, and inline image preview.
- [ ] Implement Wails v2 Desktop App with drag-and-drop canvas.
- [ ] Add Telegram & Discord bot gateways.
- [ ] Package into single standalone binary for Linux, macOS, and Windows.
