package cli_test

import (
	"testing"

	"aris/internal/adapters/ui/cli"
)

func TestParseLoRAFlags(t *testing.T) {
	tests := []struct {
		name        string
		flags       []string
		expectedLen int
		expectErr   bool
	}{
		{
			name:        "valid single flag with scale",
			flags:       []string{"neon_cyber:0.85"},
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name:        "valid single flag without scale",
			flags:       []string{"retro_anime"},
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name:        "multiple flags and comma separated",
			flags:       []string{"neon:0.8", "detail:0.6,style:1.2"},
			expectedLen: 3,
			expectErr:   false,
		},
		{
			name:        "invalid scale",
			flags:       []string{"neon:not_a_number"},
			expectedLen: 0,
			expectErr:   true,
		},
		{
			name:        "invalid empty name",
			flags:       []string{":0.8"},
			expectedLen: 0,
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configs, err := cli.ParseLoRAFlags(tc.flags)
			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(configs) != tc.expectedLen {
				t.Fatalf("expected %d configs, got %d", tc.expectedLen, len(configs))
			}
		})
	}
}

func TestParseControlNetFlags(t *testing.T) {
	tests := []struct {
		name        string
		flags       []string
		expectedLen int
		expectErr   bool
	}{
		{
			name:        "valid 3 parts (type:strength:path)",
			flags:       []string{"canny:0.75:pose.png"},
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name:        "valid 2 parts (type:path)",
			flags:       []string{"depth:depth.png"},
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name:        "valid 1 part (type)",
			flags:       []string{"openpose"},
			expectedLen: 1,
			expectErr:   false,
		},
		{
			name:        "invalid strength",
			flags:       []string{"canny:invalid:pose.png"},
			expectedLen: 0,
			expectErr:   true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			configs, err := cli.ParseControlNetFlags(tc.flags)
			if tc.expectErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.expectErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(configs) != tc.expectedLen {
				t.Fatalf("expected %d configs, got %d", tc.expectedLen, len(configs))
			}
		})
	}
}
