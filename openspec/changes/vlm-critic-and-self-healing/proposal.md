# Proposal: VLM Vision Critic & Autonomous Self-Healing Loop

## 1. Problem Statement
Generative diffusion models (such as Flux or SDXL) can occasionally fail user constraints:
- Missing requested objects (e.g. asking for "a cat with sunglasses" and rendering a cat without sunglasses).
- Anatomical artifacts (e.g. distorted limbs or blurry faces).
- Style mismatches or unwanted text/watermarks.

Without visual inspection, an agent blindly delivers flawed images to the user.

## 2. Proposed Solution
Implement an autonomous **VLM (Vision Language Model) Critic & Self-Healing Loop**:
1. **Multi-Provider Vision Adapter (`internal/adapters/vision`)**:
   - **Local Vision (Ollama)**: `qwen2.5-vl`, `granite-vision`, `llava`, `minicpm-v`.
   - **Cloud Vision (OpenAI / OpenRouter / Anthropic)**: `gpt-4o-mini`, `gpt-4o`, `claude-3-5-sonnet`.
   - **Mock / Heuristic Fallback**: Zero-token offline critic.
2. **Evaluation Protocol**:
   - Encodes rendered image and sends alongside prompt specifications to the vision model.
   - Evaluates: (a) Prompt adherence, (b) Visual quality & clarity, (c) Artifacts/text errors.
   - Outputs a normalized quality score (0.0 to 1.0) and actionable critique notes.
3. **Autonomous Self-Healing Loop**:
   - If quality score < threshold (default 0.6) and auto-heal is enabled, ARIS automatically applies targeted prompt adjustments and re-rolls once with a fresh seed.
4. **Memory Integration**:
   - Feeds critique discoveries back into the Knowledge Graph to prevent recurring errors.

## 3. Scope Boundaries
- **In Scope**:
  - `VisionCritic` adapter supporting OpenAI/Ollama vision protocols.
  - Image base64 encoding and payload assembly.
  - `CriticService` with automated self-healing retry logic.
  - CLI flags: `--critic` / `--auto-heal`.
  - Full unit tests with mock vision servers.
- **Out of Scope**:
  - Heavy local PyTorch/Python direct bindings (uses Ollama / REST API endpoints instead).
