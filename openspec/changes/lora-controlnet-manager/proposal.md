# Proposal: LoRA & ControlNet Manager

## Intent
Introduce LoRA weight stacking and ControlNet structural conditioning into ARIS domain entities, image backend adapters, image processing preprocessors, and the CLI interface.

## Business Problem
Standard diffusion models lack fine-grained stylistic control (such as custom characters or specific artistic styles) and geometric/structural guidance (such as character poses, edge definitions, and depth perspective). Without these capabilities, users cannot enforce strict composition or replicate specialized aesthetics consistently.

## Current State Gap
Currently, the ARIS CLI and image generation pipelines only support basic text-to-image (and potentially basic image-to-image) workflows. The domain model, prompt parser, and backend integration layers are missing the vocabulary and graph generation logic needed to compose multiple stylistic embeddings (LoRAs) and spatial conditions (ControlNets).

## Proposed Solution
We propose extending the core generation pipeline to support these advanced features natively across all supported backends (ComfyUI and Fal.ai Flux endpoints):

1. **Domain Entities Update**: Extend `domain.ImageSpec` to accept arrays of `LoRAConfig` and `ControlNetConfig` to accurately represent weight stacking and multiple structural controls.
2. **LoRA Weight Stacking**: Support scale bounds `[0.0 - 2.0]` and introduce prompt inline syntax `<lora:name:scale>` for intuitive user input, directly mapped to the CLI layer.
3. **ControlNet Conditioning**: Support various structural models (`canny`, `openpose`, `depth`, `lineart`) with variable strength controls.
4. **Image Processing (Preprocessor)**: Build a pure-Go Canny edge detection preprocessor inside `pkg/imgutil/controlnet.go` to prepare images for Canny ControlNet pipelines without relying on external python scripts or heavy dependencies.
5. **Dynamic Backend Adapters**: 
   - Refactor the ComfyUI integration (`internal/adapters/image/comfyui.go`) to support dynamic node chaining (dynamically wiring sequential `LoraLoader` nodes and `ApplyControlNet` nodes).
   - Update Fal.ai adapters (`falai.go`) to map the new domain structures into Flux LoRA/ControlNet payloads.
6. **CLI Enhancements**: Introduce dedicated flags (`--lora`, `--controlnet`) and subcommands (`aris lora`, `aris controlnet`) for management and discovery, plus the aforementioned regex-based inline prompt parsing.

## Target Users
Power users, technical artists, and automation scripts using ARIS that require reproducible, composition-aware, and stylistically precise image generation capabilities.

## Risks & Implications
- **Backend Compatibility**: Not all backends support all ControlNet types natively (e.g., Pollinations). We must implement graceful fallbacks or explicit errors.
- **Node Graph Complexity**: Dynamically chaining nodes in ComfyUI requires strict input/output matching to avoid graph execution failures.
- **Performance Overhead**: Edge detection (Canny) in pure Go might have performance implications on large images if not optimized properly.

## Scope Boundaries & Non-Goals
- **Model Training**: This proposal strictly covers the *usage* (inference) of LoRAs and ControlNet models. Training or fine-tuning models is explicitly out of scope.
- **Full GUI Support**: This effort targets the CLI, core domain, and backends. Any Graphical User Interface beyond CLI terminal output is out of scope.
- **Automatic Model Downloading for ComfyUI**: We assume the required LoRA `.safetensors` and ControlNet models are already present in the user's ComfyUI instance.

## Success Criteria
- Users can pass `<lora:my_character:0.8>` in a prompt or via `--lora "my_character:0.8"` and successfully see the stylistic change through ComfyUI and Fal.ai.
- Users can pass `--controlnet "canny:0.75:/path/to/image.png"` and ARIS processes the image through the Go-based Canny preprocessor and passes it to the backend.
- The ComfyUI adapter can dynamically generate node graphs for an arbitrary number of LoRAs chained together.
- All new CLI parser and preprocessor logic is backed by robust unit tests (Strict TDD applied).
