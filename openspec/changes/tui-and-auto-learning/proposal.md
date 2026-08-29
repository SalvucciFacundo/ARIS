# Proposal: Interactive Cyberpunk TUI & Autonomous Learning Loop

## 1. Problem Statement
1. Currently, ARIS only has a headless single-turn CLI (`aris gen "..."`). Users generating creative imagery need an **interactive, conversational experience** where they can refine images iteratively in real-time, view Art Director reasoning live, adjust generation parameters on the fly, and inspect the rendered image in their terminal.
2. The agent's Knowledge Graph currently requires manual entry (`aris memory add`) or static rule matching. To truly behave like **Hermes and GAIA**, ARIS needs an **autonomous learning loop** (`AutoLearner`) that actively learns user stylistic preferences, recurring themes, negative triggers, and prompt corrections without requiring manual maintenance.

## 2. Proposed Solution
1. **Interactive Cyberpunk TUI (`aris chat`)**:
   - Built with `Bubbletea`, `Lipgloss`, and `Bubbles`.
   - **Split Screen Design**:
     - *Left Pane*: Live conversational chat stream, Art Director reasoning, step-by-step dispatch status.
     - *Right Pane*: Visual generation parameters (Backend selector, Model, Aspect Ratio, Seed, CFG, Steps), Metadata inspector, and Terminal Image Preview (ANSI halfblocks + OS launcher).
   - **Interactive Prompt Refinement**: Type follow-up instructions (*"make the lighting more dramatic"*, *"remove the background buildings"*) to seamlessly trigger img2img / delta prompt updates.
2. **Auto-Learning Loop (`AutoLearner`)**:
   - Post-generation reflection engine inspired by Hermes and GAIA.
   - Analyzes conversational deltas and user feedback (e.g. *"hacela más oscura"*, *"siempre uso 16:9"*, *"me encanta el estilo synthwave"*).
   - Automatically distills and saves structured `KnowledgeFact` entries into SQLite (User, Style, Project scopes) tagged with `source_agent: "aris:autolearn"`.
   - Incorporates learned facts into future prompt synthesis automatically.

## 3. Scope Boundaries
- **In Scope**:
  - `AutoLearner` engine & prompt reflection service.
  - Interactive Bubbletea TUI model, views, update loop, and keybindings.
  - Terminal image preview rendering (ANSI halfblock color rasterizer & fallback image viewer launcher).
  - Integration with `AgentService`, `BackendRegistry`, and `KnowledgeGraphStore`.
  - Comprehensive unit tests.
- **Out of Scope**:
  - Full Wails v2 desktop webview (scheduled for Phase 4).
  - Remote Discord/Telegram bots.
