# Exploration: ComfyUI Workflow JSON Export & Metadata Interoperability

## 1. Objectives Analysis
- **Goal**: Enable embedding of ComfyUI generation data into generated PNGs and provide CLI tools to inspect/export this data.
- **Scope**:
    - Modify `internal/adapters/image/comfyui.go` to inject PNG metadata (`tEXt` chunks) during image save.
    - Implement `aris workflow inspect` to read PNG chunks and display metadata.
    - Implement `aris workflow export` to save the raw JSON workflow from the `workflow` chunk.

## 2. Technical Strategy: PNG Chunk Manipulation
PNG files are structured as a series of chunks. We need to insert `tEXt` chunks before the `IEND` chunk.

- **Chunk Structure**:
    - `Length` (4 bytes, Big Endian)
    - `ChunkType` (4 bytes: `tEXt`)
    - `Data` (Length bytes)
    - `CRC` (4 bytes, IEEE CRC-32)
- **Implementation Approach**:
    - Use `os.Open` + `os.Create` to modify the PNG.
    - Read standard PNG signature (8 bytes).
    - Iterate through chunks until reaching `IEND` (type `0x49454E44`).
    - Insert new chunks before `IEND`.
    - Update/calculate CRC for each inserted chunk.

## 3. ComfyUI Compatibility
ComfyUI specifically looks for these keys in `tEXt` or `iTXt` chunks:
- `prompt`: The JSON representation of the prompt execution.
- `workflow`: The full workflow JSON (the node graph).

## 4. CLI Architecture
- **Tooling**: Use existing CLI logic in `internal/adapters/ui/cli/`.
- **Command structure**:
    - `aris workflow inspect <file>`: Reads the file, iterates chunks, prints decoded JSON from `prompt` and `workflow` keys.
    - `aris workflow export <file> [-o output.json]`: Extracts `workflow` content, parses it, and writes to disk.

## 5. Implementation Roadmap
1. **Helper**: Create `pkg/imgutil/pngmeta.go` to handle chunk insertion/extraction logic.
2. **Integration**: Update `internal/adapters/image/comfyui.go`'s `Generate` method to call the helper after `io.Copy`.
3. **CLI**: Add `workflow` subcommands to `internal/adapters/ui/cli/cli.go`.

## 6. Risks
- **Data size**: `workflow` JSON can be large; ensure it doesn't exceed PNG chunk limits (though `tEXt` supports large data via multiple chunks or just large fields).
- **CRC Errors**: Incorrect CRC calculation will corrupt the image. Need robust testing with valid PNGs.
- **Dependencies**: Keep implementation dependency-free (standard library only) to avoid bloating `pkg/imgutil`.
