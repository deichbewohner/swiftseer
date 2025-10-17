package config

import (
	"os"
	"testing"

	"github.com/knadh/koanf/v2"
)

func TestConfig_LoadFromEnvironment(t *testing.T) {
	_ = os.Setenv("FUTURE_USER", "testuser")
	_ = os.Setenv("FUTURE_GROUP", "testgroup")
	_ = os.Setenv("FUTURE_ENVIRONMENT", "staging")
	defer func() {
		_ = os.Unsetenv("FUTURE_USER")
		_ = os.Unsetenv("FUTURE_GROUP")
		_ = os.Unsetenv("FUTURE_ENVIRONMENT")
	}()

	// Create a temporary config directory
	tempDir := t.TempDir()
	configPath := tempDir + "/config.yaml"

	// Create manager with temp path
	manager := &Manager{
		k:          koanf.New("."),
		configPath: configPath,
	}

	// Load config (should pick up env vars)
	cfg, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Debug: print all keys
	t.Logf("Loaded config keys: %v", manager.k.All())

	// Verify env vars were loaded
	if cfg.Username != "testuser" {
		t.Errorf("Username = %s, want testuser", cfg.Username)
	}

	if cfg.Group != "testgroup" {
		t.Errorf("Group = %s, want testgroup", cfg.Group)
	}

	if cfg.Environment != "staging" {
		t.Errorf("Environment = %s, want staging", cfg.Environment)
	}
}

func TestConfig_EnvVarMapping(t *testing.T) {
	tests := []struct {
		envVar   string
		envValue string
		field    string
		want     string
	}{
		{"FUTURE_USER", "alice", "Username", "alice"},
		{"FUTURE_GROUP", "team-a", "Group", "team-a"},
		{"FUTURE_ENVIRONMENT", "production", "Environment", "production"},
	}

	for _, tt := range tests {
		t.Run(tt.envVar, func(t *testing.T) {
			_ = os.Setenv(tt.envVar, tt.envValue)
			defer func() { _ = os.Unsetenv(tt.envVar) }()

			// Create manager
			tempDir := t.TempDir()
			manager := &Manager{
				k:          koanf.New("."),
				configPath: tempDir + "/config.yaml",
			}

			// Load config
			cfg, err := manager.Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			// Check the appropriate field
			var got string
			switch tt.field {
			case "Username":
				got = cfg.Username
			case "Group":
				got = cfg.Group
			case "Environment":
				got = cfg.Environment
			}

			if got != tt.want {
				t.Errorf("%s = %s, want %s", tt.field, got, tt.want)
			}
		})
	}
}
