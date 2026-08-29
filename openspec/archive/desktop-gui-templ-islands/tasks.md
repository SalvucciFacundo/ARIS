## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 2400-3200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 -> PR 2 -> PR 3 -> PR 4 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

---

## Task Breakdown by Work Unit

### PR 1: Web Server Engine, Router, Auth Middleware & SSE Broker
- [x] Implement core HTTP web server lifecycle, port-incrementing fallback, and graceful shutdown in `internal/adapters/ui/web/server.go`. <!-- sdd-owner: implementation -->
- [x] Implement HTTP request router and route mappings in `internal/adapters/ui/web/router.go`. <!-- sdd-owner: implementation -->
- [x] Implement Bearer token verification, query parameter token support, and localhost loopback bypass middleware in `internal/adapters/ui/web/auth.go`. <!-- sdd-owner: implementation -->
- [x] Implement thread-safe Server-Sent Events (SSE) publish-subscribe broker, client connection pool, and 15s heartbeat keep-alive ticker in `internal/adapters/ui/web/sse.go`. <!-- sdd-owner: implementation -->
- [x] Write comprehensive unit tests for server routing, authentication enforcement, and race-free concurrent SSE broker broadcasting in `internal/adapters/ui/web/server_test.go` and `sse_test.go`. Run `go test -race ./internal/adapters/ui/web/...`. <!-- sdd-owner: implementation -->

### PR 2: Templ Components & Static Asset Embedding
- [x] Create `//go:embed` asset manager and directory structure in `internal/adapters/ui/web/static/embed.go` embedding `dist/*` assets. <!-- sdd-owner: implementation -->
- [x] Implement core HTML5 layout shell and asset header component in `internal/adapters/ui/web/views/layout.templ`. <!-- sdd-owner: implementation -->
- [x] Implement Left Panel visual gallery thumbnail grid and lightbox modal Templ components in `internal/adapters/ui/web/views/gallery.templ`. <!-- sdd-owner: implementation -->
- [x] Implement Center Panel interactive canvas island container and aspect ratio guide overlay in `internal/adapters/ui/web/views/canvas.templ`. <!-- sdd-owner: implementation -->
- [x] Implement Right Panel conversational chat feed, reasoning accordion, and parameter controls in `internal/adapters/ui/web/views/chat.templ` and `controls.templ`. <!-- sdd-owner: implementation -->
- [x] Implement REST API handlers for generate, inpaint, history, and static asset serving with proper caching headers in `internal/adapters/ui/web/handlers.go`. <!-- sdd-owner: implementation -->
- [x] Write unit tests for handler requests, response payloads, and template compilation in `internal/adapters/ui/web/handlers_test.go`. Run `go test ./internal/adapters/ui/web/...`. <!-- sdd-owner: implementation -->

### PR 3: Interactive Canvas Island, Inpainting Mask Drawing & API Streaming Handlers
- [x] Implement compiled Tailwind CSS stylesheet and HTMX runtime asset bundle in `internal/adapters/ui/web/static/dist/app.css` and `htmx.min.js`. <!-- sdd-owner: implementation -->
- [x] Implement client-side canvas island script for drag-and-drop reference image loading, aspect ratio framing guides, and scaling in `internal/adapters/ui/web/static/dist/islands.js`. <!-- sdd-owner: implementation -->
- [x] Implement inpainting mask brush drawing, eraser mode, offscreen binary canvas serialization, and mask/image FormData multipart packaging in `internal/adapters/ui/web/static/dist/islands.js`. <!-- sdd-owner: implementation -->
- [x] Implement `/api/generate` and `/api/inpaint` backend handler endpoints with multipart validation and error handling in `internal/adapters/ui/web/handlers.go`. <!-- sdd-owner: implementation -->
- [x] Write integration test verifying canvas island event handlers and API request serialization. Run `go test ./internal/adapters/ui/web/...`. <!-- sdd-owner: implementation -->

### PR 4: Desktop Window Runner, CLI Subcommands (`serve` & `gui`), E2E Integration Tests & Docs
- [x] Implement native OS webview window runner in `internal/adapters/ui/desktop/runner.go`. <!-- sdd-owner: implementation -->
- [x] Implement fallback driver detection and automatic system browser launcher (`xdg-open` / `open` / `start`) in `internal/adapters/ui/desktop/fallback.go`. <!-- sdd-owner: implementation -->
- [x] Implement CLI subcommands `aris serve` (alias `aris ui`) and `aris gui` (alias `aris desktop`) in `cmd/aris/serve.go` and `gui.go`. <!-- sdd-owner: implementation -->
- [x] Write end-to-end integration test verifying web server startup, SSE event streaming, and fallback browser invocation in `test/integration/web_e2e_test.go`. Run `go test -v ./test/integration/...`. <!-- sdd-owner: implementation -->
- [x] Write user and developer documentation for the web and desktop UI subsystem in `docs/gui.md`. <!-- sdd-owner: implementation -->
