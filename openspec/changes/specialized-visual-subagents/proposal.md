# Proposal: Specialized Visual Subagent System

## 1. Problem Statement
In traditional image generation systems, a single monolithic prompt is used to handle everything: artistic direction, technical model syntax, quality inspection, and memory curation. This leads to compromised quality because an LLM cannot be simultaneously an unconstrained creative Art Director and a rigid model-specific syntax compiler without domain interference.

## 2. Proposed Solution
Implement a **Multi-Agent Visual Subagent System** in Go (inspired by GAIA's architecture):
1. **5 Specialized Visual Subagents**:
   - **`@director` (Art Director)**: Specializes in visual storytelling, composition, camera optics, lighting setups, and color harmony.
   - **`@promptsmith` (Technical Compiler)**: Translates conceptual art direction into model-specific syntax (Flux prose, SDXL weighted tags, ComfyUI node triggers, negative embeddings).
   - **`@critic` (VLM Quality Auditor)**: Multi-modal visual inspection of rendered images for anatomy, artifacts, and constraint adherence.
   - **`@curator` (Aesthetic Memory Curator)**: Manages style catalogs, artist biographies/styles, and user aesthetic profiles in the Knowledge Graph.
   - **`@enhancer` (Post-Processing & Super-Resolution)**: Manages super-resolution, face restoration, and image format optimization.
2. **Autonomous Pipeline Delegation & `@name` Direct Chat**:
   - The user can talk directly to any specialist in CLI/TUI: `@director conceptualize a sci-fi cyberpunk noir street` or `@promptsmith convert this concept to Flux Dev syntax`.
   - In full autonomous mode (`aris gen`), the orchestrator automatically cascades tasks through the subagent pipeline.
3. **Dynamic Subagent Definitions in SQLite**:
   - Subagents are stored in SQLite `subagent_defs` with customizable system prompts, personalities, and model parameters.

## 3. Scope Boundaries
- **In Scope**:
  - `SubagentDef` domain models and `SubagentRegistry` ports.
  - SQLite persistence for dynamic subagents (`internal/adapters/db/subagent_defs.go`).
  - `SubagentManager` service orchestrating direct `@name` routing and pipeline execution.
  - CLI `aris subagents list` / `aris subagents show <name>` and TUI `@name` mention support.
  - Full unit test coverage.
- **Out of Scope**:
  - Distributed network RPC subagents (runs within the single Go binary).
