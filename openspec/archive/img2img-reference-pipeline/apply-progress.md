# Apply Progress: ARIS Img2Img & Visual Reference Pipeline

**Change Name:** `img2img-reference-pipeline`  
**Status:** Completed  
**Artifact Store Mode:** `hybrid`  
**Delivery Strategy:** `auto-chain` (Stacked PRs to main)  

---

## 1. Completed Tasks Summary

| PR Work Unit | Task | Status | Persisted Checkbox |
| :--- | :--- | :---: | :---: |
| **PR 1** | Implement `ReferenceMode` enum & `ImageSpec` extensions in `internal/core/domain/types.go` | Completed | `- [x]` |
| **PR 1** | Write unit tests in `internal/core/domain/types_test.go` | Completed | `- [x]` |
| **PR 1** | Implement image/URL loading, size bounds, MIME & Base64 in `pkg/imgutil/reference.go` | Completed | `- [x]` |
| **PR 1** | Implement mask dimension alignment & helpers in `pkg/imgutil/mask.go` | Completed | `- [x]` |
| **PR 1** | Write unit tests in `pkg/imgutil/reference_test.go` | Completed | `- [x]` |
| **PR 2** | Implement Fal.ai img2img & inpainting endpoint payloads in `internal/adapters/image/falai.go` | Completed | `- [x]` |
| **PR 2** | Implement ComfyUI `/upload/image` & inpainting/img2img node graphs in `internal/adapters/image/comfyui.go` | Completed | `- [x]` |
| **PR 2** | Implement OpenAI DALL-E 2 multipart `/v1/images/edits` adapter in `internal/adapters/image/openai.go` | Completed | `- [x]` |
| **PR 2** | Implement Pollinations img2img query / inpaint error check in `internal/adapters/image/pollinations.go` | Completed | `- [x]` |
| **PR 2** | Write backend adapter unit tests in `internal/adapters/image/backends_test.go` | Completed | `- [x]` |
| **PR 3** | Update `AgentService.Generate` reference/mask options in `internal/core/services/agent.go` | Completed | `- [x]` |
| **PR 3** | Implement `@inpainter` and `@restyler` presets in `internal/core/domain/subagent.go` | Completed | `- [x]` |
| **PR 3** | Update `SubagentManager` pipeline routing in `internal/core/services/subagent_manager.go` | Completed | `- [x]` |
| **PR 3** | Write unit tests in `internal/core/services/agent_test.go` & `subagent_manager_test.go` | Completed | `- [x]` |
| **PR 4** | Implement `aris edit` subcommand and flag parsing in `internal/adapters/ui/cli/edit.go` | Completed | `- [x]` |
| **PR 4** | Wire `edit` into root CLI in `internal/adapters/ui/cli/cli.go` | Completed | `- [x]` |
| **PR 4** | Write E2E integration test suite in `test/integration/img2img_e2e_test.go` and CLI unit tests in `edit_test.go` | Completed | `- [x]` |
| **PR 4** | Write comprehensive documentation in `docs/img2img.md` | Completed | `- [x]` |

---

## 2. TDD Cycle Evidence

| Test Suite / Target | RED Failure Evidence | GREEN Passing Evidence | Triangulate / Refactor Notes |
| :--- | :--- | :--- | :--- |
| `internal/core/domain/types_test.go` | `undefined: domain.ModeText2Img`, `undefined: domain.ModeImg2Img` | `PASS: TestReferenceMode_Constants`, `PASS: TestImageSpec_ApplyDefaults` | Added `IsImg2Img()`, `IsInpaint()`, `ApplyDefaults()`, `Validate()` |
| `pkg/imgutil/reference_test.go` | `undefined: imgutil.LoadAndValidateImage`, `undefined: imgutil.GetDimensions` | `PASS: TestLoadAndValidateImage_*`, `PASS: TestValidateMatchingDimensions`, `PASS: TestValidateMask` | Added pure-Go WEBP header dimension decoder, MIME magic checks, and $\le 25\text{MB}$ guards |
| `internal/adapters/image/backends_test.go` | `backend.GenerateWithBaseURL undefined`, backend payload mismatch | `PASS: TestFalAIBackend_Img2ImgAndInpaintPayload`, `PASS: TestComfyUIBackend_Img2ImgAndInpaint`, `PASS: TestOpenAIBackend_EditMultipart`, `PASS: TestPollinationsBackend_InpaintUnsupported` | Handled Base64 data URI conversion, ComfyUI multipart image uploads and inpaint graph wiring |
| `internal/core/services/agent_test.go` | `unknown field DenoiseStrength in struct literal of type services.GenerateOptions` | `PASS: TestAgentService_GenerateImg2ImgAndInpaintOptions`, `PASS: TestSubagentManager_VisualSubagents` | Added `@inpainter` and `@restyler` default subagent definitions; propagated reference options |
| `internal/adapters/ui/cli/edit_test.go` & `test/integration/img2img_e2e_test.go` | Missing `aris edit` CLI command routing | `PASS: TestRunner_EditCommand`, `PASS: TestImg2Img_E2E_Pipeline` | Added `handleEdit` flag parser, dimension pre-checks, and full E2E test roundtrip |

---

## 3. Files Created and Modified

### Modified Files
- `internal/core/domain/types.go` — Added `ReferenceMode` enum, `ImageSpec` reference/mask fields, defaulting and validation methods.
- `internal/core/domain/subagent.go` — Added `@inpainter` and `@restyler` presets to `DefaultSubagents()`.
- `internal/adapters/image/falai.go` — Added `image_url`, `mask_url`, and `strength` payload building; base64 data URI preparation.
- `internal/adapters/image/comfyui.go` — Added `/upload/image` multipart handler, dynamic inpaint/img2img node graphs.
- `internal/adapters/image/openai.go` — Added multipart/form-data handler for `POST /v1/images/edits`.
- `internal/adapters/image/pollinations.go` — Added URL parameter image reference support and fail-safe inpainting rejection.
- `internal/core/services/agent.go` — Added `MaskImage`, `DenoiseStrength`, `Mode` to `GenerateOptions` and domain spec propagation.
- `internal/core/services/subagent_manager.go` — Added `MaskImage`, `DenoiseStrength`, `Mode` to `PipelineOptions`.
- `internal/adapters/db/subagent_defs_test.go` — Updated subagent count assertion to match extended default subagents list.
- `internal/adapters/ui/cli/cli.go` — Registered `edit` subcommand and updated help text.
- `openspec/changes/img2img-reference-pipeline/tasks.md` — Marked all tasks completed.

### Newly Created Files
- `internal/core/domain/types_test.go`
- `pkg/imgutil/reference.go`
- `pkg/imgutil/mask.go`
- `pkg/imgutil/reference_test.go`
- `internal/adapters/ui/cli/edit.go`
- `internal/adapters/ui/cli/edit_test.go`
- `test/integration/img2img_e2e_test.go`
- `docs/img2img.md`
- `openspec/changes/img2img-reference-pipeline/apply-progress.md`

---

## 4. Test Verification Suite
All unit and integration tests passed cleanly with race detector enabled:
```bash
go test -count=1 -race ./...
```
Output:
```text
ok  	aris/internal/adapters/db	1.190s
ok  	aris/internal/adapters/gateway	1.102s
ok  	aris/internal/adapters/gateway/discord	1.135s
ok  	aris/internal/adapters/gateway/telegram	1.134s
ok  	aris/internal/adapters/image	1.122s
ok  	aris/internal/adapters/ui/cli	1.045s
ok  	aris/internal/adapters/vision	1.011s
ok  	aris/internal/config	1.007s
ok  	aris/internal/core/domain	1.005s
ok  	aris/internal/core/services	64.873s
ok  	aris/pkg/imgutil	1.564s
ok  	aris/test/integration	1.380s
```
