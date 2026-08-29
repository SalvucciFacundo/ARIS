# Specification: ARIS Gateway Multiplexer (Telegram & Discord)

## Purpose

Define the functional and technical requirements for the ARIS Gateway subsystem, which provides secure, concurrent remote messaging interfaces for Telegram and Discord. The gateway allows authenticated remote users to trigger image generation pipelines, invoke specialized visual subagents, inspect engine state, and receive rendered images directly inside chat channels.

---

## Requirements

### Requirement: REQ-GW-1 - Gateway Configuration & Environment Binding

The system MUST support declarative gateway configuration in `~/.aris/config.yaml` with corresponding environment variable overrides.

#### Scenario: Default Gateway Configuration
- GIVEN a zero-config or freshly initialized ARIS installation
- WHEN `config.LoadConfig()` is executed without gateway YAML entries or environment variables
- THEN `cfg.Gateway.Concurrency` MUST default to `1`
- AND `cfg.Gateway.MaxQueue` MUST default to `10`
- AND `cfg.Gateway.Telegram.Enabled` MUST default to `false`
- AND `cfg.Gateway.Telegram.SendAsDocument` MUST default to `false`
- AND `cfg.Gateway.Discord.Enabled` MUST default to `false`
- AND all allowlists (`AllowedChatIDs`, `AllowedUserIDs`, `AllowedChannelIDs`) MUST default to empty slices.

#### Scenario: Environment Variable Overrides
- GIVEN the environment variables `TELEGRAM_BOT_TOKEN="tg-secret-123"`, `DISCORD_BOT_TOKEN="dc-secret-456"`, `TELEGRAM_ALLOWED_CHAT_IDS="1001,1002"`, `DISCORD_ALLOWED_CHANNEL_IDS="2001,2002"`, and `ARIS_GATEWAY_CONCURRENCY="2"` are set
- WHEN `config.LoadConfig()` is called
- THEN `cfg.Gateway.Telegram.BotToken` MUST equal `"tg-secret-123"`
- AND `cfg.Gateway.Telegram.Enabled` MUST equal `true`
- AND `cfg.Gateway.Discord.BotToken` MUST equal `"dc-secret-456"`
- AND `cfg.Gateway.Discord.Enabled` MUST equal `true`
- AND `cfg.Gateway.Telegram.AllowedChatIDs` MUST contain `[]int64{1001, 1002}`
- AND `cfg.Gateway.Discord.AllowedChannelIDs` MUST contain `[]string{"2001", "2002"}`
- AND `cfg.Gateway.Concurrency` MUST equal `2`.

---

### Requirement: REQ-GW-2 - Strict Ingress Access Control & Fail-Closed Authorization

Each adapter MUST validate incoming messages against configured allowlists BEFORE executing any command parsing, subagent routing, LLM reasoning, or image generation.

#### Scenario: Telegram Authorization Pass (Allowed Chat ID)
- GIVEN a Telegram adapter configured with `AllowedChatIDs: [11223344]`
- WHEN an update is received from `Chat.ID == 11223344`
- THEN the adapter MUST permit the message and proceed to command evaluation.

#### Scenario: Telegram Authorization Pass (Allowed User ID in Unlisted Group)
- GIVEN a Telegram adapter configured with `AllowedUserIDs: [998877]` and `AllowedChatIDs: []`
- WHEN an update is received from `From.ID == 998877`
- THEN the adapter MUST permit the message and proceed to command evaluation.

#### Scenario: Telegram Ingress Rejection (Unauthorized User & Chat)
- GIVEN a Telegram adapter configured with non-empty allowlists
- WHEN an update is received from an ID not present in either `AllowedChatIDs` or `AllowedUserIDs`
- THEN the adapter MUST drop the request immediately or reply with an unauthorized notification
- AND the adapter MUST NOT trigger any LLM, subagent, or image backend processing.

#### Scenario: Discord Ingress Authorization (Allowed Channel or User)
- GIVEN a Discord adapter configured with `AllowedChannelIDs: ["987654321"]` and `AllowedUserIDs: ["123456789"]`
- WHEN a `MessageCreate` event is received from `ChannelID == "987654321"` or `Author.ID == "123456789"`
- AND `Author.Bot` is `false`
- THEN the adapter MUST permit the message and proceed to command evaluation.

#### Scenario: Discord Ingress Rejection (Unauthorized Channel and User)
- GIVEN a Discord adapter with non-empty allowlists
- WHEN a `MessageCreate` event is received from an unauthorized channel and user
- THEN the adapter MUST ignore the message without sending any response or invoking backends.

---

### Requirement: REQ-GW-3 - Command Parsing & Routing Protocol

The gateway adapters MUST parse incoming user text for slash commands, prompt generation requests, and `@subagent` routing triggers.

#### Scenario: Generation Command via `/gen`
- GIVEN an authorized user sends `/gen A retrofuturistic train arriving at neon station --ratio 16:9`
- WHEN the adapter parses the input
- THEN it MUST strip the `/gen` command prefix
- AND extract the prompt `"A retrofuturistic train arriving at neon station"`
- AND set `GenerateOptions.AspectRatio` to `domain.RatioLandscape`
- AND dispatch the job to the generation queue.

#### Scenario: Subagent Trigger via `@<subagent>` Prefix
- GIVEN an authorized user sends `@director cinematic shot of a samurai in bamboo forest at dawn`
- WHEN the adapter parses the message
- THEN it MUST identify subagent routing for `"director"`
- AND invoke `AgentService.ExecuteSubagent` or `AgentService.PipelineGenerate` with the prompt
- AND return the synthesized result.

#### Scenario: Full Pipeline Generation via `/pipeline`
- GIVEN an authorized user sends `/pipeline A mysterious crystal cave with glowing flora`
- WHEN the adapter processes the command
- THEN it MUST execute `AgentService.PipelineGenerate`
- AND collect the multi-stage outcome (Director concept, PromptSmith spec, rendered image, critic score, enhancer advice).

---

### Requirement: REQ-GW-4 - Informational & Inspection Commands

The gateway adapters MUST support read-only informational commands to inspect the ARIS system state.

#### Scenario: Listing Registered Subagents via `/subagents`
- GIVEN registered subagents `@director`, `@promptsmith`, `@anime`, and `@photoreal`
- WHEN an authorized user sends `/subagents`
- THEN the adapter MUST respond with a formatted message listing all available subagent names, descriptions, and trigger examples.

#### Scenario: Listing Image Backends via `/backends`
- GIVEN configured backends `pollinations` (default), `comfyui`, and `falai`
- WHEN an authorized user sends `/backends`
- THEN the adapter MUST reply with the list of registered backends, supported models, and the active default backend.

#### Scenario: Querying Knowledge Memory via `/memory`
- GIVEN facts stored in the Knowledge Graph
- WHEN an authorized user sends `/memory cyberpunk`
- THEN the adapter MUST search memory via `AgentService.SearchMemory(ctx, "cyberpunk", "", 5)`
- AND return the matched facts formatted as a list.

#### Scenario: Inspecting Engine Status via `/status`
- GIVEN an active ARIS gateway instance
- WHEN an authorized user sends `/status`
- THEN the adapter MUST reply with the active LLM provider/model, default image backend, critic status, uptime, and current queue depth.

#### Scenario: Displaying Help via `/help`
- GIVEN an authorized user sends `/help` or `/start`
- WHEN the adapter processes the command
- THEN it MUST return a concise guide covering `/gen`, `/pipeline`, `/subagents`, `/backends`, `/memory`, `/status`, aspect ratio flags, and `@subagent` syntax.

---

### Requirement: REQ-GW-5 - Telegram Image Delivery & Progress Feedback

The Telegram adapter MUST provide real-time status feedback and upload rendered image assets according to user configuration.

#### Scenario: Standard Photo Delivery with Caption
- GIVEN a completed image generation producing a 2MB JPEG file and `SendAsDocument == false`
- WHEN the adapter delivers the result to Telegram
- THEN it MUST call `sendPhoto` with the local file path
- AND format the photo caption with prompt summary, backend, model, aspect ratio, seed, and elapsed duration.

#### Scenario: High-Resolution Uncompressed Document Delivery
- GIVEN `SendAsDocument == true` in config or a `/gen --doc` user flag
- WHEN the adapter delivers the image to Telegram
- THEN it MUST call `sendDocument` with the raw image file
- AND ensure Telegram does not apply lossy image re-compression.

#### Scenario: Long Generation Typing Action
- GIVEN an accepted generation request in progress
- WHEN the backend is actively rendering the image
- THEN the Telegram adapter MUST periodically invoke `sendChatAction` with action `upload_photo` or `typing` every 4 to 5 seconds until delivery completes.

---

### Requirement: REQ-GW-6 - Discord Image Delivery & Rich Embeds

The Discord adapter MUST deliver images as multipart attachments accompanied by structured rich embeds.

#### Scenario: Discord Attachment with Metadata Embed
- GIVEN a completed generation producing `output.jpg`
- WHEN the Discord adapter delivers the message
- THEN it MUST send a multipart message containing the image file attachment
- AND attach a Discord Embed displaying:
  - Title: Prompt or Subagent name
  - Image: `attachment://output.jpg`
  - Fields: Backend & Model, Aspect Ratio & Dimensions, Seed, Duration, Critic Score (if evaluated).

#### Scenario: Discord Typing Indicator
- GIVEN an accepted generation job
- WHEN processing begins
- THEN the Discord adapter MUST trigger `ChannelTyping` to show active progress in the channel.

---

### Requirement: REQ-GW-7 - Concurrency Control, Queuing & Rate Limiting

The gateway multiplexer MUST enforce concurrency limits using a bounded worker pool / semaphore to prevent GPU VRAM exhaustion.

#### Scenario: Concurrency Limit Enforcement
- GIVEN `Gateway.Concurrency == 1` and `Gateway.MaxQueue == 5`
- WHEN User A initiates `/gen A mountain` and User B immediately initiates `/gen A beach`
- THEN User A's job MUST execute immediately
- AND User B's job MUST be queued and start execution only after User A's job completes.

#### Scenario: Queue Overflow Rejection
- GIVEN the generation queue has reached `MaxQueue` pending jobs
- WHEN User C sends a new `/gen` request
- THEN the adapter MUST immediately reject the request with message `"Generation queue is full (X pending). Please try again shortly."`
- AND the adapter MUST NOT allocate backend resources.

---

### Requirement: REQ-GW-8 - Multiplexer Lifecycle & Graceful Shutdown

The `Multiplexer` MUST orchestrate the concurrent lifecycle of all enabled adapters and cleanly handle shutdown signals.

#### Scenario: Multi-Adapter Startup
- GIVEN both Telegram and Discord adapters are enabled with valid tokens
- WHEN `aris gateway` is launched
- THEN the Multiplexer MUST start both adapters concurrently in separate goroutines
- AND log successful connection status for each platform.

#### Scenario: Graceful Termination on SIGINT/SIGTERM
- GIVEN active gateway adapters with 1 in-flight generation job
- WHEN the process receives `SIGINT` (Ctrl+C) or `SIGTERM`
- THEN the Multiplexer MUST stop accepting new incoming updates
- AND allow up to a configured timeout (e.g., 30 seconds) for the in-flight generation job to complete and deliver its image
- AND cleanly close Telegram polling and Discord gateway sessions before process exit.

---

## Technical Specifications

### 1. Configuration Model (`internal/config/config.go`)

```go
type GatewayConfig struct {
	Concurrency int            `yaml:"concurrency"` // Max concurrent generations (default: 1)
	MaxQueue    int            `yaml:"max_queue"`   // Max pending queue size (default: 10)
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
	Enabled            bool     `yaml:"enabled"`
	BotToken           string   `yaml:"bot_token"`
	AllowedChannelIDs  []string `yaml:"allowed_channel_ids"`
	AllowedUserIDs     []string `yaml:"allowed_user_ids"`
}
```

#### Environment Variable Mapping:
- `ARIS_GATEWAY_CONCURRENCY` -> `Gateway.Concurrency` (int)
- `ARIS_GATEWAY_MAX_QUEUE` -> `Gateway.MaxQueue` (int)
- `TELEGRAM_BOT_TOKEN` / `ARIS_TELEGRAM_TOKEN` -> `Gateway.Telegram.BotToken` (string, auto-enables Telegram)
- `TELEGRAM_ALLOWED_CHAT_IDS` -> `Gateway.Telegram.AllowedChatIDs` (comma-separated int64)
- `TELEGRAM_ALLOWED_USER_IDS` -> `Gateway.Telegram.AllowedUserIDs` (comma-separated int64)
- `ARIS_TELEGRAM_SEND_DOCUMENT` -> `Gateway.Telegram.SendAsDocument` (bool)
- `DISCORD_BOT_TOKEN` / `ARIS_DISCORD_TOKEN` -> `Gateway.Discord.BotToken` (string, auto-enables Discord)
- `DISCORD_ALLOWED_CHANNEL_IDS` -> `Gateway.Discord.AllowedChannelIDs` (comma-separated string)
- `DISCORD_ALLOWED_USER_IDS` -> `Gateway.Discord.AllowedUserIDs` (comma-separated string)

---

### 2. Protocol & Payload Interaction Matrix

| Platform | Action | API Method / Event | Payload Details |
| :--- | :--- | :--- | :--- |
| **Telegram** | Ingress Polling | `getUpdates` | Long-polling offset + timeout (30s) |
| **Telegram** | Progress Heartbeat | `sendChatAction` | `action: "upload_photo"` or `"typing"` |
| **Telegram** | Standard Delivery | `sendPhoto` | `photo: InputFile(path)`, `caption: MarkdownV2/HTML` |
| **Telegram** | High-Res Delivery | `sendDocument` | `document: InputFile(path)`, `caption: MarkdownV2/HTML` |
| **Telegram** | Error / Text Reply | `sendMessage` | `chat_id`, `text`, `reply_to_message_id` |
| **Discord** | Ingress Event | `MessageCreate` | Gateway WebSocket payload, ignore `Author.Bot == true` |
| **Discord** | Progress Heartbeat | `ChannelTyping` | REST POST `/channels/{id}/typing` |
| **Discord** | Image Delivery | `ChannelMessageSendComplex` | `Files: []*File`, `Embed: *MessageEmbed` |
| **Discord** | Error / Text Reply | `ChannelMessageSend` | `channel_id`, `content` |

---

### 3. Error Handling & Edge Cases

| Failure Scenario | Detection Point | System Reaction | User Feedback |
| :--- | :--- | :--- | :--- |
| **Missing/Invalid Bot Token** | Startup / Preflight | Adapter initialization fails | Multiplexer logs error: `"adapter [telegram/discord] disabled: token missing or invalid"`. If no adapters remain enabled, gateway exits cleanly with code 1. |
| **Backend Timeout / Failure** | `AgentService.Generate` | `ctx.Done()` or backend error returned | Adapter sends error message: `"Generation failed: [error details]. Please check backend connectivity."` |
| **File Exceeds Size Limit** | Post-Render Pre-Upload | File size check (`> 50MB` Telegram doc, `> 10MB` Telegram photo, `> 25MB` Discord standard) | Adapter sends high-res document or compressed fallback; if exceeding absolute platform limit (e.g. >25MB on free Discord), returns local file path reference and size warning. |
| **Concurrency Queue Full** | Job Submission | Queue buffer capacity check | Immediate reply: `"Queue is currently full (X pending requests). Please retry in a few moments."` |
| **Unauthorized Ingress** | Ingress Evaluator | ID missing from allowlists | Silent drop or one-time unauthorized alert: `"Access denied: your User ID (X) or Channel ID (Y) is not in the ARIS allowlist."` |
| **Malformed Slash Command** | Command Parser | Invalid flags or missing prompt | Quick help message explaining expected syntax (e.g. `/gen <prompt> [--ratio 16:9] [--model flux]`). |

---

## Next Steps

1. Review and approve the specification.
2. Proceed to `sdd-design` to detail the Hexagonal gateway adapter interfaces, worker queue implementation, and CLI command integration.
