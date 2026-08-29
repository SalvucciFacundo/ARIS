package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application settings.
type Config struct {
	LLM struct {
		Provider string `yaml:"provider"` // "openai", "ollama", "openrouter", "anthropic", "passthrough"
		APIKey   string `yaml:"api_key"`
		BaseURL  string `yaml:"base_url"`
		Model    string `yaml:"model"`
	} `yaml:"llm"`

	Image struct {
		DefaultBackend string `yaml:"default_backend"` // "pollinations", "comfyui", "falai", "replicate", "openai", "huggingface"
		DefaultModel   string `yaml:"default_model"`   // "flux", "flux-realism", "turbo", "dall-e-3"
		DefaultRatio   string `yaml:"default_ratio"`   // "1:1", "16:9", "9:16"
		OutputDir      string `yaml:"output_dir"`
		ComfyUIHost    string `yaml:"comfyui_host"`    // "http://127.0.0.1:8188"
		FalKey         string `yaml:"fal_key"`
		ReplicateToken string `yaml:"replicate_token"`
		OpenAIKey      string `yaml:"openai_key"`
		HFToken        string `yaml:"hf_token"`
	} `yaml:"image"`

	Critic struct {
		Enabled   bool    `yaml:"enabled"`
		Provider  string  `yaml:"provider"` // "ollama", "openai", "openrouter"
		Model     string  `yaml:"model"`    // "qwen2.5-vl", "gpt-4o-mini"
		BaseURL   string  `yaml:"base_url"`
		APIKey    string  `yaml:"api_key"`
		Threshold float64 `yaml:"threshold"`
		AutoHeal  bool    `yaml:"auto_heal"`
	} `yaml:"critic"`

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Gateway GatewayConfig `yaml:"gateway"`
}

// GatewayConfig holds configuration for remote messaging adapters and concurrency.
type GatewayConfig struct {
	Concurrency int            `yaml:"concurrency"` // Max concurrent generations (default: 1)
	MaxQueue    int            `yaml:"max_queue"`   // Max pending queue size (default: 10)
	Telegram    TelegramConfig `yaml:"telegram"`
	Discord     DiscordConfig  `yaml:"discord"`
}

// TelegramConfig holds credentials and allowlists for Telegram bot integration.
type TelegramConfig struct {
	Enabled        bool    `yaml:"enabled"`
	BotToken       string  `yaml:"bot_token"`
	AllowedChatIDs []int64 `yaml:"allowed_chat_ids"`
	AllowedUserIDs []int64 `yaml:"allowed_user_ids"`
	SendAsDocument bool    `yaml:"send_as_document"`
}

// DiscordConfig holds credentials and allowlists for Discord bot integration.
type DiscordConfig struct {
	Enabled           bool     `yaml:"enabled"`
	BotToken          string   `yaml:"bot_token"`
	AllowedChannelIDs []string `yaml:"allowed_channel_ids"`
	AllowedUserIDs    []string `yaml:"allowed_user_ids"`
}

// DefaultConfig returns sane zero-config defaults.
func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	arisDir := filepath.Join(home, ".aris")

	cfg := &Config{}
	cfg.LLM.Provider = "passthrough"
	cfg.LLM.Model = "gpt-4o-mini"
	cfg.LLM.BaseURL = "https://api.openai.com/v1"

	cfg.Image.DefaultBackend = "pollinations"
	cfg.Image.DefaultModel = "flux"
	cfg.Image.DefaultRatio = "1:1"
	cfg.Image.OutputDir = filepath.Join(arisDir, "outputs")

	cfg.Critic.Enabled = false
	cfg.Critic.Provider = "ollama"
	cfg.Critic.Model = "qwen2.5-vl"
	cfg.Critic.BaseURL = "http://127.0.0.1:11434/v1"
	cfg.Critic.Threshold = 0.60
	cfg.Critic.AutoHeal = true

	cfg.Database.Path = filepath.Join(arisDir, "aris.db")

	cfg.Gateway.Concurrency = 1
	cfg.Gateway.MaxQueue = 10
	cfg.Gateway.Telegram.Enabled = false
	cfg.Gateway.Telegram.SendAsDocument = false
	cfg.Gateway.Telegram.AllowedChatIDs = []int64{}
	cfg.Gateway.Telegram.AllowedUserIDs = []int64{}
	cfg.Gateway.Discord.Enabled = false
	cfg.Gateway.Discord.AllowedChannelIDs = []string{}
	cfg.Gateway.Discord.AllowedUserIDs = []string{}

	return cfg
}

// LoadConfig reads config from ~/.aris/config.yaml and merges with environment variables.
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(home, ".aris", "config.yaml")
		if data, err := os.ReadFile(configPath); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("parse config.yaml: %w", err)
			}
		}
	}

	// Environment overrides
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		if cfg.LLM.Provider == "passthrough" {
			cfg.LLM.Provider = "openai"
		}
		cfg.LLM.APIKey = key
	}
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		cfg.LLM.Provider = "openrouter"
		cfg.LLM.BaseURL = "https://openrouter.ai/api/v1"
		cfg.LLM.APIKey = key
	}
	if host := os.Getenv("OLLAMA_HOST"); host != "" {
		cfg.LLM.Provider = "ollama"
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			host = "http://" + host
		}
		cfg.LLM.BaseURL = host + "/v1"
	}
	if p := os.Getenv("ARIS_LLM_PROVIDER"); p != "" {
		cfg.LLM.Provider = p
	}
	if m := os.Getenv("ARIS_LLM_MODEL"); m != "" {
		cfg.LLM.Model = m
	}
	if b := os.Getenv("ARIS_IMAGE_BACKEND"); b != "" {
		cfg.Image.DefaultBackend = b
	}
	if m := os.Getenv("ARIS_IMAGE_MODEL"); m != "" {
		cfg.Image.DefaultModel = m
	}

	if k := os.Getenv("FAL_KEY"); k != "" {
		cfg.Image.FalKey = k
	}
	if k := os.Getenv("REPLICATE_API_TOKEN"); k != "" {
		cfg.Image.ReplicateToken = k
	}
	if k := os.Getenv("HF_TOKEN"); k != "" {
		cfg.Image.HFToken = k
	}
	if k := os.Getenv("OPENAI_API_KEY"); k != "" {
		cfg.Image.OpenAIKey = k
	}
	if h := os.Getenv("COMFYUI_HOST"); h != "" {
		cfg.Image.ComfyUIHost = h
	}

	// Gateway environment overrides
	if c := os.Getenv("ARIS_GATEWAY_CONCURRENCY"); c != "" {
		var val int
		if _, err := fmt.Sscanf(c, "%d", &val); err == nil && val > 0 {
			cfg.Gateway.Concurrency = val
		}
	}
	if q := os.Getenv("ARIS_GATEWAY_MAX_QUEUE"); q != "" {
		var val int
		if _, err := fmt.Sscanf(q, "%d", &val); err == nil && val > 0 {
			cfg.Gateway.MaxQueue = val
		}
	}
	if t := os.Getenv("TELEGRAM_BOT_TOKEN"); t != "" {
		cfg.Gateway.Telegram.BotToken = t
		cfg.Gateway.Telegram.Enabled = true
	} else if t := os.Getenv("ARIS_TELEGRAM_TOKEN"); t != "" {
		cfg.Gateway.Telegram.BotToken = t
		cfg.Gateway.Telegram.Enabled = true
	}
	if e := os.Getenv("ARIS_TELEGRAM_ENABLED"); e != "" {
		cfg.Gateway.Telegram.Enabled = (e == "true" || e == "1")
	}
	if doc := os.Getenv("ARIS_TELEGRAM_SEND_DOCUMENT"); doc != "" {
		cfg.Gateway.Telegram.SendAsDocument = (doc == "true" || doc == "1")
	}
	if chats := os.Getenv("TELEGRAM_ALLOWED_CHAT_IDS"); chats != "" {
		cfg.Gateway.Telegram.AllowedChatIDs = parseInt64List(chats)
	}
	if users := os.Getenv("TELEGRAM_ALLOWED_USER_IDS"); users != "" {
		cfg.Gateway.Telegram.AllowedUserIDs = parseInt64List(users)
	}

	if d := os.Getenv("DISCORD_BOT_TOKEN"); d != "" {
		cfg.Gateway.Discord.BotToken = d
		cfg.Gateway.Discord.Enabled = true
	} else if d := os.Getenv("ARIS_DISCORD_TOKEN"); d != "" {
		cfg.Gateway.Discord.BotToken = d
		cfg.Gateway.Discord.Enabled = true
	}
	if e := os.Getenv("ARIS_DISCORD_ENABLED"); e != "" {
		cfg.Gateway.Discord.Enabled = (e == "true" || e == "1")
	}
	if channels := os.Getenv("DISCORD_ALLOWED_CHANNEL_IDS"); channels != "" {
		cfg.Gateway.Discord.AllowedChannelIDs = parseStringList(channels)
	}
	if users := os.Getenv("DISCORD_ALLOWED_USER_IDS"); users != "" {
		cfg.Gateway.Discord.AllowedUserIDs = parseStringList(users)
	}

	return cfg, nil
}

func parseInt64List(s string) []int64 {
	var result []int64
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var val int64
		if _, err := fmt.Sscanf(part, "%d", &val); err == nil {
			result = append(result, val)
		}
	}
	return result
}

func parseStringList(s string) []string {
	var result []string
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}
