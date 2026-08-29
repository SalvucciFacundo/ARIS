# Design: Multi-Backend Architecture & Autonomous Agent Pipeline

## 1. Hexagonal Structure & Backend Registry

```
                    ┌────────────────────────┐
                    │      AgentService      │
                    └───────────┬────────────┘
                                │
                   ┌────────────▼────────────┐
                   │     BackendRegistry     │
                   └────────────┬────────────┘
                                │
         ┌───────────────┬──────┴────────┬───────────────┐
         ▼               ▼               ▼               ▼
┌─────────────────┐ ┌─────────┐ ┌─────────────────┐ ┌─────────┐
│  Pollinations   │ │ ComfyUI │ │  Fal.ai / Repl  │ │ OpenAI  │
│  (Free Cloud)   │ │ (Local) │ │  (Cloud Managed)│ │(DALL-E) │
└─────────────────┘ └─────────┘ └─────────────────┘ └─────────┘
```

### Backend Registry Interface
```go
package ports

type BackendRegistry interface {
    Register(backend ImageBackend) error
    Get(name string) (ImageBackend, error)
    List() []string
    GetDefault() ImageBackend
}
```

## 2. ComfyUI Local Client Protocol
ComfyUI requires a WebSocket connection to track generation progress and a REST POST request with the node graph:
1. Generate `clientId` (UUID).
2. Connect to `ws://127.0.0.1:8188/ws?clientId={clientId}` in a background goroutine.
3. Construct prompt JSON (KSampler, CheckpointLoader, CLIPTextEncode, EmptyLatentImage, SaveImage).
4. POST `/prompt` with `{ "prompt": graph, "client_id": clientId }`.
5. Await WebSocket message `executed` with output image filenames.
6. GET `/view?filename={filename}&subfolder={subfolder}&type=output` to download bytes.

## 3. Fal.ai / Replicate Cloud Polling Protocol
1. POST `/queue` or `/predictions` with prompt, width, height, seed, and model.
2. Receive request ID / prediction ID.
3. Poll status every 500ms with exponential backoff until `status: completed`.
4. Download image from CDN URL to local output cache.
