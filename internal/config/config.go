package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Config struct {
    Username    string `koanf:"username"`
    Group       string `koanf:"group"`
    Environment string `koanf:"environment"` // "production", "staging", "development"
    AccessToken      string    `koanf:"access_token"`
    RefreshToken     string    `koanf:"refresh_token"`
    TokenExpiresAt   time.Time `koanf:"token_expires_at"`
    RefreshExpiresAt time.Time `koanf:"refresh_expires_at"`
}

type Manager struct {
    k          *koanf.Koanf
    configPath string
}

func NewManager() (*Manager, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	k := koanf.New(".")
	return &Manager{
		k:          k,
		configPath: configPath,
	}, nil
}

func (m *Manager) Load() (*Config, error) {
    if _, err := os.Stat(m.configPath); err == nil {
        if err := m.k.Load(file.Provider(m.configPath), yaml.Parser()); err != nil {
            return nil, fmt.Errorf("failed to load config file: %w", err)
        }
    }
    if err := m.k.Load(env.Provider("FUTURE_", "__", func(s string) string {
        key := strings.ToUpper(strings.TrimPrefix(strings.ToLower(s), "future_"))
        switch key {
        case "USER":
            return "username"
        case "GROUP":
            return "group"
        case "ENVIRONMENT":
            return "environment"
        default:
            return "" // ignore unknowns
        }
    }), nil); err != nil {
        return nil, fmt.Errorf("failed to load environment variables: %w", err)
    }

	cfg := &Config{}
	if err := m.k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if cfg.Environment == "" {
		cfg.Environment = "production"
	}

	return cfg, nil
}

func (m *Manager) Save(cfg *Config) error {
    data := map[string]interface{}{
        "username":    cfg.Username,
        "group":       cfg.Group,
        "environment": cfg.Environment,
        // Do NOT persist access token or its expiry; only persist refresh token details
        "refresh_token":      cfg.RefreshToken,
        "refresh_expires_at": cfg.RefreshExpiresAt,
    }
    yamlData, err := yaml.Parser().Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal config: %w", err)
    }
    if err := os.WriteFile(m.configPath, yamlData, 0600); err != nil {
        return fmt.Errorf("failed to write config file: %w", err)
    }

	return nil
}

func (m *Manager) Clear() error {
    if err := os.Remove(m.configPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to remove config file: %w", err)
    }
    return nil
}

func (m *Manager) GetConfigPath() string {
    return m.configPath
}

func (c *Config) IsTokenValid() bool {
	if c.AccessToken == "" {
		return false
	}
	return time.Now().Before(c.TokenExpiresAt)
}

func (c *Config) NeedsTokenRefresh() bool {
	if c.RefreshToken == "" {
		return false
	}
    return time.Until(c.TokenExpiresAt) < 5*time.Minute
}

func (c *Config) IsRefreshTokenValid() bool {
	if c.RefreshToken == "" {
		return false
	}
	return time.Now().Before(c.RefreshExpiresAt)
}

func (c *Config) UpdateToken(accessToken, refreshToken string, expiresIn, refreshExpiresIn int) {
	c.AccessToken = accessToken
	c.RefreshToken = refreshToken
	c.TokenExpiresAt = time.Now().Add(time.Duration(expiresIn) * time.Second)
	c.RefreshExpiresAt = time.Now().Add(time.Duration(refreshExpiresIn) * time.Second)
}
