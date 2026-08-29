# ARIS Desktop GUI & Web Interface Documentation

ARIS features a visual 3-panel workspace built with **Go Templ**, **HTMX**, **Tailwind CSS**, and **Templ Islands**, packaged directly into a single binary with zero Node.js runtime dependencies.

---

## 1. Quick Start

### Start Local Web Server
```bash
# Start on default localhost:8080 (or auto-incrementing if busy)
aris serve

# Or with custom port and remote token protection
aris serve --host 0.0.0.0 --port 9090 --token "my-secret-token"

# Alias:
aris ui
```

### Start Native Desktop GUI / Browser Window
```bash
# Launch local in-process web server and open system browser/webview
aris gui

# Connect to a remote ARIS VPS server without starting local engine
aris gui --remote https://vps.aris.ai:8080 --token "vps-secret-xyz"

# Alias:
aris desktop
```

---

## 2. Workspace Layout

The visual interface is organized into a responsive 3-panel layout:

```text
+---------------------+-------------------------------+-------------------------+
|    Visual Gallery   |     Center Canvas & Inpaint   |   Reasoning Chat &      |
|     (Left Panel)    |         (Center Panel)        |   Parameter Controls    |
|                     |                               |      (Right Panel)      |
|  * History grid     |  * Drag-and-drop dropzone     |  * Conversational feed  |
|  * Metadata badges  |  * Aspect ratio overlay       |  * Streaming reasoning  |
|  * Lightbox modal   |  * Mask drawing & brush tool  |  * @subagent badges     |
|  * Image reuse      |  * Binary mask serialization  |  * Ratio & model knobs  |
+---------------------+-------------------------------+-------------------------+
```

### Left Panel: Visual Gallery
- **Chronological Stream**: Automatically populates with past generations and new real-time renders via Server-Sent Events (SSE).
- **Metadata Badges**: Shows aspect ratio, backend, model, and VLM Critic quality scores.
- **Lightbox Modal**: Click any thumbnail to inspect full resolution, zoom/pan, download raw PNG, or click **"Use as Reference"** to load it directly into the Center Canvas.

### Center Panel: Interactive Canvas & Inpainting Island
- **Drag & Drop**: Drop any PNG, JPEG, or WebP image onto the canvas dropzone to use as reference.
- **Inpainting Mask Brush**: Paint over regions to edit with a semi-transparent magenta brush (`#FF005580`).
- **Brush & Eraser Controls**: Adjust brush radius between 5px and 120px, switch between brush/eraser mode, or clear masks.
- **Aspect Ratio Guide**: Real-time framing guide overlay for `1:1`, `16:9`, `9:16`, `4:3`, `3:2`, and `21:9`.
- **Inpaint Area**: Serializes base image and binary alpha mask into multipart `FormData` sent to `POST /api/inpaint`.

### Right Panel: Chat & Subagent Controls
- **Conversational Input**: Multi-line input (`Shift+Enter` for newline, `Enter` to submit).
- **Subagent Routing**: Type `@` or click badges (e.g. `@director`, `@promptsmith`, `@inpainter`) to route tasks directly to specialized agents.
- **Parameter Controls**: Tune backend (`pollinations`, `comfyui`, `falai`), model, aspect ratio, steps, CFG scale, and seed. Settings are saved to browser `localStorage`.

---

## 3. Remote VPS Mode & Security

When running ARIS on a remote GPU server or VPS:

1. **Start the Web Server on the VPS:**
   ```bash
   ARIS_WEB_TOKEN="your-secure-token-here" aris serve --host 0.0.0.0 --port 8080
   ```
2. **Access from Local Machine / Browser:**
   - Open: `http://<vps-ip>:8080?token=your-secure-token-here`
   - Or use the desktop client launcher:
     ```bash
     aris gui --remote http://<vps-ip>:8080 --token your-secure-token-here
     ```
3. **Authentication Rules:**
   - Localhost loopback (`127.0.0.1`, `::1`) bypasses token authentication by default.
   - Remote requests require `Authorization: Bearer <token>` or `?token=<token>`.
   - Public assets under `/assets/*` are unauthenticated and cached with immutable headers.

---

## 4. API Reference

| Method | Path | Description | Payload / Response |
| :--- | :--- | :--- | :--- |
| `GET` | `/` | Application HTML shell | HTML (Templ layout) |
| `GET` | `/api/events` | Real-time Server-Sent Events | SSE stream (`progress`, `reasoning`, `image_ready`, `critic_evaluation`) |
| `POST` | `/api/generate` | Dispatch generation job | JSON `{ "prompt": "...", "aspect_ratio": "16:9", ... }` |
| `POST` | `/api/inpaint` | Dispatch inpainting job | Multipart form (`prompt`, `image`, `mask`, `backend`, `model`) |
| `GET` | `/api/history` | Chronological generation records | JSON array of generation objects |
| `GET` | `/api/image/{id}` | Serve rendered image file | Image binary (`image/png`, `image/webp`) |
| `GET` | `/api/subagents` | List registered subagents | JSON array |
| `GET` | `/api/backends` | List registered backends & models | JSON array |
| `GET` | `/assets/*` | Embedded static CSS / JS / icons | Static assets |
