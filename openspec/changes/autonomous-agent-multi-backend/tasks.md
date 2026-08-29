# Tasks: Autonomous Agent Core & Multi-Backend Suite

## Implementation Tasks

- [x] **Task 1: Hexagonal Domain & Core Interfaces**
  - [x] Create `domain.ImageSpec`, `domain.ImageResult`, `domain.KnowledgeFact`.
  - [x] Define `ports.LLMProvider`, `ports.ImageBackend`, `ports.BackendRegistry`, `ports.KnowledgeGraphStore`, `ports.HistoryStore`.
  - [x] Add unit tests for domain dimension calculations and aspect ratio parsing.

- [x] **Task 2: SQLite Knowledge Graph & History Store**
  - [x] Implement `internal/adapters/db/sqlite.go` with migrations.
  - [x] Implement 3-scope memory (`user`, `style`, `project`) with FTS5 search in `knowledge.go`.
  - [x] Implement generation history persistence in `history.go`.
  - [x] Write comprehensive unit tests in `db_test.go`.

- [x] **Task 3: Pollinations Free Cloud Backend**
  - [x] Implement `internal/adapters/image/pollinations.go` with URL encoding and parameters.
  - [x] Add HTTP mock tests in `pollinations_test.go`.

- [x] **Task 4: Backend Registry & Multi-Backend Implementations**
  - [x] Implement thread-safe `internal/adapters/image/registry.go`.
  - [x] Implement `ComfyUIBackend` (`internal/adapters/image/comfyui.go`) for local GPU generation via REST + WebSocket polling.
  - [x] Implement `FalAIBackend` (`internal/adapters/image/falai.go`) for managed cloud inference.
  - [x] Implement `ReplicateBackend` (`internal/adapters/image/replicate.go`) for community models.
  - [x] Implement `OpenAIBackend` (`internal/adapters/image/openai.go`) for DALL-E 3.
  - [x] Implement `HuggingFaceBackend` (`internal/adapters/image/huggingface.go`) for HF Inference API.
  - [x] Write unit tests with HTTP/mock servers for all backends in `backends_test.go`.

- [x] **Task 5: LLM Prompt Architect & Conversational Iterations**
  - [x] Wire multi-provider LLM adapter (`OpenAIClient` + `PassthroughProvider`).
  - [x] Implement structured prompt reasoning and Knowledge Graph fact recall.
  - [x] Add unit tests for prompt decomposition and agent workflow in `agent_test.go`.

- [x] **Task 6: CLI Command Suite for Multi-Backend**
  - [x] Add `aris backends` command to inspect all registered local & cloud providers and models.
  - [x] Update `aris gen` to support `--backend` flag dynamically with auto-discovery.
  - [x] Support environment variables for zero-config and cloud API keys (`FAL_KEY`, `REPLICATE_API_TOKEN`, `OPENAI_API_KEY`, `HF_TOKEN`, `COMFYUI_HOST`).
  - [x] Verify end-to-end execution.
