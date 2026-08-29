# ARIS Gateway Multiplexer (Telegram & Discord)

The ARIS Gateway subsystem provides secure, concurrent remote messaging interfaces for **Telegram** and **Discord**. It allows remote users in authorized chats/channels to trigger generative image synthesis, invoke visual subagents (such as `@director`, `@promptsmith`), search the Knowledge Graph, and receive rendered images directly inside chat channels.

---

## Architecture & Features

- **Hexagonal Architecture**: Platform adapters (`internal/adapters/gateway/telegram`, `internal/adapters/gateway/discord`) are driving adapters that interact with the core engine via the `GatewayEngine` port interface.
- **Worker Pool / Semaphore Concurrency**: Incoming requests are bounded by `Gateway.Concurrency` (max active generation jobs) and `Gateway.MaxQueue` (max pending buffer) to protect GPU VRAM from saturation. Overflow requests are rejected immediately with user-friendly warnings (`ErrQueueFull`).
- **Strict Ingress Authorization**: Fail-closed allowlist evaluation per platform:
  - **Telegram**: Allowed Chat IDs (`AllowedChatIDs`) and Allowed User IDs (`AllowedUserIDs`).
  - **Discord**: Allowed Channel IDs (`AllowedChannelIDs`) and Allowed User IDs (`AllowedUserIDs`).
- **Slash Commands & Subagent Routing**:
  - `/gen <prompt> [flags]`: Direct generation with optional aspect ratio (`--ratio 16:9`), model, backend, seed, negative prompt, document mode (`--doc`), and VLM critique (`--critic`).
  - `/pipeline <prompt>`: Full multi-agent reasoning pipeline (Director concept -> PromptSmith spec -> Render -> Critic evaluation -> Enhancer advice).
  - `@<subagent> <prompt>`: Direct execution of visual subagents (e.g. `@director`, `@promptsmith`).
  - `/subagents`, `/backends`, `/memory`, `/status`, `/help`: Real-time engine introspection.
- **Platform Deliveries**:
  - **Telegram**: Direct photo uploads (`sendPhoto`) with metadata captions, high-res uncompressed document delivery (`sendDocument`), and periodic typing/upload heartbeat.
  - **Discord**: Multipart attachment uploads with rich Cyberpunk embeds containing dimension, aspect ratio, model, seed, duration, and VLM score metadata.

---

## Configuration

Gateway options can be specified in `~/.aris/config.yaml` or via environment variables:

```yaml
gateway:
  concurrency: 2      # Max concurrent generations (default: 1)
  max_queue: 10       # Max pending queue capacity (default: 10)
  telegram:
    enabled: true
    bot_token: "YOUR_TELEGRAM_BOT_TOKEN"
    allowed_chat_ids:
      - 100123456789
    allowed_user_ids:
      - 987654321
    send_as_document: false
  discord:
    enabled: true
    bot_token: "YOUR_DISCORD_BOT_TOKEN"
    allowed_channel_ids:
      - "123456789012345678"
    allowed_user_ids:
      - "987654321098765432"
```

### Environment Variable Overrides

| Environment Variable | Description | Default |
| :--- | :--- | :--- |
| `ARIS_GATEWAY_CONCURRENCY` | Maximum concurrent generations running simultaneously | `1` |
| `ARIS_GATEWAY_MAX_QUEUE` | Maximum requests waiting in queue before rejection | `10` |
| `TELEGRAM_BOT_TOKEN` / `ARIS_TELEGRAM_TOKEN` | Telegram Bot API Token (auto-enables Telegram adapter) | `""` |
| `TELEGRAM_ALLOWED_CHAT_IDS` | Comma-separated list of allowed Telegram chat IDs | `""` |
| `TELEGRAM_ALLOWED_USER_IDS` | Comma-separated list of allowed Telegram user IDs | `""` |
| `ARIS_TELEGRAM_SEND_DOCUMENT` | Always send uncompressed files on Telegram (`true`/`false`) | `false` |
| `DISCORD_BOT_TOKEN` / `ARIS_DISCORD_TOKEN` | Discord Bot Token (auto-enables Discord adapter) | `""` |
| `DISCORD_ALLOWED_CHANNEL_IDS` | Comma-separated list of allowed Discord channel IDs | `""` |
| `DISCORD_ALLOWED_USER_IDS` | Comma-separated list of allowed Discord user IDs | `""` |

---

## Running the Gateway

To start the Gateway Multiplexer:

```bash
# Using CLI command
aris gateway

# Or shorthand
aris gw
```

Upon launching, ARIS will:
1. Initialize the SQLite database and Knowledge Graph.
2. Register image backend providers (Pollinations, ComfyUI, Fal.ai, Replicate, OpenAI, HuggingFace).
3. Connect all enabled messaging adapters concurrently.
4. Listen for incoming slash commands and `@subagent` prompts.
5. Gracefully drain in-flight generation jobs when receiving `SIGINT` (Ctrl+C) or `SIGTERM`.

---

## Chat Commands & Interaction Examples

### Image Generation
```text
/gen A cyberpunk samurai in neon rain --ratio 16:9 --backend pollinations
/gen An ancient futuristic temple hidden in fog --ratio 21:9 --doc --critic
```

### Subagent Ingress
```text
@director A dark neo-noir detective looking out a rainy skyscraper window
@promptsmith: masterwork portrait of an android girl with glowing blue eyes
```

### Multi-Agent Pipeline
```text
/pipeline A surreal floating island with waterfalls descending into clouds
```

### Engine Inspection
```text
/status
/subagents
/backends
/memory cyberpunk
/help
```
