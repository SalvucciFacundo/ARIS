# Tasks: ARIS Gateway Multiplexer (Telegram & Discord)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1200-1500 lines |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 -> PR 2 -> PR 3 -> PR 4 |
| Delivery strategy | auto-chain |
| Chain strategy | stacked-to-main |
| Decision needed before apply | No |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

---

## PR Work Unit Breakdown

### PR 1: Substrate, Config, Interfaces, Engine Bridge & JobQueue / Concurrency Controller
*Focus: Foundation layer, configuration binding, concurrency semaphore, job queue, and core engine interfaces.*

- [x] Write unit tests for configuration loading and environment variable overrides (`ARIS_GATEWAY_CONCURRENCY`, `TELEGRAM_BOT_TOKEN`, `DISCORD_BOT_TOKEN`, allowlists) in `internal/config/config_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement Gateway configuration structs (`GatewayConfig`, `TelegramConfig`, `DiscordConfig`) and Viper environment variable bindings in `internal/config/config.go`. <!-- sdd-owner: implementation -->
- [x] Write concurrent race tests for `JobQueue` (verifying concurrency limit, queue depth bounding, and `ErrQueueFull` rejection) in `internal/adapters/gateway/queue_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement `JobQueue` and semaphore worker pool in `internal/adapters/gateway/queue.go`. <!-- sdd-owner: implementation -->
- [x] Define core Gateway interfaces (`GatewayAdapter`, `GatewayMultiplexer`, `GatewayEngine`, `GenerateOptions`, `PipelineResult`) in `internal/adapters/gateway/engine.go`. <!-- sdd-owner: implementation -->
- [x] Run `go test ./internal/config/... ./internal/adapters/gateway/...` to verify PR 1 test suite. <!-- sdd-owner: implementation -->

---

### PR 2: Telegram Adapter (Auth, Long-Polling, Commands & @subagent Ingress, Photo/Doc Delivery)
*Focus: Telegram bot integration, strict allowlist authorization, command parsing, chat actions, and photo/document delivery.*

- [x] Write table-driven unit tests for Telegram allowlist authorization (`auth.go`) covering allowed chat IDs, user IDs, and fail-closed rejection in `internal/adapters/gateway/telegram/auth_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement Telegram allowlist verification in `internal/adapters/gateway/telegram/auth.go`. <!-- sdd-owner: implementation -->
- [x] Write unit tests for command parser and subagent routing (`/gen`, `/pipeline`, `/subagents`, `@director`) in `internal/adapters/gateway/parser_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement command parser and flag extraction (e.g. `--ratio 16:9`) in `internal/adapters/gateway/parser.go`. <!-- sdd-owner: implementation -->
- [x] Write unit tests for Telegram message handlers and delivery formatters (photo vs document, captions, typing action loop) in `internal/adapters/gateway/telegram/handlers_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement Telegram adapter (`Start`, `Stop`, long-polling loop, typing heartbeat, `sendPhoto`, `sendDocument`) in `internal/adapters/gateway/telegram/adapter.go` and `handlers.go`. <!-- sdd-owner: implementation -->
- [x] Run `go test ./internal/adapters/gateway/...` to verify PR 2 integration. <!-- sdd-owner: implementation -->

---

### PR 3: Discord Adapter (Auth, Event Listener, Commands & @subagent Ingress, Attachment/Embed Delivery)
*Focus: Discord bot integration using discordgo, channel/user allowlist auth, message event handling, multipart attachments, and rich embeds.*

- [x] Write table-driven unit tests for Discord channel and user allowlist authorization (`auth.go`) in `internal/adapters/gateway/discord/auth_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement Discord allowlist authorization and bot message filtering in `internal/adapters/gateway/discord/auth.go`. <!-- sdd-owner: implementation -->
- [x] Write unit tests for Discord message event handlers, embed builder, and multipart file attachment formatting in `internal/adapters/gateway/discord/handlers_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement Discord adapter (`Start`, `Stop`, `MessageCreate` event listener, `ChannelTyping` heartbeat, `ChannelMessageSendComplex` with rich embeds) in `internal/adapters/gateway/discord/adapter.go` and `handlers.go`. <!-- sdd-owner: implementation -->
- [x] Run `go test ./internal/adapters/gateway/...` to verify PR 3 integration. <!-- sdd-owner: implementation -->

---

### PR 4: Multiplexer, CLI Subcommand `aris gateway`, E2E Integration Tests & Docs
*Focus: Multiplexing adapters concurrently, graceful shutdown on SIGINT/SIGTERM, CLI subcommand wiring, and comprehensive documentation.*

- [x] Write unit tests for `GatewayMultiplexer` lifecycle (concurrent start, graceful shutdown timeout, zero-enabled adapter fallback) in `internal/adapters/gateway/multiplexer_test.go`. <!-- sdd-owner: implementation -->
- [x] Implement `GatewayMultiplexer` in `internal/adapters/gateway/multiplexer.go`. <!-- sdd-owner: implementation -->
- [x] Implement `aris gateway` CLI subcommand wiring in `cmd/gateway.go` (or `cmd/root.go`). <!-- sdd-owner: implementation -->
- [x] Write end-to-end integration test suite simulating full message ingress -> queue -> mock engine -> delivery roundtrip in `test/integration/gateway_e2e_test.go`. <!-- sdd-owner: implementation -->
- [x] Add documentation and configuration examples for Telegram and Discord bot setup in `docs/gateway.md`. <!-- sdd-owner: implementation -->
- [x] Run full test suite `go test -race ./...` and verify clean build. <!-- sdd-owner: implementation -->
