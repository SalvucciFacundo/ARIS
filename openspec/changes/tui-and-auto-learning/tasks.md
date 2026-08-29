# Tasks: Cyberpunk TUI & Auto-Learning Loop

## Implementation Tasks

- [x] **Task 1: Auto-Learning Engine (`AutoLearner`)**
  - [x] Create `internal/core/services/autolearn.go` with reflection heuristics & LLM distillation.
  - [x] Connect `AutoLearner` to `AgentService` so it triggers automatically after generation turns.
  - [x] Write comprehensive unit tests in `autolearn_test.go`.

- [x] **Task 2: Terminal Image Rasterizer (`pkg/imgutil`)**
  - [x] Implement ANSI 24-bit halfblock downsampling rasterizer (`pkg/imgutil/rasterizer.go`).
  - [x] Implement OS default viewer launcher (`pkg/imgutil/open.go`).
  - [x] Write unit tests for image downsampling and ANSI output.

- [x] **Task 3: Bubbletea Interactive TUI (`internal/adapters/ui/tui/`)**
  - [x] Install Bubbletea, Lipgloss, Bubbles dependencies.
  - [x] Implement Cyberpunk theme styles, layout, and split view panes (`styles.go`).
  - [x] Implement TUI model, update loop, and message handlers (`model.go`, `view.go`).
  - [x] Integrate real-time generation dispatch, reasoning thoughts, and parameter switching.

- [x] **Task 4: CLI Integration & Verification**
  - [x] Add `aris chat` / `aris tui` command to CLI entrypoint (`internal/adapters/ui/cli/cli.go`).
  - [x] Verify full end-to-end flow and all tests passing.
