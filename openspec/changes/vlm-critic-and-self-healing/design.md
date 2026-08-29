# Design: VLM Critic & Self-Healing Pipeline

## 1. Sequence Diagram

```
[AgentService] ──► [1. Dispatch Generation] ──► Rendered Image (image_01.jpg)
       │
       ▼
[CriticService] ──► [2. Encode Base64] ──► [3. Invoke VLM (Ollama / OpenAI)]
       │
       ▼
 [Check Score < Threshold?]
       ├── (No, Score >= 0.6) ──► Persist & Deliver
       │
       └── (Yes, Score < 0.6 & AutoHeal=true)
              │
              ▼
    [4. Apply Suggested Fix to Prompt & Pick New Seed]
              │
              ▼
    [5. Re-Dispatch Generation] ──► Rendered Image (image_02.jpg)
              │
              ▼
    [6. Persist Healed Image & Save Critic Notes to SQLite]
```

## 2. Port Contract
```go
package ports

type VisionCritic interface {
    Name() string
    Evaluate(ctx context.Context, imagePath string, spec *domain.ImageSpec) (score float64, critique string, err error)
}
```
