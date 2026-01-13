// Package config handles digest configuration
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is the main configuration structure
type Config struct {
	Theme           string                  `yaml:"theme"`
	ShowAtStartup   bool                    `yaml:"show_at_startup"`
	MaxItemsPerSection int                  `yaml:"max_items_per_section"`
	Sources         map[string]SourceConfig `yaml:"sources"`
}

// SourceConfig holds configuration for a single source
type SourceConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Settings map[string]any `yaml:"settings,omitempty"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Theme:           "synthwave",
		ShowAtStartup:   true,
		MaxItemsPerSection: 5,
		Sources: map[string]SourceConfig{
			"hackernews": {
				Enabled: true,
				Settings: map[string]any{
					"max": 8,
				},
			},
			"rss": {
				Enabled: false,
				Settings: map[string]any{
					"feeds": []map[string]any{
						{
							"name": "TechCrunch",
							"url":  "https://techcrunch.com/feed/",
							"max":  5,
						},
					},
				},
			},
		},
	}
}

// Load reads configuration from file, or returns default if not found
func Load() (*Config, error) {
	configPath := getConfigPath()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save writes configuration to file
func Save(cfg *Config) error {
	configPath := getConfigPath()

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	// Check XDG config first
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "digest", "digest.yaml")
	}

	// Fall back to ~/.config
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "digest", "digest.yaml")
}

// Init creates a default config file if it doesn't exist
func Init() error {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); err == nil {
		// Config already exists
		return nil
	}

	return Save(Default())
}
