# Exploration: LoRA & ControlNet Manager

## 1. Domain Model Update
The `domain.ImageSpec` needs to be extended to support advanced generation features.

### `internal/core/domain/types.go`
*   **New Structs**:
    *   `LoRAConfig`: `Name string`, `Scale float64`
    *   `ControlNetConfig`: `Type string` (canny, openpose, etc.), `Strength float64`, `InputPath string`
*   **Update `ImageSpec`**:
    *   Add `LoRAs []LoRAConfig`
    *   Add `ControlNets []ControlNetConfig`
*   **Helper Methods**:
    *   `HasLoRA() bool`
    *   `HasControlNet() bool`
    *   `ApplyDefaults()`: Validate LoRA scales (default 0.75 or 1.0) and ControlNet strengths (default 0.7 or 1.0).

## 2. CLI Updates
*   **Prompt Parsing**: Add regex/parsing logic in `internal/adapters/ui/cli/cli.go` to extract `<lora:name:scale>` from raw prompts.
*   **Command Line Flags**:
    *   `aris gen --lora "name:scale,name2:scale2"`
    *   `aris gen --controlnet "type:strength:path"`
*   **Subcommands**: 
    *   `aris lora list` (if supported via backend)
    *   `aris controlnet list` (if supported)

## 3. ComfyUI Integration (`internal/adapters/image/comfyui.go`)
The `buildComfyGraph` function needs to be refactored to support dynamic node chaining.

*   **LoRA Chaining**:
    *   Create a `LoraLoader` node for each LoRA in `ImageSpec.LoRAs`.
    *   Chain the model input through sequential `LoraLoader` nodes.
    *   Connect the final LoRA model output to the `KSampler`.
*   **ControlNet Chaining**:
    *   Pre-process images using `pkg/imgutil/controlnet.go` if necessary (e.g., Canny edges).
    *   Create a `ControlNetLoader` node for the specified type (e.g., `control_v11p_sd15_canny.pth` or Flux-equivalent).
    *   Create an `ApplyControlNet` node connecting the Conditioning output and the processed ControlNet image.

## 4. Backend-Specific Implementation
*   **Fal.ai**: Update `falai.go` to map `LoRAs` and `ControlNets` to their specific API payloads. Flux LoRA payloads: `loras: [{ path, scale }]`.
*   **Pollinations**: Minimal support (mostly standard prompting) or specific fallback if API supports LoRAs (usually via prompt injection).

## 5. Next Steps
1.  **Refactor Domain**: Update `internal/core/domain/types.go`.
2.  **CLI Updates**: Implement parsing logic in `internal/adapters/ui/cli/`.
3.  **Backend Refactor**: Update `internal/adapters/image/comfyui.go` to handle arbitrary node chaining for LoRAs and ControlNets.
