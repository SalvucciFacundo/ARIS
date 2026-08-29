# Design: TUI Architecture & Auto-Learning Engine

## 1. Auto-Learning Architecture

```
[User Input / Correction]
           │
           ▼
     [AgentService] ──► Generates Image Spec & Result
           │
           ▼ (Async Goroutine)
     [AutoLearner]
           │
     ┌─────┴────────────────────────────┐
     ▼                                  ▼
[LLM / Heuristic Reflection]   [Contradiction & Deduplication Check]
     │                                  │
     └──────────────────┬───────────────┘
                        │
                        ▼
         [SQLite Knowledge Graph Store]
           (scope: user | style | project)
```

### AutoLearner Reflection Protocol
```go
package services

type ReflectionTurn struct {
    RawInput       string
    PreviousPrompt string
    EnhancedPrompt string
    NegativePrompt string
    UserFeedback   string
}

type AutoLearner interface {
    ReflectTurn(ctx context.Context, turn ReflectionTurn) ([]domain.KnowledgeFact, error)
}
```

## 2. Bubbletea TUI State Machine

```
                 ┌────────────────────────────────┐
                 │       TUI Model (State)        │
                 └───────────────┬────────────────┘
                                 │
     ┌───────────────────────────┼───────────────────────────┐
     ▼                           ▼                           ▼
[Chat Viewport]          [Input Textarea]           [Side Inspector]
• Streamed Thoughts      • Prompt input             • Active Backend/Ratio
• User messages          • Multi-line buffer        • ANSI Image Preview
• Spec cards             • History recall           • Recalled Facts
```
