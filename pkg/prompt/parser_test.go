package prompt_test

import (
	"testing"

	"aris/internal/core/domain"
	"aris/pkg/prompt"
)

func TestParsePromptLoRAs(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expectedClean string
		expectedLoRAs []domain.LoRAConfig
	}{
		{
			name:          "no loras in prompt",
			input:         "a portrait of a cyberpunk girl in Tokyo",
			expectedClean: "a portrait of a cyberpunk girl in Tokyo",
			expectedLoRAs: nil,
		},
		{
			name:          "single lora with explicit scale",
			input:         "a cyberpunk portrait <lora:neon_cyber:0.85> in Tokyo",
			expectedClean: "a cyberpunk portrait in Tokyo",
			expectedLoRAs: []domain.LoRAConfig{
				{Name: "neon_cyber", Scale: 0.85},
			},
		},
		{
			name:          "single lora with default scale (no colon)",
			input:         "a retro style character <lora:retro_anime>",
			expectedClean: "a retro style character",
			expectedLoRAs: []domain.LoRAConfig{
				{Name: "retro_anime", Scale: 1.0},
			},
		},
		{
			name:          "multiple loras with explicit and default scales",
			input:         "a portrait <lora:neon_cyber:0.85> <lora:detail_booster:0.6> of a hero <lora:vibrant_colors> in city",
			expectedClean: "a portrait of a hero in city",
			expectedLoRAs: []domain.LoRAConfig{
				{Name: "neon_cyber", Scale: 0.85},
				{Name: "detail_booster", Scale: 0.6},
				{Name: "vibrant_colors", Scale: 1.0},
			},
		},
		{
			name:          "case insensitive tag",
			input:         "masterpiece <LoRA:StudioLighting:1.2>",
			expectedClean: "masterpiece",
			expectedLoRAs: []domain.LoRAConfig{
				{Name: "StudioLighting", Scale: 1.2},
			},
		},
		{
			name:          "tag with extra spaces and punctuation around it",
			input:         "photo of astronaut, <lora:space_suit:0.75>, high quality",
			expectedClean: "photo of astronaut, high quality",
			expectedLoRAs: []domain.LoRAConfig{
				{Name: "space_suit", Scale: 0.75},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cleanPrompt, loras := prompt.ExtractLoRAs(tc.input)
			if cleanPrompt != tc.expectedClean {
				t.Errorf("expected clean prompt %q, got %q", tc.expectedClean, cleanPrompt)
			}
			if len(loras) != len(tc.expectedLoRAs) {
				t.Fatalf("expected %d loras, got %d", len(tc.expectedLoRAs), len(loras))
			}
			for i, exp := range tc.expectedLoRAs {
				if loras[i].Name != exp.Name {
					t.Errorf("lora %d: expected name %q, got %q", i, exp.Name, loras[i].Name)
				}
				if loras[i].Scale != exp.Scale {
					t.Errorf("lora %d: expected scale %f, got %f", i, exp.Scale, loras[i].Scale)
				}
			}
		})
	}
}

func TestMergeLoRAs(t *testing.T) {
	base := []domain.LoRAConfig{
		{Name: "character_v2", Scale: 0.9},
		{Name: "neon_cyber", Scale: 0.8},
	}
	cliFlags := []domain.LoRAConfig{
		{Name: "neon_cyber", Scale: 1.2}, // should override or replace
		{Name: "style_vintage", Scale: 0.5},
	}

	merged := prompt.MergeLoRAs(base, cliFlags)
	if len(merged) != 3 {
		t.Fatalf("expected 3 merged loras, got %d", len(merged))
	}

	expected := map[string]float64{
		"character_v2":  0.9,
		"neon_cyber":    1.2,
		"style_vintage": 0.5,
	}

	for _, l := range merged {
		expScale, ok := expected[l.Name]
		if !ok {
			t.Errorf("unexpected lora name %q", l.Name)
		}
		if l.Scale != expScale {
			t.Errorf("lora %s: expected scale %f, got %f", l.Name, expScale, l.Scale)
		}
	}
}
