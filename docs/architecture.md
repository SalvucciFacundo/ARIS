# ARIS Hexagonal Architecture & Design Principles

ARIS is designed from first principles following strict **Hexagonal Architecture (Ports & Adapters)** in Go. The core business logic is completely isolated from LLM providers, diffusion backends, persistence stores, and presentation adapters.

---

## Architecture Blueprint

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

## Core Domain (`internal/core/domain`)

- **`ImageSpec`**: The canonical technical blueprint for an image synthesis request. Encapsulates raw prompt, enhanced prompt, negative prompt, aspect ratio, resolution dimensions, steps, CFG scale, seed, backend, model, input image path, inpainting mask path, and denoise strength.
- **`ImageResult`**: The output asset record with local path, remote URL, format, byte size, duration, and metadata (critic scores, self-healing status).
- **`KnowledgeFact`**: An atomic recalled or learned fact in SQLite across `user`, `style`, or `project` scopes.
- **`SubagentDef`**: Definition of a visual subagent (name, display name, role, system prompt, temperature, allowed tools).

---

## Ports (`internal/core/ports`)

```go
type LLMProvider interface {
    Name() string
    Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error)
    ReasonPrompt(ctx context.Context, input string, facts []domain.KnowledgeFact) (*domain.ImageSpec, error)
}

type ImageBackend interface {
    Name() string
    SupportsModels() []string
    Generate(ctx context.Context, spec *domain.ImageSpec) (*domain.ImageResult, error)
}

type VisionCritic interface {
    Evaluate(ctx context.Context, imagePath string, spec *domain.ImageSpec) (score float64, critique string, err error)
}

type KnowledgeGraphStore interface {
    AddFact(ctx context.Context, fact domain.KnowledgeFact) (string, error)
    SearchFacts(ctx context.Context, query string, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error)
    GetFactsByTopic(ctx context.Context, topic string) ([]domain.KnowledgeFact, error)
    ListAllFacts(ctx context.Context, scope domain.MemoryScope, limit int) ([]domain.KnowledgeFact, error)
    DeleteFact(ctx context.Context, id string) error
}
```

---

## Presentation & Gateway Adapters

1. **CLI Adapter (`internal/adapters/ui/cli`)**: Cobra-style dispatcher for shell scripting, one-off generations, and CI/CD automation.
2. **TUI Adapter (`internal/adapters/ui/tui`)**: Split-screen Cyberpunk TUI with Bubbletea, Lipgloss, viewport streaming, and ANSI image rendering.
3. **Web & Desktop Adapter (`internal/adapters/ui/web` & `desktop`)**: Single-binary web server powered by Templ, HTMX, Templ Islands, and Tailwind CSS with native desktop windowing and remote VPS client mode.
4. **Messaging Gateways (`internal/adapters/gateway`)**: Telegram & Discord bots coordinated by a `GatewayMultiplexer` and protected by a bounded `JobQueue` to avoid GPU VRAM exhaustion.
