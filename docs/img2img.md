# ARIS Img2Img & Visual Reference Pipeline

The **ARIS Img2Img & Visual Reference Pipeline** extends the ARIS generative system to support image-to-image transformations (img2img), style reference transfers, and masked inpainting workflows across multiple backends (**Fal.ai**, **ComfyUI**, **OpenAI**, and **Pollinations**).

---

## Key Features

1. **Local & Remote Reference Inputs**:
   - Accepts local file paths (PNG, JPEG, WEBP) or HTTP/HTTPS URLs.
   - Enforces format validation, dimension detection, and payload bounds ($\le 25\text{ MB}$).
   - Automatically encodes local inputs to Base64 Data URIs for remote APIs.

2. **Masked Inpainting & Dimensional Safety**:
   - Validates mask file existence, format, and dimension matching against the base reference image.
   - Prevents invalid dimension executions locally before sending API payloads.

3. **Denoise Strength Parameter**:
   - Governs degree of divergence from source images in the range $[0.0, 1.0]$.
   - Defaults to `0.70` for img2img and inpaint workflows; easily adjusted via `--strength` / `-s`.

4. **Multi-Backend Execution Adapters**:
   - **Fal.ai**: Routes to `fal-ai/flux/dev/image-to-image` and `fal-ai/flux-general/inpainting` using `image_url` and `mask_url`.
   - **ComfyUI**: Uploads image and mask buffers via `/upload/image`, builds dynamic `LoadImage`, `VAEEncodeForInpaint`, and `KSampler` node graphs.
   - **OpenAI**: Encodes multipart/form-data for `POST /v1/images/edits` with DALL-E 2.
   - **Pollinations**: Supports URL image reference query parameter or cleanly rejects inpainting with actionable capability errors.

5. **Visual Subagent Routing**:
   - `@inpainter`: Visual Inpainting Specialist optimizing seamless background blending and edge feathering.
   - `@restyler`: Style Transfer Specialist for medium transformations and palette migration with a default `0.65` denoise strength.

6. **CLI Command `aris edit`**:
   - Dedicated subcommand `aris edit <image_path> "<prompt>" [options]`.

---

## CLI Usage Guide

### 1. Basic Image-to-Image Transformation
Re-render or transform an existing image into a new style or concept:

```bash
aris edit input.png "cyberpunk neon overhaul, volumetric rain reflections" --strength 0.65 --backend falai
```

### 2. Masked Inpainting
Replace or edit a specific region defined by a black-and-white or alpha mask:

```bash
aris edit portrait.png "remove glasses and add cybernetic eye implant" --mask mask.png --backend comfyui --strength 0.85
```

### 3. Remote URL Reference
Use a remote image as a reference:

```bash
aris edit https://example.com/character.jpg "oil painting masterpiece in the style of Rembrandt" --backend falai
```

### 4. CLI Flags & Options

| Flag | Shorthand | Description | Default |
| :--- | :--- | :--- | :--- |
| `--mask <path>` | — | Local path to mask image (for inpainting) | `""` |
| `--strength <float>` | `-s` | Denoise strength $[0.0, 1.0]$ | `0.70` |
| `--backend <name>` | `-b` | Backend provider (`falai`, `comfyui`, `openai`, `pollinations`) | `pollinations` |
| `--model <name>` | `-m` | Specific model override | Backend default |
| `--ratio <string>` | `-r` | Aspect ratio (`1:1`, `16:9`, `9:16`, `4:3`, `3:4`, `21:9`) | `1:1` |
| `--negative <string>` | `-n` | Negative prompt keywords | `""` |
| `--critic` | — | Run VLM visual critique on output | `false` |
| `--auto-heal` | — | Automatically retry and adjust prompt on low critique score | `false` |

---

## Backend Details & Capabilities

| Backend | Img2Img Supported | Inpainting Supported | Notes |
| :--- | :---: | :---: | :--- |
| **Fal.ai** (`falai`) | ✅ Yes | ✅ Yes | Uses Flux Dev img2img and Flux General Inpaint endpoints |
| **ComfyUI** (`comfyui`) | ✅ Yes | ✅ Yes | Full local execution, uploads images to `/upload/image` |
| **OpenAI** (`openai`) | ✅ Yes | ✅ Yes | Uses `POST /v1/images/edits` (DALL-E 2) |
| **Pollinations** (`pollinations`) | ✅ URL only | ❌ No | Rejects inpainting with clean capability error |

---

## Visual Subagents

Invoke specialized visual subagents directly from chat, gateway, or CLI:

```bash
# Style transfer subagent
aris gen "@restyler ukiyo-e woodblock print of a modern skyscraper"

# Inpainting specialist subagent
aris gen "@inpainter seamless background replacement with futuristic skyline"
```
