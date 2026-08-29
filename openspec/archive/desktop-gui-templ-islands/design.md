# Architecture Design: ARIS Desktop GUI & Web Interface

## 1. System Overview
The ARIS GUI subsystem introduces a unified visual interface built on Go-native web technologies (Templ, HTMX, Tailwind) integrated into a standard Hexagonal Architecture. It supports running as a standalone HTTP server (headless VPS mode) or encapsulated within a native desktop window (desktop mode with remote client capabilities).

## 2. Directory Layout and Hexagonal Adapters

### `internal/adapters/ui/web/`
Handles the HTTP interface, routing, and real-time event streaming.
*   `server.go`: Orchestrates the HTTP `http.Server`, graceful shutdown context, and route registration.
*   `router.go`: Standard `net/http` ServeMux wiring API endpoints to handlers.
*   `auth.go`: Middleware for verifying Bearer tokens, injecting `Token` into context, and skipping validation for local loopback connections.
*   `handlers.go`: Maps REST routes to Application/Domain ports (e.g., `HandleGenerate`, `HandleInpaint`, `HandleHistory`).
*   `sse.go`: Pub/sub event broker mapping core domain events to connected web clients via Server-Sent Events.

### `internal/adapters/ui/web/components/`
Contains type-safe Go Templ components for the view layer.
*   `layout.templ`: HTML5 shell, header, and overall grid container.
*   `gallery.templ`: Left panel, handles image thumbnail iterations.
*   `canvas.templ`: Center panel, anchors the HTML5 canvas island.
*   `chat.templ`: Right panel, streaming thought blocks and message layout.
*   `inspector.templ` / `controls.templ`: Settings, aspect ratio toggles, generation parameters.

### `internal/adapters/ui/web/static/`
Embedded assets using `//go:embed`.
*   `embed.go`: `embed.FS` declaration for `dist/`.
*   `dist/app.css`: Pre-compiled Tailwind classes.
*   `dist/htmx.min.js`: Client-side AJAX/DOM swapping.
*   `dist/islands.js`: Client-side logic for the HTML5 canvas and masking.

### `internal/adapters/ui/desktop/`
Handles desktop integration and process lifecycle.
*   `runner.go`: Instantiates the native window (e.g., via `webview/webview` or `lorca`).
*   `fallback.go`: Detects windowing driver failure and falls back to opening the system's default browser (via `exec.Command`).

### `cmd/aris/` CLI Registration
*   `serve.go`: Bootstraps `ui.web` server (aliases: `aris ui`).
*   `gui.go`: Bootstraps `ui.desktop` runner (aliases: `aris desktop`).

## 3. Real-Time Event Architecture (SSE Broker)

The SSE broker follows a thread-safe publish-subscribe pattern allowing the core engine to emit domain events that are propagated to HTTP clients.

```text
+----------------+       +-------------------+       +--------------------+
|                |       |                   |       |                    |
|  Core Engine   +------>+    SSE Broker     +------>+ Connected Web      |
|  (Events)      |       |  (Pub/Sub Hub)    |       | Clients            |
|                |       |                   |       | (Browser/Webview)  |
+----------------+       +---------+---------+       +--------------------+
                                   |
                                   |
                          map[chan[]byte]bool
```

**Broker Implementation:**
*   `Register(client chan []byte)`: Subscribes a new HTTP request context to the stream.
*   `Unregister(client chan []byte)`: Removes a closed client safely.
*   `Broadcast(event struct)`: Marshals a domain event to JSON and dispatches it non-blocking to all active channels. A 15-second heartbeat prevents proxy timeouts.

## 4. Templ & Templ Islands Integration

We use Templ to strongly type the HTML generation. HTMX handles AJAX requests for component swapping, but state-heavy UI elements (Canvas) use Templ Islands to encapsulate client-side JS.
*   **Component Contracts**: Components accept domain structs (e.g., `GenerationHistory`, `JobStatus`) as arguments.
*   **JSON Hydration**: To pass complex states to islands, initial states are embedded as JSON payloads in `<script type="application/json">` tags, which the `islands.js` runtime unmarshals on mount.
*   **Optimistic UI Updates**: HTMX triggers (`hx-post`, `hx-swap`) will optimistically update local panels while relying on SSE for authoritative state.

## 5. Interactive Canvas & Inpainting

The canvas operates completely on the client (`islands.js`) before communicating with the server.
1.  **Loader**: Implements HTML5 Drag-and-Drop API to load base images via `FileReader`, drawing them to a `<canvas>`.
2.  **Brush Controls**: Tracks mouse coordinates to paint a semi-transparent stroke. Uses a hidden `OffscreenCanvas` to maintain a binary (pure black/white) representation of the mask to prevent blending artifacts.
3.  **Base Export**: When dispatching `/api/inpaint`, the island serializes both the visible base and the hidden mask to `base64` PNGs and POSTs them via `FormData` to the handler.

## 6. ASCII Sequence Diagrams & Component Flow

**Generation Request & Event Stream**

```text
User (Browser)          Web Handler          SSE Broker          Core Engine
      |                      |                    |                   |
      | POST /api/generate   |                    |                   |
      |--------------------->|                    |                   |
      |                      | Submit Job         |                   |
      |                      |--------------------------------------->|
      |   202 Accepted       |                    |                   |
      |<---------------------|                    |                   |
      |                      |                    |                   |
      | GET /api/events      |                    |                   |
      |--------------------->|                    |                   |
      |                      | Register()         |                   |
      |                      |------------------->|                   |
      |                      |                    | Event: Progress   |
      |                      |                    |<------------------|
      | SSE: progress        |                    |                   |
      |<------------------------------------------|                   |
      |                      |                    | Event: ImageReady |
      |                      |                    |<------------------|
      | SSE: image_ready     |                    |                   |
      |<------------------------------------------|                   |
```

## 7. Testing Strategy

1.  **Unit Tests (Handlers & Router)**: Use `net/http/httptest.ResponseRecorder` to validate JSON endpoints, token middleware block/allow behavior, and form parsing without real network overhead.
2.  **SSE Race Tests**: Spin up concurrent mock clients subscribing and unregistering while a mock engine aggressively broadcasts events to ensure the pub/sub map has no race conditions. Run `go test -race`.
3.  **Mock Engine Adapter**: Replace the core engine with a stub that instantly returns fixed image paths to validate the complete UI request lifecycle.
4.  **Integration Test Suite**: Test the fallback boot process: verify that when window drivers fail, the local server still starts and `exec.Command` for the OS browser is correctly invoked.
