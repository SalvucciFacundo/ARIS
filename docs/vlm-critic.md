# ARIS Vision Critic & Self-Healing Loop

ARIS includes a vision-language model (VLM) evaluation loop that inspects generated images against user prompt constraints, anatomical consistency, and lighting rules before delivering them.

---

## Evaluation & Self-Healing Workflow

```text
[Generated Image] ──► [Vision Critic Evaluator (Qwen2.5-VL / GPT-4o Vision)]
                                 │
                                 ▼
                     Score >= Threshold (0.60)?
                     ├── YES ──► [Deliver Image to User]
                     └── NO  ──► [Self-Healing Prompt Refinement]
                                        │
                                        ▼
                                 [Re-roll Image Generation]
                                        │
                                        ▼
                                 [Deliver Healed Image]
```

---

## Supported Vision Models

1. **Ollama Local**: `qwen2.5-vl`, `llava`, `granite-vision` (zero cloud cost, completely private).
2. **Cloud APIs**: `gpt-4o-mini`, `gpt-4o`, `claude-3-5-sonnet`.

---

## Enabling Critique & Self-Healing

In CLI:
```bash
# Enable critique report
aris gen "cyberpunk street scene" --critic

# Enable critique + automated self-healing re-roll
aris gen "cyberpunk street scene" --critic --auto-heal
```

In Python / Web GUI:
Toggle the **"Vision Critic"** and **"Auto-Heal"** switches in the Right Control Panel.
