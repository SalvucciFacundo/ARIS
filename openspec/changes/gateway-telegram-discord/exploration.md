# Exploration: ARIS Gateway Integration (Telegram & Discord)

## Overview
This exploration addresses the integration of Telegram and Discord gateways into the ARIS platform to enable remote image generation and agent interaction.

## Architecture Analysis
The current architecture is Hexagonal (`internal/core`, `internal/adapters`). The `AgentService` acts as the orchestrator.

### Proposed Gateway Pattern
Instead of adding new Ports in `internal/core/ports`, we should define a `Gateway` interface or simply inject an `AgentService` instance into the gateway adapter.

Given `AgentService` is quite large, a specific interface representing the required gateway capabilities might be cleaner:

```go
type GatewayAgent interface {
    Generate(ctx context.Context, input string, opts GenerateOptions) (*domain.ImageSpec, *domain.ImageResult, error)
    PipelineGenerate(ctx context.Context, prompt string, opts PipelineOptions) (*PipelineResult, error)
    // Add Memory/Knowledge management if needed
}
```

### Implementation Strategy
1.  **Adapters Location:** `internal/adapters/gateway/telegram.go`, `internal/adapters/gateway/discord.go`, `internal/adapters/gateway/multiplexer.go`.
2.  **Dependencies:** 
    *   **Telegram:** Use `go-telegram-bot-api/telegram-bot-api`. It's robust and simplifies `getUpdates` and file uploads. Stdlib/HTTP client is possible but adds unnecessary complexity (polling logic, JSON marshalling).
    *   **Discord:** Use `bwmarrin/discordgo`. It's the standard. Implementing raw WS + REST is too verbose for a gateway.

## Configuration & Auth
*   `internal/config/config.go` should be extended to support gateway secrets:
    *   `TelegramToken`
    *   `DiscordToken`
    *   `AllowedTelegramChatIDs []int64`
    *   `AllowedDiscordChannelIDs []string` (or UserIDs)

## Command & Syntax Handling
The gateway adapter should perform pre-processing on incoming messages:
1.  **Slash Commands:** Parse `/gen`, `/subagents`, `/backends`, `/help`.
2.  **Subagents:** Detect `@subagent_name` syntax.
3.  **Logic:**
    *   If command detected -> route to respective handler.
    *   Else -> route to `AgentService.Generate`.

## Image Buffer Streaming
*   Telegram API `sendPhoto` accepts `InputFile`. We can stream from `ImageResult.LocalPath` or an `io.Reader`.
*   Discord `files` field supports `io.Reader`.

## Risks
*   **Concurrency:** Handling concurrent message processing to avoid blocking the gateway adapter event loop. Use a worker pool or just spawn `go generate(...)` carefully.
*   **Security:** Ensure the allowlist is checked BEFORE calling any core service.

## Next Steps
1.  Define `Gateway` struct in `internal/adapters/gateway/`.
2.  Add config types.
3.  Implement Telegram adapter using `go-telegram-bot-api`.
4.  Implement Discord adapter using `discordgo`.
