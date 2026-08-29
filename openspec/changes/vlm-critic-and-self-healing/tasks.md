# Tasks: VLM Vision Critic & Autonomous Self-Healing Loop

## Implementation Tasks

- [x] **Task 1: Vision Critic Adapter (`internal/adapters/vision/critic.go`)**
  - [x] Implement Base64 image encoding and multi-modal message payload assembly.
  - [x] Implement OpenAI / Ollama vision API communication.
  - [x] Parse structured JSON critique output (`score`, `adherence`, `defects`, `suggested_fix`).
  - [x] Write unit tests with mock vision HTTP server (`critic_test.go`).

- [x] **Task 2: Critic Service & Self-Healing Loop (`internal/core/services/critic_service.go`)**
  - [x] Implement `CriticService` with threshold evaluation.
  - [x] Implement self-healing logic (single re-roll with prompt delta adjustment).
  - [x] Write unit tests for self-healing pass/fail scenarios (`critic_service_test.go`).

- [x] **Task 3: Integration with Config, Agent, and CLI**
  - [x] Update `internal/config/config.go` to add `Critic` settings.
  - [x] Wire `CriticService` into `AgentService.Generate`.
  - [x] Add `--critic` and `--auto-heal` flags to `aris gen` in `internal/adapters/ui/cli/cli.go`.
  - [x] Run full test suite.
