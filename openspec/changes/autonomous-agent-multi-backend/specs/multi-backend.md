# Specification: Multi-Backend Image Generation Engine

## Requirements

### REQ-MB-1: Backend Registry & Discovery
- The system must maintain a thread-safe `BackendRegistry` where image backends register themselves by name.
- Users must be able to list available backends and their supported models via CLI (`aris backends list`).
- If a requested backend requires an API key that is not configured, the system must return a clear actionable error message.

### REQ-MB-2: Backend Implementations

#### 1. Pollinations Backend (`pollinations`)
- Type: Cloud API (Zero-config, free).
- Supported Models: `flux`, `flux-realism`, `flux-cablyai`, `flux-anime`, `flux-3d`, `turbo`.
- Protocol: HTTP GET `https://image.pollinations.ai/prompt/{prompt}?width=W&height=H&seed=S&model=M&nologo=true&negative=N`.

#### 2. ComfyUI Local Backend (`comfyui`)
- Type: Local Hardware (GPU).
- Default Host: `http://127.0.0.1:8188`.
- Protocol:
  - WebSocket connection for execution progress & status events (`/ws?clientId=...`).
  - REST POST `/prompt` with prompt graph JSON.
  - REST GET `/view?filename=...` to fetch generated image bytes.
- Workflow: Supports customizable SDXL and Flux node graph templates.

#### 3. Fal.ai Cloud Backend (`falai`)
- Type: Cloud Managed API.
- Supported Models: `fal-ai/flux-pro/v1.1`, `fal-ai/flux/dev`, `fal-ai/flux/schnell`, `fal-ai/flux-realism`.
- Auth: `FAL_KEY` or `config.yaml` `image.fal_key`.
- Protocol: REST POST `https://queue.fal.run/{model}` with polling for result image URL.

#### 4. Replicate Cloud Backend (`replicate`)
- Type: Cloud Managed API.
- Supported Models: `black-forest-labs/flux-schnell`, `black-forest-labs/flux-dev`, `stability-ai/sdxl`.
- Auth: `REPLICATE_API_TOKEN`.
- Protocol: REST POST `https://api.replicate.com/v1/predictions` with polling.

#### 5. OpenAI DALL-E Backend (`openai`)
- Type: Cloud Managed API.
- Supported Models: `dall-e-3`, `dall-e-2`.
- Auth: `OPENAI_API_KEY`.
- Protocol: REST POST `https://api.openai.com/v1/images/generations`.

#### 6. HuggingFace Backend (`huggingface`)
- Type: Cloud Inference API.
- Supported Models: `stabilityai/stable-diffusion-3.5-large`, `black-forest-labs/FLUX.1-dev`.
- Auth: `HF_TOKEN` (optional for public models).

### REQ-MB-3: Standardized Image Result Contract
Every backend must return a unified `domain.ImageResult`:
- `LocalPath`: Saved location on disk (`~/.aris/outputs/YYYY-MM-DD/aris_*.jpg`).
- `Duration`: Milliseconds elapsed.
- `SizeInBytes`: Image size.
- `Format`: `jpg`, `png`, or `webp`.
- `Metadata`: Backend-specific details (seed, model, steps, api latency).
