# Proposal: ARIS ComfyUI Workflow JSON Export & Metadata Interoperability

## Intent
Implement native PNG metadata (`tEXt`/`iTXt`) injection and extraction in ARIS to embed ComfyUI execution graphs and parameters directly into generated images. This solves the problem of losing generation recipes once images are saved to disk. By embedding this data, users can preserve reproducibility, inspect configuration via the CLI, and seamlessly drag-and-drop ARIS-generated PNGs back into ComfyUI to reload the exact node graph.

## Scope
- **`pkg/imgutil/pngmeta.go`**: Pure-Go implementation for reading, injecting, and recalculating IEEE CRC-32 checksums for PNG chunks without external dependencies.
- **`internal/adapters/image/comfyui.go`**: Update the image generation flow to inject `prompt`, `workflow`, and `parameters` JSON strings immediately before the `IEND` chunk during image save.
- **CLI Commands**: Implement `aris workflow inspect <image.png>` and `aris workflow export <image.png> [-o workflow.json]`.

## Affected Areas
- Image processing and saving routines within the ComfyUI adapter.
- CLI command registry and argument parsing.

## Risks & Edge Cases
- **CRC Validation**: Incorrect checksum calculation will render the output PNG corrupted or unreadable to strict parsers.
- **Payload Size Constraints**: Uncompressed `tEXt` chunks might significantly increase file sizes if workflows contain embedded assets (e.g., base64 masks). If large, `iTXt` (compressed) may be necessary.
- **Streaming/Memory**: Parsing large PNG files directly into memory could spike RAM usage; stream processing (read-copy-write) is required for chunk insertion.

## Rollback
- Revert the ComfyUI adapter changes to write raw downloaded bytes bypassing `imgutil`.
- Remove the CLI `workflow` subcommands.

## Success Criteria
- **Drag-and-Drop**: ARIS-generated PNGs dragged into a vanilla ComfyUI web canvas completely load the original node graph.
- **Inspection**: `aris workflow inspect <image.png>` successfully locates and prints ComfyUI metadata chunks.
- **Export**: `aris workflow export <image.png> -o workflow.json` successfully extracts the workflow graph into a valid JSON file.

## Proposal Question Round
Before proceeding to Spec & Design, please review these assumptions and clarify product tradeoffs:

1. **Export failures**: Should `aris workflow export` strictly error out if the PNG lacks ComfyUI specific keys (`prompt`/`workflow`), or should it gracefully dump any generic metadata it finds? 
2. **File Overwrites**: When running `export -o workflow.json`, should the CLI overwrite an existing file silently, or fail and require a `--force` flag? *(Assumption: fail by default to prevent data loss)*.
3. **CLI Output Format**: For `aris workflow inspect`, do we want a human-readable summary table of the nodes, or raw JSON output to support piping into tools like `jq`? *(Assumption: human-readable by default, with an optional `--json` flag)*.
4. **Compression Need**: Are we expecting ARIS to handle exceptionally large workflows (e.g., with embedded base64 images) that would mandate implementing `iTXt` (compressed) chunks immediately, or is uncompressed `tEXt` acceptable for this first slice? *(Assumption: `tEXt` is sufficient for the first slice)*.