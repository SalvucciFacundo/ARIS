# Specification: VLM Vision Critic & Quality Assurance

## Requirements

### REQ-VLM-1: Image Encoding & Payload Construction
- Given a local image path (`.jpg`, `.png`, `.webp`), the vision adapter must encode the bytes to a standard Base64 Data URL (`data:image/jpeg;base64,...`).
- Constructs a multi-modal message containing:
  1. System instruction defining criteria for evaluation.
  2. Image data URL.
  3. Original user prompt and technical spec.

### REQ-VLM-2: Structured Evaluation Output
The VLM must respond with a JSON object:
```json
{
  "score": 0.85,
  "adherence": "High - cat and sunglasses are clearly visible",
  "defects": "Minor blur in the far background",
  "suggested_fix": "Add 'crisp focus' to positive prompt"
}
```

### REQ-VLM-3: Self-Healing Trigger
- Threshold: Configurable (default `0.60`).
- Max Retries: Exactly 1 automated re-roll per generation turn to prevent runaway loops.
- When triggered:
  - Updates negative prompt with detected defects.
  - Generates a new random seed.
  - Re-invokes `ImageBackend.Generate`.
  - Logs self-healing event in generation metadata (`self_healed: true`, `initial_score: 0.45`, `final_score: 0.88`).

### REQ-VLM-4: Configuration & CLI Flags
- Config (`~/.aris/config.yaml`):
  ```yaml
  critic:
    enabled: true
    provider: "ollama" # or "openai", "openrouter"
    model: "qwen2.5-vl"
    threshold: 0.60
    auto_heal: true
  ```
- CLI Flags:
  - `--critic`: Force run vision critique on this generation.
  - `--auto-heal`: Enable automated retry if critique score is below threshold.
