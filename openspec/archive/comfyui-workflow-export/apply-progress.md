# Apply Progress: ARIS ComfyUI Workflow JSON Export & Metadata Interoperability

- **Change Name**: `comfyui-workflow-export`
- **Milestone**: Milestone 5 (v1.4.0)
- **Status**: Completed (100% Strict TDD verified)
- **Artifact Store**: `hybrid`
- **Delivery Strategy**: `auto-chain` (`stacked-to-main`)

---

## Strict TDD Cycle Evidence

| Work Unit | RED Target Test File | GREEN Implementation File | REFACTOR / Race Check | Status |
|---|---|---|---|:---:|
| **PR 1: PNG Chunk Pipeline** | `pkg/imgutil/png_chunks_test.go` | `pkg/imgutil/png_chunks.go` | `go test -race ./pkg/imgutil/...` | ✅ GREEN |
| **PR 2: ComfyUI Embedding** | `internal/adapters/image/backends_test.go` | `internal/adapters/image/comfyui.go` | `go test -race ./internal/adapters/image/...` | ✅ GREEN |
| **PR 3: Universal Metadata** | `internal/core/services/agent_test.go` | `internal/core/services/agent.go` | `go test -race ./internal/core/services/...` | ✅ GREEN |
| **PR 4: CLI & Integration** | `internal/adapters/ui/cli/workflow_test.go`, `test/integration/workflow_e2e_test.go` | `internal/adapters/ui/cli/workflow.go`, `cli.go` | `go test -race ./...` | ✅ GREEN |

---

## Summary of Code Changes
1. `pkg/imgutil/png_chunks.go`: Pure-Go streaming PNG chunk parser, reader, injector (`InjectPNGMetadata`), and extractor (`ExtractPNGMetadata`) with IEEE CRC-32 recalculation.
2. `internal/adapters/image/comfyui.go`: Dynamic injection of ComfyUI execution graphs (`prompt`) and visual layout (`workflow`) into generated PNG output streams.
3. `internal/core/services/agent.go`: Injection of standard `parameters` metadata chunk across cloud backends (Fal.ai, Pollinations, OpenAI).
4. `internal/adapters/ui/cli/workflow.go`: CLI subcommands `aris workflow inspect <image.png> [--json]` and `aris workflow export <image.png> [-o <path>] [--force]`.
5. `internal/adapters/ui/cli/cli.go`: Wired `case "workflow", "wf": return r.handleWorkflow(args[2:])`.
