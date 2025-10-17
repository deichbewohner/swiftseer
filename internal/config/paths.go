package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	configDirName  = "swiftseer"
	configFileName = "config.yaml"
)

func GetConfigPath() (string, error) {
	var configDir string

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, configDirName)
	} else {
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", fmt.Errorf("failed to get home directory: %w", err)
			}
			xdgConfigHome = filepath.Join(homeDir, ".config")
		}
		configDir = filepath.Join(xdgConfigHome, configDirName)
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, configFileName), nil
}
