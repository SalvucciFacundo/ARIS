# ARIS Configuration Reference

ARIS looks for configuration in `~/.aris/config.yaml` or a local `./config.yaml`. Configuration values can also be supplied or overridden via environment variables.

---

## Example `config.yaml`

```yaml
general:
  default_backend: "pollinations"     # pollinations, comfyui, falai, replicate, openai, huggingface
  default_model: "flux"               # flux, flux-realism, dall-e-3, sd-3.5
  default_aspect_ratio: "16:9"        # 1:1, 16:9, 9:16, 4:3, 3:4, 21:9
  save_dir: "./outputs"               # Directory where generated images are saved
  db_path: "~/.aris/aris.db"          # SQLite Knowledge Graph database path

llm:
  provider: "ollama"                  # ollama, openai, anthropic, deepseek, openrouter, groq
  model: "qwen2.5-coder"              # Model name
  base_url: "http://localhost:11434"  # Base URL for local/custom endpoints
  api_key: ""                         # LLM API key (if required)

backends:
  comfyui:
    url: "http://localhost:8188"      # ComfyUI WebSocket/REST endpoint
  falai:
    api_key: "your-fal-api-key"       # Fal.ai API key
  openai:
    api_key: "your-openai-api-key"    # OpenAI API key (for DALL-E 3 & edits)
  replicate:
    api_key: "your-replicate-api-key" # Replicate API token
  huggingface:
    api_key: "your-hf-token"          # HuggingFace token

gateway:
  concurrency: 2                      # Maximum concurrent generation workers
  max_queue: 10                       # Maximum queued requests before rejecting (ErrQueueFull)
  telegram:
    enabled: true                     # Enable Telegram bot adapter
    bot_token: "your-telegram-token"  # Telegram Bot API token from @BotFather
    allowed_chat_ids: [12345678]      # Whitelisted Telegram chat IDs (empty = all)
    allowed_user_ids: []              # Whitelisted Telegram user IDs
    send_as_document: false           # Send uncompressed PNG document instead of photo
  discord:
    enabled: true                     # Enable Discord bot adapter
    bot_token: "your-discord-token"   # Discord Bot token from Developer Portal
    allowed_channel_ids: ["987654321"]# Whitelisted Discord channel IDs
    allowed_user_ids: []              # Whitelisted Discord user IDs

ui:
  host: "127.0.0.1"                   # Web server host (use 0.0.0.0 for VPS)
  port: 8080                          # Web server port
  auth_token: ""                      # Bearer token for remote access (empty = open/local)
  auto_port: true                     # Automatically increment port if occupied
```

---

## Environment Variable Overrides

Environment variables take precedence over `config.yaml` settings:

| Environment Variable | Target Configuration Key |
|---|---|
| `ARIS_BACKEND` | `general.default_backend` |
| `ARIS_MODEL` | `general.default_model` |
| `ARIS_RATIO` | `general.default_aspect_ratio` |
| `ARIS_SAVE_DIR` | `general.save_dir` |
| `ARIS_DB_PATH` | `general.db_path` |
| `LLM_PROVIDER` | `llm.provider` |
| `LLM_MODEL` | `llm.model` |
| `LLM_BASE_URL` | `llm.base_url` |
| `LLM_API_KEY` / `OPENAI_API_KEY` | `llm.api_key` |
| `COMFYUI_URL` | `backends.comfyui.url` |
| `FAL_KEY` / `FALAI_API_KEY` | `backends.falai.api_key` |
| `REPLICATE_API_TOKEN` | `backends.replicate.api_key` |
| `HF_TOKEN` | `backends.huggingface.api_key` |
| `TELEGRAM_BOT_TOKEN` | `gateway.telegram.bot_token` (also sets `enabled=true`) |
| `TELEGRAM_ALLOWED_CHAT_IDS` | `gateway.telegram.allowed_chat_ids` (comma-separated) |
| `DISCORD_BOT_TOKEN` | `gateway.discord.bot_token` (also sets `enabled=true`) |
| `DISCORD_ALLOWED_CHANNEL_IDS` | `gateway.discord.allowed_channel_ids` (comma-separated) |
| `ARIS_GATEWAY_CONCURRENCY` | `gateway.concurrency` |
| `ARIS_WEB_TOKEN` | `ui.auth_token` |
