# Apply Progress: Specialized Visual Subagent System

## Completed Tasks

- [x] **Task 1: Subagent Domain & Ports**
  - Defined `domain.SubagentDef` and `ports.SubagentStore`.
  - Defined 5 default core subagents (`director`, `promptsmith`, `critic`, `curator`, `enhancer`).
- [x] **Task 2: SQLite Subagent Store**
  - Added `subagent_defs` schema migration in `sqlite.go`.
  - Implemented CRUD methods in `SQLiteSubagentStore`.
  - Added database bootstrap and tests in `subagent_defs_test.go`.
- [x] **Task 3: Subagent Manager & `@name` Dispatcher**
  - Implemented `SubagentManager` with `ExecuteDirect`, `PipelineExecute`, and `ParseSubagentRoute`.
  - Integrated `SubagentManager` with `AgentService` (`ExecuteSubagent`, `PipelineGenerate`).
  - Implemented comprehensive unit tests in `subagent_manager_test.go`.
- [x] **Task 4: CLI & TUI Integration**
  - Added `aris subagents [list|show|run]` CLI command tree.
  - Added `@name` mention detection and direct routing in `aris gen` and interactive TUI.
  - Added subagent styling badges in TUI.
  - Verified full test suite (`go test -race ./...`).

## Files Changed

- `internal/core/services/subagent_manager.go` (new)
- `internal/core/services/subagent_manager_test.go` (new)
- `internal/core/services/agent.go` (modified)
- `internal/core/services/agent_test.go` (modified)
- `internal/adapters/ui/cli/cli.go` (modified)
- `internal/adapters/ui/tui/model.go` (modified)
- `internal/adapters/ui/tui/view.go` (modified)
- `internal/adapters/ui/tui/styles.go` (modified)
- `openspec/changes/specialized-visual-subagents/tasks.md` (modified)

## Verification Evidence

- `go test -v ./...` -> All packages PASS
- `go test -race ./...` -> All packages PASS with race detector enabled
- `go build ./cmd/aris` -> Binary compiles cleanly
- Manual CLI execution verified for `aris subagents list`, `aris subagents show director`, `aris subagents run promptsmith "..."`, `aris gen "@director ..."`
