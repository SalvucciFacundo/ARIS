# Specification: Autonomous Learning Loop (Hermes & GAIA Architecture)

## Requirements

### REQ-AL-1: Turn-Based Reflection & Fact Extraction
- After each user prompt or conversational correction, `AutoLearner.Reflect(ctx, turnContext)` is invoked asynchronously.
- The reflection engine detects:
  1. **Aesthetic Preferences**: Colors, lighting, aspect ratios (e.g. *"prefers 16:9 widescreen"*, *"likes high-contrast neon palettes"*).
  2. **Negative Triggers**: Explicit dislikes or unwanted elements (e.g. *"never include sunglasses"*, *"dislikes watermarks or logos"*).
  3. **Artistic Recipes**: Discovered artist styles or rendering techniques (e.g. *"Studio Ghibli style watercolor textures"*).
- Extracted facts are assigned appropriate scopes:
  - `user`: Global aesthetic/negative rules.
  - `style`: Specific art movements, lighting presets, or camera settings.
  - `project`: Character tokens or thematic consistency for an ongoing image series.

### REQ-AL-2: Autonomous Persistence & Deduplication
- Before saving, `AutoLearner` checks if an identical or contradictory fact exists in SQLite.
- If an existing fact contradicts the new rule, the old fact is updated.
- Facts are stored in SQLite `knowledge_facts` with `source_agent: "aris:autolearn"`.

### REQ-AL-3: Nudge & Self-Learning Hooks
- Counter-based learning triggers after every N interactions (or on explicit feedback / rating).
- The prompt architect automatically loads newly learned facts into subsequent generation turns.
