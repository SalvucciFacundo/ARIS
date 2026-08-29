package gateway_test

import (
	"testing"

	"aris/internal/adapters/gateway"
	"aris/internal/core/domain"
)

func TestParseMessage_SlashCommands(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedCmd  gateway.CommandType
		expectedText string
		checkOptions func(t *testing.T, opts gateway.ParsedOptions)
	}{
		{
			name:         "Simple /gen",
			input:        "/gen a cute cyberpunk cat",
			expectedCmd:  gateway.CmdGen,
			expectedText: "a cute cyberpunk cat",
			checkOptions: func(t *testing.T, opts gateway.ParsedOptions) {
				if opts.GenOpts.AspectRatio != domain.RatioSquare {
					t.Errorf("expected ratio 1:1, got %s", opts.GenOpts.AspectRatio)
				}
				if opts.SendAsDocument {
					t.Errorf("expected SendAsDocument=false")
				}
			},
		},
		{
			name:         "/gen with flags --ratio 16:9 --backend comfyui --doc",
			input:        "/gen neon skyscraper --ratio 16:9 --backend comfyui --model flux-realism --seed 42 --doc --critic",
			expectedCmd:  gateway.CmdGen,
			expectedText: "neon skyscraper",
			checkOptions: func(t *testing.T, opts gateway.ParsedOptions) {
				if opts.GenOpts.AspectRatio != domain.RatioLandscape {
					t.Errorf("expected ratio 16:9, got %s", opts.GenOpts.AspectRatio)
				}
				if opts.GenOpts.Backend != "comfyui" {
					t.Errorf("expected backend comfyui, got %s", opts.GenOpts.Backend)
				}
				if opts.GenOpts.Model != "flux-realism" {
					t.Errorf("expected model flux-realism, got %s", opts.GenOpts.Model)
				}
				if opts.GenOpts.Seed != 42 {
					t.Errorf("expected seed 42, got %d", opts.GenOpts.Seed)
				}
				if !opts.SendAsDocument {
					t.Errorf("expected SendAsDocument=true")
				}
				if !opts.GenOpts.EnableCritic {
					t.Errorf("expected EnableCritic=true")
				}
			},
		},
		{
			name:         "/pipeline command",
			input:        "/pipeline A glowing crystal cave",
			expectedCmd:  gateway.CmdPipeline,
			expectedText: "A glowing crystal cave",
		},
		{
			name:         "@director subagent routing",
			input:        "@director cinematic samurai in bamboo forest",
			expectedCmd:  gateway.CmdSubagent,
			expectedText: "cinematic samurai in bamboo forest",
			checkOptions: func(t *testing.T, opts gateway.ParsedOptions) {
				if opts.SubagentName != "director" {
					t.Errorf("expected subagent 'director', got %q", opts.SubagentName)
				}
			},
		},
		{
			name:         "/subagents list",
			input:        "/subagents",
			expectedCmd:  gateway.CmdSubagents,
			expectedText: "",
		},
		{
			name:         "/backends list",
			input:        "/backends",
			expectedCmd:  gateway.CmdBackends,
			expectedText: "",
		},
		{
			name:         "/memory query",
			input:        "/memory cyberpunk aesthetic",
			expectedCmd:  gateway.CmdMemory,
			expectedText: "cyberpunk aesthetic",
		},
		{
			name:         "/status",
			input:        "/status",
			expectedCmd:  gateway.CmdStatus,
			expectedText: "",
		},
		{
			name:         "/help",
			input:        "/help",
			expectedCmd:  gateway.CmdHelp,
			expectedText: "",
		},
		{
			name:         "/start",
			input:        "/start",
			expectedCmd:  gateway.CmdHelp,
			expectedText: "",
		},
		{
			name:         "Plain text fallback to generation",
			input:        "a peaceful zen garden at sunrise",
			expectedCmd:  gateway.CmdGen,
			expectedText: "a peaceful zen garden at sunrise",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := gateway.ParseMessage(tt.input)
			if parsed.Type != tt.expectedCmd {
				t.Fatalf("expected command %v, got %v", tt.expectedCmd, parsed.Type)
			}
			if parsed.CleanPrompt != tt.expectedText {
				t.Errorf("expected clean prompt %q, got %q", tt.expectedText, parsed.CleanPrompt)
			}
			if tt.checkOptions != nil {
				tt.checkOptions(t, parsed.Options)
			}
		})
	}
}
