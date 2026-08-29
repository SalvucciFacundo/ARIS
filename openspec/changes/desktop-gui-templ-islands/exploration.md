# Exploration: Desktop GUI and Web Interface with Templ + Islands (ARIS)

## Overview
This exploration investigates the architectural requirements and technical strategy for integrating a modern, component-based web UI into ARIS, supporting both a headless web server mode and a desktop native GUI.

## Architecture Analysis
ARIS currently features a `cli` and a `tui` adapter within `internal/adapters/ui/`. Integrating a web-based UI requires introducing a new `web` adapter.

### Proposed Architecture for `internal/adapters/ui/web/`
- **Engine**: We will use `github.com/a-h/templ` for type-safe HTML components.
- **Interactivity**: 
    - **HTMX**: For standard AJAX-based dynamic updates.
    - **Templ Islands**: `github.com/SalvucciFacundo/templ-islands` allows embedding interactive "islands" of logic into server-side rendered templates. This is ideal for our complex UI elements.
- **Styling**: Embedded Tailwind CSS (via `embed.FS` and `templ`'s asset bundling).
- **Communication**: 
    - HTTP REST/API endpoints for state.
    - SSE (Server-Sent Events) via `/api/events` for streaming generation updates and VLM critic feedback to the UI.

### Dual-Mode Strategy
- **Headless (`aris serve`)**: Standard `net/http` server serving the Templ-based UI. Needs auth support (token-based).
- **Desktop Native (`aris gui`)**: Utilize a native GUI wrapper (e.g., Wails, as seen in `go/bin/wails` and related dependencies).
    - *Decision*: We can leverage Wails or a simple WebView2 wrapper that loads our internal web server. Given Wails support is in `go/bin`, we should evaluate if Wails is preferred over a pure WebView approach.

## UI Structure
- **Gallery Island**: Visual history with zoom, metadata inspector, and image lifecycle management.
- **Canvas Island**: In-process image loader, mask editing, aspect ratio management.
- **Chat/Control Island**: Sub-agent interaction, model selection, and prompt architect inspector.

## Implementation Roadmap
1. Create `internal/adapters/ui/web` structure.
2. Setup `templ` + `templ-islands` workflow.
3. Integrate `Tailwind` for component styling.
4. Implement `aris serve` with authentication middleware.
5. Integrate with native windowing system.

## Risks
- **Asset Bundling**: Managing `embed.FS` efficiently for both web and native modes.
- **State Sync**: Ensuring the internal Go domain state stays in sync with optimistic UI updates in islands.
- **Performance**: High latency in VLM critic feedback during streaming generation.
