// Package config handles hotbrew configuration
package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the main configuration structure
type Config struct {
	Theme              string                  `yaml:"theme"`
	Profile            string                  `yaml:"profile,omitempty"`
	ShowAtStartup      bool                    `yaml:"show_at_startup"`
	MaxItemsPerSection int                     `yaml:"max_items_per_section"`
	Sources            map[string]SourceConfig `yaml:"sources"`
	CustomTheme        *CustomThemeConfig      `yaml:"custom_theme,omitempty"`

	// TRSS settings
	DBPath       string `yaml:"db_path,omitempty"`
	SyncInterval string `yaml:"sync_interval,omitempty"` // e.g. "15m", "1h"
	DigestWindow string `yaml:"digest_window,omitempty"` // e.g. "24h", "12h"
	DigestMax    int    `yaml:"digest_max,omitempty"`    // max items in digest
	StreamLog    string `yaml:"stream_log,omitempty"`    // path to stream.log

	AI *AIConfig `yaml:"ai,omitempty"`

	X *XConfig `yaml:"x,omitempty"`

	Briefing *BriefingConfig `yaml:"briefing,omitempty"`
}

// BriefingConfig overrides the defaults in briefing.DefaultBalanceLimits.
// Zero values fall back to the built-in defaults, so users only need to
// set the knobs they want to change.
type BriefingConfig struct {
	MaxClustersPerTheme int `yaml:"max_clusters_per_theme,omitempty"`
	MaxLeadsPerDomain   int `yaml:"max_leads_per_domain,omitempty"`
	MaxTotalClusters    int `yaml:"max_total_clusters,omitempty"`
}

// XConfig holds X (Twitter) API credentials. OAuth 2.0 PKCE treats the
// client id as public, so it's safe to commit to hotbrew.yaml. The env
// var HOTBREW_X_CLIENT_ID overrides this value when set.
type XConfig struct {
	ClientID string `yaml:"client_id,omitempty"`
}

// AIConfig controls optional AI-powered summary enrichment.
// Leaving the whole section empty (or omitting Provider) disables
// AI — the app falls back to deterministic template summaries.
//
// APIKeyEnv is the name of an environment variable holding the key,
// not the key itself. Storing the literal key in hotbrew.yaml is
// possible (via APIKey) but discouraged; the env form keeps secrets
// out of dotfiles.
type AIConfig struct {
	Provider  string `yaml:"provider,omitempty"`    // "", "anthropic"
	Model     string `yaml:"model,omitempty"`       // provider-specific
	APIKey    string `yaml:"api_key,omitempty"`     // literal key (discouraged)
	APIKeyEnv string `yaml:"api_key_env,omitempty"` // env var holding the key
	MaxTokens int    `yaml:"max_tokens,omitempty"`
}

// CustomThemeConfig holds custom theme colors
type CustomThemeConfig struct {
	Primary        string   `yaml:"primary,omitempty"`
	Secondary      string   `yaml:"secondary,omitempty"`
	Accent         string   `yaml:"accent,omitempty"`
	Muted          string   `yaml:"muted,omitempty"`
	Background     string   `yaml:"background,omitempty"`
	Text           string   `yaml:"text,omitempty"`
	TextMuted      string   `yaml:"text_muted,omitempty"`
	HeaderGradient []string `yaml:"header_gradient,omitempty"`
}

// SourceConfig holds configuration for a single source
type SourceConfig struct {
	Enabled  bool           `yaml:"enabled"`
	Settings map[string]any `yaml:"settings,omitempty"`
}

// Default returns the default configuration
func Default() *Config {
	return &Config{
		Theme:              "synthwave",
		Profile:            "default",
		ShowAtStartup:      true,
		MaxItemsPerSection: 5,
		Sources: map[string]SourceConfig{
			"hackernews": {
				Enabled: true,
				Settings: map[string]any{
					"max": 8,
				},
			},
			"xbookmarks": {
				Enabled: false,
				Settings: map[string]any{
					"max": 20,
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0o600)
}

// getConfigPath returns the path to the config file
func getConfigPath() string {
	// Check XDG config first
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "hotbrew", "hotbrew.yaml")
	}

	// Fall back to ~/.config
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hotbrew", "hotbrew.yaml")
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

// GetDBPath returns the database path, with a sensible default.
func (c *Config) GetDBPath() string {
	if c.DBPath != "" {
		return c.DBPath
	}
	return filepath.Join(configDir(), "hotbrew.db")
}

// GetSyncInterval returns the sync interval as a duration.
func (c *Config) GetSyncInterval() time.Duration {
	if c.SyncInterval != "" {
		if d, err := time.ParseDuration(c.SyncInterval); err == nil {
			return d
		}
	}
	return 15 * time.Minute
}

// GetDigestWindow returns the digest window as a duration.
func (c *Config) GetDigestWindow() time.Duration {
	if c.DigestWindow != "" {
		if d, err := time.ParseDuration(c.DigestWindow); err == nil {
			return d
		}
	}
	return 24 * time.Hour
}

// GetDigestMax returns the max items per digest.
func (c *Config) GetDigestMax() int {
	if c.DigestMax > 0 {
		return c.DigestMax
	}
	return 25
}

// GetStreamLogPath returns the stream log file path.
func (c *Config) GetStreamLogPath() string {
	if c.StreamLog != "" {
		return c.StreamLog
	}
	return filepath.Join(configDir(), "stream.log")
}

// GetProfileName returns the selected profile or default.
func (c *Config) GetProfileName() string {
	if c.Profile == "" {
		return "default"
	}
	return c.Profile
}

// configDir returns the hotbrew config directory.
func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "hotbrew")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "hotbrew")
}
