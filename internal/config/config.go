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

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`
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

	cfg.Database.Path = filepath.Join(arisDir, "aris.db")
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

	return cfg, nil
}
