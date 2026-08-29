# Specification: Autonomous Agent & Memory Reasoning Loop

## Requirements

### REQ-AG-1: 5-Stage Autonomous Execution
When `agent.Generate(ctx, prompt, opts)` is called:
1. **Recall**: Query Knowledge Graph using FTS5 to extract relevant user aesthetic rules and style recipes.
2. **Reason**: Prompt Architect converts raw input + recalled facts into an optimized `ImageSpec`.
3. **Dispatch**: Routes `ImageSpec` to the selected `ImageBackend` from the registry.
4. **Critic**: Optional Vision VLM evaluation for prompt constraint adherence.
5. **Persist**: Stores generation parameters, image path, and duration in `generations` SQLite table.

### REQ-AG-2: Conversational Prompt Refinement
- When modifying a previous generation (e.g. "make the lighting warmer", "remove the sunglasses"):
  - Agent loads the previous `ImageSpec`.
  - Applies delta prompt transformations while retaining style consistency.
  - Generates a new seed or reuses the previous seed depending on user request.

### REQ-AG-3: Knowledge Graph Operations
- Users can manage facts via CLI:
  - `aris memory list [--scope user|style|project]`
  - `aris memory add --topic <t> --concept <c> --fact <f> [--scope <s>]`
  - `aris memory search "<keyword>"`
  - `aris memory delete <id>`
- Agent automatically creates `KnowledgeFact` entries when users rate generations with positive/negative feedback.
