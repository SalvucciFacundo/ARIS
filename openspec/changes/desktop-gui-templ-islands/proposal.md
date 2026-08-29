# Proposal: ARIS Desktop GUI & Web Interface

## 1. Intent and Problem Statement
ARIS currently lacks a visual interface for rich graphical workflows. The project needs a visually interactive 3-panel canvas (Gallery, Drag-and-Drop Image Canvas with Inpainting Mask Drawing, and Interactive Chat with Reasoning Stream & Controls) to empower users in complex visual generation and critic workflows. 
Crucially, this UI must run as a single Go binary, be accessible in the browser, support self-hosting on a remote VPS with token-based authentication, and be capable of opening as a native desktop window.

## 2. Proposed Solution
We propose introducing a new `web` UI adapter (`internal/adapters/ui/web/`) using a modern, Go-native web stack.
*   **Stack**: `templ` + `github.com/SalvucciFacundo/templ-islands` + `htmx` + embedded Tailwind CSS (via `embed.FS`).
*   **Real-time Updates**: Progress updates, reasoning streams, and generation events will be streamed via Server-Sent Events (SSE) at `/api/events`.
*   **Dual-Mode Deployment**:
    1.  **Headless (`aris serve` / `aris ui`)**: A web server with optional Bearer Token authentication for VPS deployments.
    2.  **Desktop Native (`aris gui` / `aris desktop`)**: A native desktop window runner connecting either to a local in-process engine or a remote VPS (`--remote <url> --token <token>`).
*   **3-Panel Layout**:
    *   **Left Panel**: Visual Gallery & History inspector with full-resolution zoom.
    *   **Center Panel**: Interactive Canvas with a drag-and-drop reference image loader, aspect ratio guide, and an inpainting mask brush.
    *   **Right Panel**: Conversational Chat with a reasoning stream, subagent routing badges (`@director`, `@anime`, `@inpainter`, etc.), and parameter knobs.

## 3. Scope and Affected Areas
*   **`internal/adapters/ui/web/`**: New directory for the UI components, endpoints, and assets.
*   **`cmd/aris/`**: New CLI subcommands (`serve`/`ui` and `gui`/`desktop`) for bootstrapping the web server and desktop runner.
*   **SSE streaming mechanism**: Enhancing or adding event emitters in the core domain to broadcast real-time feedback.

## 4. Alternatives Considered
*   **Electron / Tauri / Next.js**: Rejected due to binary bloat, memory footprint overhead, and the introduction of external runtime dependencies (Node.js/NPM). The goal is to maintain a single, lightweight Go binary.
*   **Pure React/Vue SPA**: Rejected in favor of HTMX and Templ Islands to keep UI state management tightly coupled with the Go backend, reducing context switching and API duplication.

## 5. Risks and Mitigations
*   **Native webview driver availability on Linux/macOS/Windows**: 
    *   *Mitigation*: Use robust cross-platform libraries (like webview/webview or Wails) and gracefully fallback to opening the default web browser if a native window cannot be initialized.
*   **SSE connection drops during long diffusion runs**:
    *   *Mitigation*: Implement robust client-side reconnection logic and server-side state hydration so the UI can resume listening without losing generation progress.

## 6. Out of Scope (Non-goals)
*   **Native mobile apps (iOS/Android)**: Building dedicated App Store / Play Store native apps is out of scope. The responsive mobile web UI covers mobile device usage.

## 7. Success Criteria
*   Users can launch the UI locally via a single command and see a native window.
*   Users can run the engine on a VPS and connect the desktop client to it using `--remote` and `--token`.
*   All three panels (Gallery, Canvas, Chat) communicate fluidly over SSE/HTMX without requiring full page reloads.
*   The final application builds into a single self-contained Go binary.

## Proposal question round
*These questions are meant to uncover edge cases, business rules, and product tradeoffs before writing the specifications.*
1. **Remote Connections:** When the desktop app connects to a remote VPS (`--remote`), should it cache images locally to improve performance and avoid latency, or always stream them from the remote server?
2. **Inpainting Canvas:** How precise must the inpainting mask drawing be? Do we need adjustable brush sizes and undo/redo history in this first UI slice, or is a simple binary mask adequate?
3. **Subagent Routing:** Are the subagent routing badges (`@director`, `@anime`, `@inpainter`) dynamically fetched from the engine, or statically defined in the UI for this iteration?
