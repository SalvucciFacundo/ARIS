# Specification: Specialized Visual Subagent System

## Requirements

### REQ-SUB-1: Pre-Configured Core Subagents
The system must bootstrap 5 core visual specialists:
1. **Director (`director`)**:
   - System Prompt: Visual storytelling, photographic lighting, cinematic lenses, artistic composition.
   - Temperature: `0.8` (High creativity).
2. **PromptSmith (`promptsmith`)**:
   - System Prompt: Model-specific syntax compiler (Flux, SDXL, DALL-E, LoRAs, samplers, CFG).
   - Temperature: `0.3` (Precise, deterministic).
3. **Critic (`critic`)**:
   - System Prompt: Multi-modal visual quality assurance, anatomy defect detection, prompt adherence scoring.
   - Temperature: `0.1` (Strict, analytical).
4. **Curator (`curator`)**:
   - System Prompt: Aesthetic Knowledge Graph archivist, artist catalog search, negative prompt librarian.
   - Temperature: `0.4`.
5. **Enhancer (`enhancer`)**:
   - System Prompt: Image super-resolution, upscale factor calculator, color grading, face repair.
   - Temperature: `0.2`.

### REQ-SUB-2: Direct `@name` Routing
- If a user input starts with `@<name>` (e.g. `@director`, `@promptsmith`, `@curator`), the message is routed directly to that subagent's isolated context.
- The subagent responds with its specialized domain perspective and personality.

### REQ-SUB-3: Subagent Pipeline Execution
When running full autonomous generation:
1. Orchestrator -> `@director`: Conceptualizes scene, lighting, camera.
2. `@director` -> `@promptsmith`: Compiles into technical `ImageSpec`.
3. `@promptsmith` -> `ImageBackend`: Synthesizes image.
4. `ImageBackend` -> `@critic`: Inspects rendered output.
5. `@critic` -> `@enhancer`: Recommends post-processing / upscaling if needed.
6. Orchestrator -> `@curator`: Saves successful recipes and learned facts.

### REQ-SUB-4: SQLite Storage & Dynamic Subagents
- `subagent_defs` SQLite table storing `name`, `description`, `system_prompt`, `personality`, `temperature`, `model`, `allowed_tools`, `created_at`, `updated_at`.
- Users can add or customize subagents via CLI.
