# Tasks: Specialized Visual Subagent System

## Implementation Tasks

- [x] **Task 1: Subagent Domain & Ports**
  - [x] Define `domain.SubagentDef` and `ports.SubagentStore` in `internal/core/domain` and `internal/core/ports`.
  - [x] Define 5 default pre-configured subagents (`director`, `promptsmith`, `critic`, `curator`, `enhancer`).

- [x] **Task 2: SQLite Subagent Store (`internal/adapters/db/subagent_defs.go`)**
  - [x] Add `subagent_defs` table migration in `internal/adapters/db/sqlite.go`.
  - [x] Implement CRUD methods for subagent definitions.
  - [x] Auto-bootstrap default subagent definitions on first database initialization.
  - [x] Write unit tests in `subagent_defs_test.go`.

- [x] **Task 3: Subagent Manager & `@name` Dispatcher (`internal/core/services/subagent_manager.go`)**
  - [x] Implement `SubagentManager` supporting `@name` message routing and isolated prompt execution.
  - [x] Implement multi-agent autonomous pipeline coordination (`Director -> PromptSmith -> Backend -> Critic -> Enhancer`).
  - [x] Write unit tests in `subagent_manager_test.go`.

- [x] **Task 4: CLI & TUI Integration**
  - [x] Add `aris subagents [list|show|run]` command to CLI.
  - [x] Update `aris chat` TUI to highlight and route `@name` mentions to the respective subagent.
  - [x] Verify full test suite.
