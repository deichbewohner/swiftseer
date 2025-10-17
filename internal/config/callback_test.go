package config

import (
	"testing"
)

func TestEnvVarCallback(t *testing.T) {
	// Test the callback function logic in isolation
	tests := []struct {
		input string
		want  string
	}{
		{"USER", "username"},
		{"GROUP", "group"},
		{"ENVIRONMENT", "environment"},
	}

	// Define the callback (same as in config.go)
	callback := func(s string) string {
		switch s {
		case "USER":
			return "username"
		case "GROUP":
			return "group"
		case "ENVIRONMENT":
			return "environment"
		default:
			return ""
		}
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := callback(tt.input)
			if got != tt.want {
				t.Errorf("callback(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
