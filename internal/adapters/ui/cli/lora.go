package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"aris/internal/core/domain"
)

// ParseLoRAFlags parses CLI flags for LoRA stacking (e.g. "name:scale" or "name").
func ParseLoRAFlags(flags []string) ([]domain.LoRAConfig, error) {
	var configs []domain.LoRAConfig
	for _, f := range flags {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		// Split multiple comma-separated items if provided
		parts := strings.Split(f, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			tokens := strings.Split(part, ":")
			if len(tokens) == 1 {
				name := strings.TrimSpace(tokens[0])
				if name == "" {
					return nil, fmt.Errorf("invalid lora format %q: name cannot be empty", part)
				}
				configs = append(configs, domain.LoRAConfig{
					Name:  name,
					Scale: 1.0,
				})
			} else if len(tokens) == 2 {
				name := strings.TrimSpace(tokens[0])
				if name == "" {
					return nil, fmt.Errorf("invalid lora format %q: name cannot be empty", part)
				}
				scale, err := strconv.ParseFloat(strings.TrimSpace(tokens[1]), 64)
				if err != nil {
					return nil, fmt.Errorf("invalid lora scale in %q: %w", part, err)
				}
				configs = append(configs, domain.LoRAConfig{
					Name:  name,
					Scale: scale,
				})
			} else {
				return nil, fmt.Errorf("invalid lora format %q (expected <name>:<scale> or <name>)", part)
			}
		}
	}
	return configs, nil
}

// handleLoRA processes `aris lora [list|help]` subcommands.
func (r *Runner) handleLoRA(args []string) int {
	if len(args) == 0 {
		args = []string{"list"}
	}

	subcmd := strings.ToLower(args[0])
	switch subcmd {
	case "list", "ls":
		return r.listLoRAs()
	case "help", "-h", "--help":
		r.printLoRAHelp()
		return 0
	default:
		fmt.Printf("Unknown lora command: %s\n", subcmd)
		r.printLoRAHelp()
		return 1
	}
}

func (r *Runner) listLoRAs() int {
	home, _ := os.UserHomeDir()
	loraDir := filepath.Join(home, ".aris", "models", "loras")
	_ = os.MkdirAll(loraDir, 0755)

	entries, err := os.ReadDir(loraDir)
	if err != nil {
		fmt.Printf("❌ Failed to read LoRA directory %s: %v\n", loraDir, err)
		return 1
	}

	var found []string
	for _, e := range entries {
		if !e.IsDir() {
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext == ".safetensors" || ext == ".pt" || ext == ".bin" {
				found = append(found, e.Name())
			}
		}
	}

	fmt.Printf("🎨 ARIS LoRA Management (Path: %s)\n\n", loraDir)
	if len(found) == 0 {
		fmt.Println("No local LoRA weights found.")
		fmt.Println("Tip: Place your .safetensors LoRA files in ~/.aris/models/loras/ to use them locally.")
		return 0
	}

	fmt.Printf("Found %d available LoRA model(s):\n", len(found))
	for i, name := range found {
		fmt.Printf("  %d. %s\n", i+1, name)
	}
	return 0
}

func (r *Runner) printLoRAHelp() {
	fmt.Println(`Usage: aris lora [command]

Commands:
  list, ls          List local installed LoRA weight models in ~/.aris/models/loras/
  help              Show this help message

Examples:
  aris lora list
  aris gen "portrait <lora:neon_cyber:0.85>" --lora "detail_booster:0.6"`)
}
