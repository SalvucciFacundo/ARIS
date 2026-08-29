# Design: ARIS Gateway Multiplexer (Telegram & Discord)

## 1. System Architecture & Hexagonal Structure

The Gateway subsystem will be implemented as a set of primary adapters (driving adapters) in the Hexagonal Architecture. These adapters listen to external messaging platforms and invoke the `AgentService` (application core) to perform tasks.

### Interfaces
- `GatewayAdapter`: Defines the contract for an individual messaging platform adapter.
- `GatewayMultiplexer`: Manages the lifecycle of multiple adapters.
- `GatewayEngine`: An interface abstracting the `AgentService` to decouple the adapters from the concrete service implementation (Dependency Inversion).

```go
package gateway

import "context"

// GatewayAdapter represents a single messaging platform adapter (Telegram, Discord)
type GatewayAdapter interface {
	// Start begins listening for incoming messages/events
	Start(ctx context.Context) error
	// Stop gracefully shuts down the adapter
	Stop(ctx context.Context) error
	// Name returns the adapter's identifier
	Name() string
}

// GatewayMultiplexer manages multiple GatewayAdapters
type GatewayMultiplexer interface {
	// Start starts all registered adapters concurrently
	Start(ctx context.Context) error
	// Stop gracefully stops all adapters
	Stop(ctx context.Context) error
}

// GatewayEngine abstracts the core AgentService for the gateway adapters
type GatewayEngine interface {
	Generate(ctx context.Context, prompt string, opts GenerateOptions) (string, error)
	ExecuteSubagent(ctx context.Context, subagent, prompt string) (string, error)
	PipelineGenerate(ctx context.Context, prompt string) (PipelineResult, error)
	SearchMemory(ctx context.Context, query string, filter string, limit int) ([]string, error)
	// Additional informational methods...
}
```

### Directory Structure
```
internal/
└── adapters/
    └── gateway/
        ├── multiplexer.go     # GatewayMultiplexer implementation
        ├── engine.go          # GatewayEngine interface and related types
        ├── queue.go           # Worker pool / semaphore for concurrency
        ├── parser.go          # Command and @subagent routing pipeline
        ├── telegram/          
        │   ├── adapter.go     # Telegram adapter implementation
        │   ├── handlers.go    # Telegram specific message handlers
        │   └── auth.go        # Allowlist validation
        └── discord/
            ├── adapter.go     # Discord adapter implementation
            ├── handlers.go    # Discord specific event handlers
            └── auth.go        # Allowlist validation
```

## 2. Concurrency & Resource Management

To prevent GPU VRAM exhaustion, all incoming generation requests will be routed through a centralized worker pool/semaphore managed by the Multiplexer or a shared `JobQueue`.

- **Semaphore/Channel Pattern**: A buffered channel of size `Gateway.Concurrency` acts as a semaphore. A queue channel of size `Gateway.MaxQueue` holds pending jobs.
- **Queue Depth Bounding**: If the queue channel is full, the job is immediately rejected, returning a "Queue full" message.
- **Timeout Handling**: `context.WithTimeout` will be attached to each generation request. If a shutdown signal is received, the queue stops accepting new jobs and waits for in-flight jobs up to a grace period.

```go
package gateway

type Job struct {
	Ctx      context.Context
	Task     func(context.Context) error
	ReplyErr func(error)
}

type JobQueue struct {
	workers int
	queue   chan Job
	sem     chan struct{}
}

func NewJobQueue(concurrency, maxQueue int) *JobQueue {
	return &JobQueue{
		workers: concurrency,
		queue:   make(chan Job, maxQueue),
		sem:     make(chan struct{}, concurrency),
	}
}

// Submit tries to add a job; returns error if queue is full
func (q *JobQueue) Submit(job Job) error {
	select {
	case q.queue <- job:
		return nil
	default:
		return ErrQueueFull
	}
}
```

## 3. Adapter Details

### Telegram Adapter
- **Library**: `go-telegram-bot-api/telegram-bot-api/v5`
- **Polling Loop**: Uses `getUpdates` with a long-polling offset.
- **Auth Validator**: Checks `update.Message.Chat.ID` and `update.Message.From.ID` against `AllowedChatIDs` and `AllowedUserIDs`. If unauthorized, drops the message silently (or sends one-time alert).
- **Message Dispatcher**: Passes authorized text to the `parser.go` logic. If a job is created, it submits to `JobQueue`.
- **Photo/Document Upload**: Uses `tgbotapi.NewPhoto` or `tgbotapi.NewDocument`.
- **Typing Action**: A goroutine running alongside the generation job that sends `tgbotapi.NewChatAction(chatID, "upload_photo")` every 4-5 seconds until completion or context cancellation.

### Discord Adapter
- **Library**: `bwmarrin/discordgo`
- **Event Listener**: Registers a `MessageCreate` event handler.
- **Auth Validator**: Checks `m.ChannelID` and `m.Author.ID`. Ignores if `m.Author.Bot == true`.
- **Message Dispatcher**: Similar to Telegram, uses `parser.go`.
- **Attachment Upload**: Uses `discordgo.File` struct with `ChannelMessageSendComplex`.
- **Embed Builder**: Constructs a `discordgo.MessageEmbed` with the image URL (`attachment://...`), title, and generation metadata fields.
- **Typing Indicator**: Calls `Session.ChannelTyping(channelID)` in a loop during generation.

## 4. Command & Subagent Routing Pipeline

The `parser.go` module handles incoming text extraction.

1. **Prefix Matching**:
   - If text starts with `/gen `, extract prompt and flags. Dispatch to `GatewayEngine.Generate`.
   - If text starts with `@`, extract subagent name (e.g., `@director`). Dispatch to `GatewayEngine.ExecuteSubagent`.
   - If text matches `/pipeline`, `/subagents`, `/backends`, `/memory`, `/status`, `/help`, route to corresponding engine info methods.
2. **Flag Parsing**: Use a lightweight flag parser (e.g., `pflag` configured not to exit on error) to extract options like `--ratio 16:9` from the `/gen` command.
3. **Response Formatting**: The pipeline returns unified metadata (result path, text response, metadata) which the specific adapter formats into platform-native API calls.

## 5. Config Extension

Extend `internal/config/config.go`:

```go
type GatewayConfig struct {
	Concurrency int            `yaml:"concurrency"` 
	MaxQueue    int            `yaml:"max_queue"`   
	Telegram    TelegramConfig `yaml:"telegram"`
	Discord     DiscordConfig  `yaml:"discord"`
}

type TelegramConfig struct {
	Enabled        bool    `yaml:"enabled"`
	BotToken       string  `yaml:"bot_token"`
	AllowedChatIDs []int64 `yaml:"allowed_chat_ids"`
	AllowedUserIDs []int64 `yaml:"allowed_user_ids"`
	SendAsDocument bool    `yaml:"send_as_document"`
}

type DiscordConfig struct {
	Enabled           bool     `yaml:"enabled"`
	BotToken          string   `yaml:"bot_token"`
	AllowedChannelIDs []string `yaml:"allowed_channel_ids"`
	AllowedUserIDs    []string `yaml:"allowed_user_ids"`
}
```
Add environment variable bindings in the `viper` setup (`config.LoadConfig()`).

## 6. Sequence Diagrams (ASCII)

### User Interaction: Successful Generation
```
User      Adapter(TG/Discord)      Parser & Auth      JobQueue      GatewayEngine
 |                 |                     |               |                 |
 |--- /gen cat --->|                     |               |                 |
 |                 |-- Validate Auth --->|               |                 |
 |                 |<------- OK ---------|               |                 |
 |                 |-- Parse Command --->|               |                 |
 |                 |<---- Job Payload ---|               |                 |
 |                 |--- Submit Job --------------------->|                 |
 |                 |<------ Enqueued --------------------|                 |
 |                 |                                     |                 |
 |<- Typing Action |                                     |--- Generate --->|
 |                 |                                     |<-- Output.jpg --|
 |<-- Img + Meta --|<----- Format Platform Message ------|<-- Result ------|
```

### Error Handling: Queue Full / Unauthorized
```
User      Adapter(TG/Discord)      Parser & Auth      JobQueue
 |                 |                     |               |
 |--- /gen cat --->|                     |               |
 |                 |-- Validate Auth --->|               |
 |                 |<---- UNAUTHORIZED --|               |
 |<- Access Denied |                     |               |
 |                 |                     |               |
 |--- /gen dog --->|                     |               |
 |                 |-- Validate Auth --->|               |
 |                 |<------- OK ---------|               |
 |                 |--- Submit Job --------------------->|
 |                 |<----- ErrQueueFull -----------------|
 |<-- Queue Full --|                     |               |
```

## 7. Testing Strategy

- **Mocking Bot APIs (RED -> GREEN TDD)**: 
  Create interfaces for the external bot APIs or use httptest to mock the Telegram API server and Discord API endpoints. This allows writing deterministic unit tests for the adapters without real network calls.
- **Unit Testing Auth & Parser**: 
  Thorough table-driven tests for `auth.go` (testing allowlists) and `parser.go` (testing various command formats, invalid flags, and missing text).
- **Concurrency Race Test Suite**: 
  Use `go test -race` on the `JobQueue`. Create a test that submits `Concurrency + MaxQueue + 5` jobs simultaneously. Verify that exactly `Concurrency` jobs run at once, `MaxQueue` jobs wait, and `5` jobs immediately return `ErrQueueFull`.
- **Integration**:
  Mock `GatewayEngine` using something like `mockery` to ensure adapters correctly pass the prompt and options to the engine.
