# Specification: Cyberpunk Interactive TUI

## Requirements

### REQ-TUI-1: Terminal User Interface Layout
- **Header**: ARIS banner with active backend, model, memory fact count, and status indicator.
- **Left Pane (Chat & Reasoning)**:
  - Scrollable viewport with user messages, Art Director reasoning thoughts, prompt specs, and delivery notifications.
  - Input textarea with multi-line support and history navigation.
- **Right Pane (Inspector & Controls)**:
  - Active Parameters: Aspect Ratio, Backend, Model, Seed, CFG, Steps.
  - Recalled Memory facts inspector.
  - Terminal Image Preview: ANSI 24-bit halfblock rasterizer showing a scaled thumbnail of the generated image in-terminal.
  - Quick action to open image in OS default viewer (`o` key).

### REQ-TUI-2: Keyboard Shortcuts & Navigation
- `Enter`: Submit prompt / instruction.
- `Tab`: Switch focus between Chat Input and Parameter Controls.
- `Ctrl+C` / `Esc`: Exit TUI.
- `Ctrl+O`: Open current image in native OS photo viewer (`xdg-open` on Linux, `open` on macOS, `start` on Windows).
- `Ctrl+L`: Clear chat history.
- `Ctrl+M`: Toggle Knowledge Graph memory inspector.

### REQ-TUI-3: Terminal Image Rendering (ANSI Halfblocks)
- `pkg/imgutil`: Downsamples generated JPEG/PNG to terminal character grid width (e.g. 40x20).
- Emits ANSI 24-bit background/foreground color escape codes using unicode upper half blocks (`▀` / `\u2580`) for crisp terminal graphics across all terminals (Linux, macOS, Windows Terminal).
