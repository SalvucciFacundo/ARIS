# Design: Visual Subagent Architecture

## 1. Domain Models & Architecture

```
                               ┌──────────────────────┐
                               │   SubagentManager    │
                               └──────────┬───────────┘
                                          │
                  ┌───────────────────────┼───────────────────────┐
                  ▼                       ▼                       ▼
          ┌───────────────┐       ┌───────────────┐       ┌───────────────┐
          │   @director   │       │ @promptsmith  │       │    @critic    │
          │ (Art Director)│       │(Syntax Compiler)      │ (VLM QA & Fix)│
          └───────────────┘       └───────────────┘       └───────────────┘
                  │                       │                       │
                  └───────────────────────┼───────────────────────┘
                                          │
                  ┌───────────────────────┴───────────────────────┐
                  ▼                                               ▼
          ┌───────────────┐                               ┌───────────────┐
          │   @curator    │                               │   @enhancer   │
          │(Memory Curator)                               │ (Post-Process)│
          └───────────────┘                               └───────────────┘
```

### Domain Definition
```go
package domain

type SubagentDef struct {
    Name         string   `json:"name"`          // "director", "promptsmith", etc.
    DisplayName  string   `json:"display_name"`  // "Art Director"
    Role         string   `json:"role"`          // "Conceptualization"
    Description  string   `json:"description"`
    SystemPrompt string   `json:"system_prompt"`
    Personality  string   `json:"personality"`
    Temperature  float64  `json:"temperature"`
    Model        string   `json:"model,omitempty"`
    AllowedTools []string `json:"allowed_tools"`
}
```
