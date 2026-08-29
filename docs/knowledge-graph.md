# ARIS Knowledge Graph & Memory System

ARIS incorporates a 3-scope Knowledge Graph persistence layer in pure Go using **SQLite3** and **FTS5 (Full-Text Search)**, based on the GAIA memory model.

---

## 3-Scope Memory Architecture

To balance memory recall accuracy and prompt context window size, facts are organized into three distinct scopes:

```text
┌────────────────────────────────────────────────────────────────────────┐
│                        3-SCOPE MEMORY MODEL                            │
├─────────────────┬──────────────────────────────────────────────────────┤
│  Scope: User    │ Global aesthetic preferences (e.g. "prefer 16:9 for  │
│                 │ landscapes", "always avoid motion blur").             │
├─────────────────┼──────────────────────────────────────────────────────┤
│  Scope: Style   │ Curated artistic recipes, lighting techniques, and   │
│                 │ camera setups (e.g. "cyberpunk: volumetric teal fog")│
├─────────────────┼──────────────────────────────────────────────────────┤
│  Scope: Project │ Character consistency sheets, specific campaign      │
│                 │ assets, or recurring thematic rules.                 │
└─────────────────┴──────────────────────────────────────────────────────┘
```

---

## Auto-Learning Loop

During generation, the `AutoLearner` service:
1. Analyzes user prompts and positive feedback.
2. Extracts recurring aesthetic concepts, negative defaults, and camera keywords.
3. Automatically saves extracted facts into the SQLite Knowledge Graph without requiring manual entry.

---

## CLI Memory Management

```bash
# List all facts in a given scope
aris memory list --scope style
aris memory list --scope user

# Search memory facts using SQLite FTS5
aris memory search "cyberpunk neon"

# Manually add a knowledge fact
aris memory add --topic "style:anime" --concept "palette" --fact "Vibrant saturated pastel tones with crisp lineart" --scope style
```
