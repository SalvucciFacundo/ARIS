package services_test

import (
	"testing"

	"aris/internal/core/services"
)

func TestMatrixEngine_Expand_SingleGroup(t *testing.T) {
	engine := services.NewMatrixEngine(100, false)
	prompt := "a [cyberpunk|anime|oil painting] portrait of an astronaut"

	got, err := engine.Expand(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"a cyberpunk portrait of an astronaut",
		"a anime portrait of an astronaut",
		"a oil painting portrait of an astronaut",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d prompts, got %d: %v", len(expected), len(got), got)
	}

	for i, exp := range expected {
		if got[i] != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, got[i])
		}
	}
}

func TestMatrixEngine_Expand_MultiGroupCartesian(t *testing.T) {
	engine := services.NewMatrixEngine(100, false)
	prompt := "a [cyberpunk|steampunk] [cat|fox] in [Tokyo|London]"

	got, err := engine.Expand(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"a cyberpunk cat in Tokyo",
		"a cyberpunk cat in London",
		"a cyberpunk fox in Tokyo",
		"a cyberpunk fox in London",
		"a steampunk cat in Tokyo",
		"a steampunk cat in London",
		"a steampunk fox in Tokyo",
		"a steampunk fox in London",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d prompts, got %d: %v", len(expected), len(got), got)
	}

	for i, exp := range expected {
		if got[i] != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, got[i])
		}
	}
}

func TestMatrixEngine_Expand_EscapedBrackets(t *testing.T) {
	engine := services.NewMatrixEngine(100, false)
	prompt := `a cat wearing \[cyberpunk\] goggles in [neon|dark] alley`

	got, err := engine.Expand(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{
		"a cat wearing [cyberpunk] goggles in neon alley",
		"a cat wearing [cyberpunk] goggles in dark alley",
	}

	if len(got) != len(expected) {
		t.Fatalf("expected %d prompts, got %d: %v", len(expected), len(got), got)
	}

	for i, exp := range expected {
		if got[i] != exp {
			t.Errorf("at index %d: expected %q, got %q", i, exp, got[i])
		}
	}
}

func TestMatrixEngine_Expand_NoBrackets(t *testing.T) {
	engine := services.NewMatrixEngine(100, false)
	prompt := "a serene mountain lake at sunrise"

	got, err := engine.Expand(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 || got[0] != prompt {
		t.Fatalf("expected 1 prompt %q, got: %v", prompt, got)
	}
}

func TestMatrixEngine_Expand_SingleOptionInGroup(t *testing.T) {
	engine := services.NewMatrixEngine(100, false)
	prompt := "a photo of a [robot] dog"

	got, err := engine.Expand(prompt)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got) != 1 || got[0] != "a photo of a robot dog" {
		t.Fatalf("expected %q, got: %v", "a photo of a robot dog", got)
	}
}

func TestMatrixEngine_Expand_MaxJobsLimit(t *testing.T) {
	// 2 * 2 * 2 = 8 variants, limit is 5
	engine := services.NewMatrixEngine(5, false)
	prompt := "a [cyberpunk|steampunk] [cat|fox] in [Tokyo|London]"

	_, err := engine.Expand(prompt)
	if err == nil {
		t.Fatal("expected error when matrix size exceeds limit, got nil")
	}

	// With force=true, it should succeed
	forcedEngine := services.NewMatrixEngine(5, true)
	got, err := forcedEngine.Expand(prompt)
	if err != nil {
		t.Fatalf("unexpected error with force=true: %v", err)
	}
	if len(got) != 8 {
		t.Fatalf("expected 8 prompts, got %d", len(got))
	}
}

func TestMatrixEngine_Expand_EmptyPrompt(t *testing.T) {
	engine := services.NewMatrixEngine(100, false)
	_, err := engine.Expand("")
	if err == nil {
		t.Fatal("expected error for empty prompt, got nil")
	}

	_, err = engine.Expand("   \t\n")
	if err == nil {
		t.Fatal("expected error for whitespace prompt, got nil")
	}
}
