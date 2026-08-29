# Specification: ARIS Desktop GUI & Web Interface (Templ + Islands + HTMX + Tailwind)

## Purpose

Define the functional and architectural specifications for the ARIS Web and Native Desktop graphical user interface subsystem. The subsystem provides an interactive 3-panel workspace (Visual Gallery, Inpainting/Reference Canvas, and Conversational Chat with Parameter Controls) packaged into a single Go binary, accessible locally via a native desktop window, in the browser, or remotely hosted on a VPS with token authentication.

---

## Requirements

### Requirement: REQ-WEB-1 - Web Server Execution, Routing & Token Authentication

The system MUST provide an HTTP web server adapter (`internal/adapters/ui/web`) executable via `aris serve` (alias `aris ui`) that serves HTML rendered with Templ components, static embedded assets, and REST/SSE API endpoints.

#### Scenario: Default Local Web Server Startup
- GIVEN a user executes `aris serve` without flags or environment overrides
- WHEN the server starts
- THEN it MUST bind to `127.0.0.1:8080` (or the next available port if 8080 is occupied)
- AND it MUST bypass authentication for connections from `127.0.0.1` and `::1`
- AND it MUST output the listening URL to stdout.

#### Scenario: Custom Host and Port Binding
- GIVEN a user runs `aris serve --host 0.0.0.0 --port 9090`
- WHEN the server initializes
- THEN it MUST listen on `0.0.0.0:9090`
- AND accept incoming HTTP connections on all network interfaces.

#### Scenario: Token Authentication via Environment Variable or CLI Flag
- GIVEN `ARIS_WEB_TOKEN="aris-secret-xyz"` is set in the environment or passed via `--token aris-secret-xyz`
- WHEN an HTTP request is made to any endpoint other than `/assets/*`
- THEN the server MUST require an `Authorization: Bearer aris-secret-xyz` header OR a `?token=aris-secret-xyz` query parameter
- AND if the token is missing or invalid, the server MUST return HTTP `401 Unauthorized` with JSON error `{"error": "Unauthorized: invalid or missing access token"}`.

#### Scenario: Core Web Route Registration
- GIVEN the web server is running
- WHEN clients make requests to the server
- THEN the server MUST handle the following route table:
  - `GET /` -> Render main 3-panel application shell (Templ layout)
  - `GET /api/events` -> Server-Sent Events (SSE) stream for progress, reasoning, and image updates
  - `POST /api/generate` -> Submit text-to-image or subagent generation job
  - `POST /api/inpaint` -> Submit image-to-image or inpainting job with base image and mask
  - `GET /api/history` -> Retrieve chronological list of generated image metadata
  - `GET /api/image/{id}` -> Serve the rendered image file with appropriate MIME type
  - `GET /api/subagents` -> List registered subagents, descriptions, and trigger patterns
  - `GET /api/backends` -> List registered backends and their supported models
  - `GET /assets/*` -> Serve embedded static assets (CSS, JS, icons) with client caching headers.

---

### Requirement: REQ-WEB-2 - Desktop Window Runner & Remote VPS Client Connection

The system MUST provide a desktop native window launcher (`aris gui` / `aris desktop`) that opens a native webview window connected either to a local in-process engine or to a remote ARIS VPS server.

#### Scenario: Local Desktop Mode Launch
- GIVEN a user runs `aris gui` without remote flags
- WHEN the command executes
- THEN it MUST start the internal web server in-process on an ephemeral or default localhost port
- AND it MUST instantiate a native OS window (via WebKit2GTK on Linux, Cocoa WebKit on macOS, WebView2 on Windows) pointing to `http://localhost:<port>`
- AND closing the window MUST trigger a graceful shutdown of the background web server and exit the process cleanly.

#### Scenario: Remote VPS Mode Client Connection
- GIVEN a remote ARIS instance running at `https://vps.aris.ai:8080` with token `sec-vps-99`
- WHEN a user runs `aris gui --remote https://vps.aris.ai:8080 --token sec-vps-99`
- THEN the desktop runner MUST NOT start a local generation engine or web server
- AND it MUST open the native webview window directly pointing to `https://vps.aris.ai:8080?token=sec-vps-99`
- AND it MUST attach the token to the session context for all subsequent AJAX and SSE requests.

#### Scenario: Webview Missing Driver Graceful Fallback
- GIVEN a host environment where native GUI webview libraries (e.g. WebKit2GTK / WebView2) are unavailable or fail initialization
- WHEN `aris gui` is executed
- THEN the system MUST NOT crash with an unhandled panic
- AND it MUST log a warning: `"Native webview unavailable; falling back to default web browser."`
- AND it MUST start the local web server and open the user's default system browser using the OS browser opener (`xdg-open` on Linux, `open` on macOS, `start` on Windows).

---

### Requirement: REQ-WEB-3 - 3-Panel Layout & Templ Component Hierarchy

The web interface MUST render a responsive 3-panel layout structured using type-safe Go Templ components and Templ Islands for interactive zones.

#### Scenario: Initial Page Load and Shell Structure
- GIVEN a browser client navigates to `GET /`
- WHEN the template renders
- THEN the response MUST include a valid HTML5 document containing:
  - Header: System status indicator, active backend badge, connected model name, and theme toggle.
  - Left Panel (`#panel-gallery`): Visual gallery thumbnail grid, search filter, and full-resolution lightbox trigger.
  - Center Panel (`#panel-canvas`): Interactive canvas with drag-and-drop dropzone, aspect ratio overlay, and inpainting mask brush tool.
  - Right Panel (`#panel-chat`): Conversational chat feed, streaming reasoning accordion, `@subagent` selector badges, and generation parameter knobs.

#### Scenario: Panel Toggle and Responsive Collapse
- GIVEN the application is viewed on a screen with viewport width `< 1024px` (tablet/mobile)
- WHEN the user toggles panel visibility
- THEN the interface MUST allow collapsing the Left and Right panels into off-canvas drawers while keeping the Center Canvas or Chat visible.

---

### Requirement: REQ-WEB-4 - Real-Time SSE Stream (`/api/events`) for Generation & Critic

The web server MUST stream live pipeline events to connected clients over a persistent HTTP Server-Sent Events (SSE) connection at `/api/events`.

#### Scenario: SSE Stream Subscription & Keep-Alive
- GIVEN an authenticated client connects to `GET /api/events`
- WHEN the connection is established
- THEN the server MUST set headers `Content-Type: text/event-stream`, `Cache-Control: no-cache`, and `Connection: keep-alive`
- AND the server MUST send a periodic `:ping\n\n` heartbeat comment every 15 seconds to prevent network timeouts.

#### Scenario: Streaming Generation Progress Events
- GIVEN an active image generation job with ID `job-456`
- WHEN the backend processes the generation steps
- THEN the server MUST emit an SSE event with format:
  ```
  event: progress
  data: {"job_id":"job-456","stage":"rendering","current_step":20,"total_steps":40,"percent":50,"message":"Sampling step 20/40"}
  ```
- AND the client UI MUST update the progress bar and status indicator without a page reload.

#### Scenario: Streaming Reasoning Thoughts
- GIVEN a subagent (e.g. `@director` or `@promptsmith`) is reasoning before generation
- WHEN reasoning chunks are produced
- THEN the server MUST emit:
  ```
  event: reasoning
  data: {"job_id":"job-456","subagent":"director","chunk":"Balancing volumetric lighting and color temperature..."}
  ```
- AND the client Chat panel MUST append the chunk to the active collapsible thought block in real time.

#### Scenario: Delivery of Completed Image and Critic Evaluation
- GIVEN image generation completes and the VLM Critic evaluates the output
- WHEN results are finalized
- THEN the server MUST emit:
  ```
  event: image_ready
  data: {"job_id":"job-456","image_id":"img-789","url":"/api/image/img-789","prompt":"cyberpunk city","aspect_ratio":"16:9","seed":4242}
  ```
- AND subsequent critic event:
  ```
  event: critic_evaluation
  data: {"job_id":"job-456","image_id":"img-789","score":8.7,"critique":"Excellent atmosphere and detail; high coherence.","passed":true}
  ```
- AND the client Canvas and Gallery MUST automatically display the new image and critic badge.

#### Scenario: Client Reconnection Handling
- GIVEN a temporary network disconnection occurs during generation
- WHEN the client EventSource reconnects
- THEN the client MUST send `Last-Event-ID`
- AND the server MUST replay missed state or provide the current job status.

---

### Requirement: REQ-WEB-5 - Interactive Canvas (Drag-and-Drop, Mask Brush, Aspect Ratio Guide)

The Center Panel MUST feature an interactive HTML5/JavaScript Templ Island for loading reference images, painting inpainting masks, and previewing framing guides.

#### Scenario: Reference Image Drag-and-Drop
- GIVEN the user drags an image file (PNG, JPEG, WebP) onto the Center Canvas dropzone
- WHEN the file is dropped
- THEN the canvas MUST load and render the image
- AND store the image buffer in client memory
- AND display an image loaded badge with dimensions and file size.

#### Scenario: Inpainting Mask Drawing & Eraser
- GIVEN an active base image loaded into the canvas
- WHEN the user selects the Mask Brush tool and draws over an image region
- THEN the canvas MUST render a semi-transparent mask stroke (e.g. 50% opacity magenta `#FF005580`) on an overlay layer
- AND the user MUST be able to adjust brush size between 5px and 120px
- AND the user MUST be able to switch between Brush, Eraser, and Clear Mask actions.

#### Scenario: Mask Export and Inpaint Dispatch
- GIVEN a base image and a painted mask on the canvas
- WHEN the user clicks "Inpaint Area" with prompt `"replace car with futuristic hovercraft"`
- THEN the client MUST serialize the mask as a black/white alpha PNG
- AND send a multipart `POST /api/inpaint` request containing `prompt`, `image` file, and `mask` file
- AND trigger the inpainting pipeline on the backend.

#### Scenario: Aspect Ratio Guide Framing
- GIVEN the user selects an aspect ratio in parameter controls (e.g. `16:9`, `1:1`, `9:16`, `21:9`)
- WHEN the ratio changes
- THEN the canvas MUST adjust its visual framing guide overlay box to match the exact selected proportions
- AND display resolution pixel estimates (e.g. `1344x768` for 16:9 landscape).

---

### Requirement: REQ-WEB-6 - Visual Gallery Island (History, Full-Res Zoom, Metadata Viewer)

The Left Panel MUST provide a visual history gallery of past generations with instant full-resolution preview, parameter inspection, and image reuse actions.

#### Scenario: Historical Generation Listing
- GIVEN historical generation records stored on the server
- WHEN the gallery initializes or receives an `image_ready` event
- THEN it MUST display image thumbnails in reverse-chronological order
- AND each thumbnail MUST display a hover badge indicating backend, aspect ratio, and critic score.

#### Scenario: Full-Resolution Lightbox Zoom and Pan
- GIVEN a user clicks on any gallery thumbnail
- WHEN the lightbox opens
- THEN the interface MUST display the full-resolution image in an interactive modal
- AND allow zooming (mouse wheel / pinch) and panning (click & drag)
- AND provide a "Download" button to save the raw uncompressed image file.

#### Scenario: Image Reuse and Parameter Transfer
- GIVEN an image open in the Lightbox or selected in the Gallery
- WHEN the user clicks "Use as Reference"
- THEN the image MUST be loaded into the Center Canvas as the active reference image.
- WHEN the user clicks "Copy Parameters"
- THEN the prompt, negative prompt, seed, aspect ratio, and backend MUST be copied into the Right Panel parameter controls.

---

### Requirement: REQ-WEB-7 - Conversational Chat & Parameter Controls Island

The Right Panel MUST provide a unified conversational interface for directing subagents, entering prompts, and tuning generation hyperparameters.

#### Scenario: Conversational Prompting and Multi-Line Input
- GIVEN the user focuses on the chat input textarea
- WHEN the user enters a prompt and presses `Enter` (without `Shift`)
- THEN the input MUST be submitted via `POST /api/generate`
- AND `Shift+Enter` MUST insert a newline without submitting.

#### Scenario: Subagent Routing Autocomplete & Badges
- GIVEN registered subagents `@director`, `@promptsmith`, `@anime`, `@photoreal`, and `@inpainter`
- WHEN the user types `@` into the chat input
- THEN the UI MUST display an autocomplete popup list of available subagents
- AND clicking a subagent badge (e.g. `@anime`) MUST prepend or route the prompt to that subagent.

#### Scenario: Parameter Knobs & Knobs Persistence
- GIVEN the Parameter Controls accordion
- WHEN the user adjusts settings:
  - Backend (Dropdown: `pollinations`, `comfyui`, `falai`)
  - Model (Dropdown populated dynamically per backend)
  - Aspect Ratio (Buttons: `1:1`, `16:9`, `9:16`, `4:3`, `3:2`, `21:9`)
  - Steps (Slider: 1 - 100)
  - CFG Scale (Slider: 1.0 - 20.0)
  - Seed (Input: number or random `-1`)
  - Critic Feedback (Toggle: Enabled/Disabled, Max Retries: 0-3)
- THEN all subsequent `/api/generate` and `/api/inpaint` requests MUST include these parameters
- AND parameter values MUST persist in browser `localStorage` across page reloads.

---

### Requirement: REQ-WEB-8 - Single Binary Asset Embedding & Zero-Node Tailwind Compilation

The web adapter MUST embed all HTML templates, client JavaScript islands, CSS stylesheets, and static icons into the compiled Go binary using `embed.FS`, requiring zero Node.js/NPM dependencies at application runtime.

#### Scenario: Go Binary Asset Embedding
- GIVEN the ARIS project is compiled using `go build ./cmd/aris`
- WHEN the binary is executed in a fresh container or clean OS environment without Node.js, NPM, or external web assets
- THEN all CSS styles, HTMX library scripts, Templ Island runtime scripts, and icons MUST be served directly from `//go:embed` embedded filesystem
- AND the application MUST render completely and function identically without internet access (offline capability for local backends).

#### Scenario: Static Asset Cache Headers
- GIVEN an HTTP request to `GET /assets/app.css` or `GET /assets/htmx.min.js`
- WHEN the server serves embedded assets
- THEN the response MUST include `Cache-Control: public, max-age=31536000, immutable` and `ETag` headers for optimal browser caching performance.

---

## Technical Specifications

### 1. Web Adapter Package Structure (`internal/adapters/ui/web`)

```text
internal/adapters/ui/web/
├── server.go             # HTTP server lifecycle, routing, and graceful shutdown
├── auth.go               # Bearer token and localhost bypass middleware
├── sse.go                # Server-Sent Events broker and client connection manager
├── handlers.go           # REST API request handlers (/api/generate, /api/inpaint, etc.)
├── desktop.go            # Native desktop webview runner and remote client connection
├── views/                # Templ components
│   ├── layout.templ      # Main HTML document shell, header, and script includes
│   ├── gallery.templ     # Left panel: history gallery & lightbox modal
│   ├── canvas.templ      # Center panel: HTML5 canvas, mask brush, aspect ratio guide
│   ├── chat.templ        # Right panel: chat stream, reasoning accordion, message bubbles
│   └── controls.templ    # Parameter controls: sliders, selectors, subagent badges
└── assets/               # Static embedded assets
    ├── embed.go          # //go:embed all:dist/* asset declarations
    └── dist/
        ├── app.css       # Compiled Tailwind CSS
        ├── htmx.min.js   # Embedded HTMX runtime
        └── islands.js    # Interactive canvas and gallery client scripts
```

---

### 2. API Contract & Payload Definitions

#### `POST /api/generate` Request Payload
```json
{
  "prompt": "Cyberpunk detective examining holographic clues in rain",
  "negative_prompt": "blurry, low quality, distorted hands",
  "subagent": "director",
  "backend": "pollinations",
  "model": "flux",
  "aspect_ratio": "16:9",
  "width": 1344,
  "height": 768,
  "steps": 30,
  "cfg_scale": 7.5,
  "seed": -1,
  "enable_critic": true,
  "critic_max_retries": 2
}
```

#### `POST /api/inpaint` Multipart Form Data
- `prompt`: String
- `backend`: String
- `model`: String
- `image`: File (PNG/JPEG base image)
- `mask`: File (PNG binary/alpha mask)
- `denoising_strength`: Float (0.1 - 1.0)

#### `GET /api/history` Response Payload
```json
[
  {
    "id": "img-20260829-001",
    "prompt": "Cyberpunk detective examining holographic clues in rain",
    "backend": "pollinations",
    "model": "flux",
    "aspect_ratio": "16:9",
    "seed": 8493021,
    "critic_score": 9.0,
    "created_at": "2026-08-29T15:30:00Z",
    "image_url": "/api/image/img-20260829-001.png"
  }
]
```

---

### 3. Server-Sent Events (SSE) Protocol

| Event Name | Payload Schema | Trigger Condition |
| :--- | :--- | :--- |
| `ping` | `":ping"` | Heartbeat every 15s to maintain keep-alive |
| `progress` | `{"job_id":"...", "stage":"...", "percent":45, "message":"..."}` | Sampling step progression or stage transition |
| `reasoning` | `{"job_id":"...", "subagent":"...", "chunk":"..."}` | LLM Art Director / PromptSmith streaming reasoning token |
| `image_ready` | `{"job_id":"...", "image_id":"...", "url":"...", "prompt":"..."}` | Image render completed and saved to disk |
| `critic_evaluation` | `{"job_id":"...", "image_id":"...", "score":8.5, "critique":"...", "passed":true}` | VLM Critic analysis finished |
| `error` | `{"job_id":"...", "code":"ERR_CODE", "message":"..."}` | Pipeline failure or backend rejection |

---

### 4. Error Handling & Edge Cases

| Failure Scenario | Detection Point | System Reaction | User Feedback |
| :--- | :--- | :--- | :--- |
| **Unauthorized Request** | Auth Middleware | Bearer token missing / invalid when auth enabled | Return `401 Unauthorized`. Desktop client displays token input dialog. |
| **Port Already In Use** | Server Startup | `net.Listen` returns `EADDRINUSE` | In local mode, automatically increment port (`8080` -> `8081` -> `8082`) up to 10 attempts, then log chosen port. |
| **Native Webview Driver Missing** | Desktop Runner | Webview library initialization fails | Log informative warning, start in-process web server, and open default OS browser (`xdg-open` / `open` / `start`). |
| **SSE Connection Interrupted** | Client EventSource | `onerror` fired on EventSource | Client initiates exponential backoff auto-reconnect (1s, 2s, 4s, max 10s) and restores active job listeners. |
| **Invalid Image Mask Format** | `POST /api/inpaint` | Canvas mask dimensions mismatch base image | Return `400 Bad Request` with message: `"Mask dimensions must match base image dimensions."` |
| **Backend GPU/Network Timeout** | Generation Pipeline | Context deadline exceeded or backend unreachable | SSE emits `error` event; UI marks job as failed with retry action button. |

---

## Next Steps

1. Review and approve the specification.
2. Proceed to `sdd-design` to detail the Templ component hierarchy, SSE broker architecture, Webview bindings, and state hydration logic.
