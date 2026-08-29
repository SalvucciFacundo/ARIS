## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1200-1600 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 -> PR 2 -> PR 3 -> PR 4 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |

```text
Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High
```

---

## PR 1: Domain Entities, Reference Utilities & Dimension Validators

- [x] Implement image spec domain structs, reference image modes, and denoise strength validation in `internal/core/domain/types.go`. <!-- sdd-owner: implementation -->
- [x] Write unit tests for domain validation rules in `internal/core/domain/types_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement file/URL validation, size checks (<= 25 MB), MIME type detection, and base64 encoding utilities in `pkg/imgutil/reference.go`. <!-- sdd-owner: implementation -->
- [x] Implement mask dimension alignment and error handling in `pkg/imgutil/mask.go`. <!-- sdd-owner: implementation -->
- [x] Write comprehensive unit tests for image and mask utils in `pkg/imgutil/reference_test.go`. <!-- sdd-owner: implementation -->

---

## PR 2: Multi-Backend Img2Img & Inpainting Adapters

- [x] Update `ImageGenerator` interface and backend factory to accept img2img and inpainting specs in `internal/adapters/image/provider.go`. <!-- sdd-owner: implementation -->
- [x] Implement Fal.ai Flux Dev img2img and inpainting endpoints payload builder in `internal/adapters/image/falai.go`. <!-- sdd-owner: implementation -->
- [x] Implement ComfyUI upload, node graph (`LoadImage`, `VAEEncodeForInpaint`, `KSampler`) in `internal/adapters/image/comfyui.go`. <!-- sdd-owner: implementation -->
- [x] Implement OpenAI DALL-E 2 multipart/form-data edit adapter in `internal/adapters/image/openai.go`. <!-- sdd-owner: implementation -->
- [x] Implement Pollinations image reference / unsupported inpainting check in `internal/adapters/image/pollinations.go`. <!-- sdd-owner: implementation -->
- [x] Write mock and unit tests for all backend adapters in `internal/adapters/image/*_test.go`. <!-- sdd-owner: implementation -->

---

## PR 3: Service Layer & Subagents

- [x] Update `ImageService` interface and domain coordinator for reference and mask routing in `internal/core/services/image_service.go`. <!-- sdd-owner: implementation -->
- [x] Implement `@inpainter` subagent preset with blending instructions and high denoise strength defaults in `internal/core/services/subagents/inpainter.go`. <!-- sdd-owner: implementation -->
- [x] Implement `@restyler` subagent preset with artistic style transfer and denoise defaults in `internal/core/services/subagents/restyler.go`. <!-- sdd-owner: implementation -->
- [x] Integrate subagents into `SubagentManager` in `internal/core/services/subagent_manager.go`. <!-- sdd-owner: implementation -->
- [x] Write unit tests for service layer and subagent prompt generation in `internal/core/services/agent_test.go`. <!-- sdd-owner: implementation -->

---

## PR 4: CLI Subcommand `aris edit`, E2E Integration Suite & Documentation

- [x] Implement `aris edit` CLI subcommand, flag parsing (`--strength`, `--mask`, `--backend`), and output serialization in `internal/adapters/ui/cli/edit.go`. <!-- sdd-owner: implementation -->
- [x] Wire `edit` command into root CLI application in `internal/adapters/ui/cli/cli.go`. <!-- sdd-owner: implementation -->
- [x] Write end-to-end integration test suite simulating img2img and inpainting pipelines in `test/integration/img2img_e2e_test.go`. <!-- sdd-owner: implementation -->
- [x] Write comprehensive user guide and technical documentation for the image-to-image and visual reference pipeline in `docs/img2img.md`. <!-- sdd-owner: implementation -->
