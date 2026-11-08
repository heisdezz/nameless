// Package config persists user settings in config.json.
//
// Owned by workstream A (progress.md). Contract: docs/implementation-plan.md §3.
//
// The on-disk shape follows the pre-existing repo-root config.json: snake_case
// keys with durations as time.ParseDuration strings ("1s", "5m"), plus the
// "theme" object written by Save. A missing file (or missing keys) yields the
// defaults below, so partial or legacy configs always load into a valid Config.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Defaults used when config.json (or individual keys) are absent.
const (
	DefaultWorkers          = 3
	DefaultDelay            = time.Second
	DefaultPerVideoTimeout  = 5 * time.Minute
	DefaultDownloadLocation = "./downloads"
	DefaultEnvFile          = ".env"
	DefaultAccentHex        = "#4f46e5"
	DefaultRadius           = float32(12) // DIP
	DefaultSpacing          = float32(4)  // mygo spacing unit
)

// ThemeConfig is the persisted theme state; internal/theme adapts it to ui.Theme.
type ThemeConfig struct {
	AccentHex string  // "#4f46e5"
	Dark      bool
	Radius    float32 // DIP, default 12
	Spacing   float32 // mygo spacing unit, default 4 (3 compact … 6 roomy)
}

// Config is the full application settings file.
type Config struct {
	Cookie, SessionID, EnvFile string // as in existing config.json
	DefaultDownloadLocation    string
	Workers                    int           // default 3
	Delay                      time.Duration // parsed from "1s"
	PerVideoTimeout            time.Duration // parsed from "5m"
	PreferYTDLP, InstallYTDLP  bool
	Theme                      ThemeConfig
}

// Default returns the settings used when config.json is absent or incomplete.
func Default() *Config {
	return &Config{
		EnvFile:                 DefaultEnvFile,
		DefaultDownloadLocation: DefaultDownloadLocation,
		Workers:                 DefaultWorkers,
		Delay:                   DefaultDelay,
		PerVideoTimeout:         DefaultPerVideoTimeout,
		Theme: ThemeConfig{
			AccentHex: DefaultAccentHex,
			Radius:    DefaultRadius,
			Spacing:   DefaultSpacing,
		},
	}
}

// themeFile mirrors the "theme" object in config.json.
type themeFile struct {
	Accent  string  `json:"accent"`
	Dark    bool    `json:"dark"`
	Radius  float32 `json:"radius"`
	Spacing float32 `json:"spacing"`
}

// configFile mirrors the on-disk config.json shape (snake_case keys).
type configFile struct {
	Cookie                  string     `json:"cookie"`
	SessionID               string     `json:"session_id"`
	EnvFile                 string     `json:"env_file"`
	DefaultDownloadLocation string     `json:"default_download_location"`
	Workers                 int        `json:"workers"`
	Delay                   string     `json:"delay"`
	PerVideoTimeout         string     `json:"per_video_timeout"`
	PreferYTDLP             bool       `json:"prefer_ytdlp"`
	InstallYTDLP            bool       `json:"install_ytdlp"`
	Theme                   *themeFile `json:"theme,omitempty"`
}

// Load reads path; a missing file yields defaults, not an error.
// An empty or truncated file (e.g. after a crash mid-write) also yields defaults.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return nil, fmt.Errorf("config: read %s: %w", path, err)
	}
	return parse(raw, path)
}

// LoadOrCreate reads path, writing defaults first if the file is absent.
func LoadOrCreate(path string) (*Config, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return Load(path)
	case errors.Is(err, os.ErrNotExist):
		cfg := Default()
		if err := cfg.Save(path); err != nil {
			return nil, err
		}
		return cfg, nil
	default:
		return nil, fmt.Errorf("config: stat %s: %w", path, err)
	}
}

// Save writes the config in the config.json shape (snake_case keys + theme).
func (c *Config) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("config: create directory for %s: %w", path, err)
	}
	raw, err := json.MarshalIndent(c.toFile(), "", "  ")
	if err != nil {
		return fmt.Errorf("config: encode: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("config: write %s: %w", path, err)
	}
	return nil
}

// Validate checks ranges: Workers 1..16, durations > 0.
func (c *Config) Validate() error {
	if c.Workers < 1 || c.Workers > 16 {
		return fmt.Errorf("config: workers must be between 1 and 16, got %d", c.Workers)
	}
	if c.Delay <= 0 {
		return fmt.Errorf("config: delay must be positive, got %s", c.Delay)
	}
	if c.PerVideoTimeout <= 0 {
		return fmt.Errorf("config: per_video_timeout must be positive, got %s", c.PerVideoTimeout)
	}
	return nil
}

// parse overlays the file values onto the defaults: absent keys (or empty
// strings) keep the default, so legacy configs without a theme section still
// load into a fully populated Config.
func parse(raw []byte, path string) (*Config, error) {
	if len(strings.TrimSpace(string(raw))) == 0 {
		return Default(), nil
	}
	var f configFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", path, err)
	}
	cfg := Default()

	if f.Workers != 0 {
		cfg.Workers = f.Workers
	}
	if f.Delay != "" {
		v, err := time.ParseDuration(f.Delay)
		if err != nil {
			return nil, fmt.Errorf("config: %s: parse delay: %w", path, err)
		}
		cfg.Delay = v
	}
	if f.PerVideoTimeout != "" {
		v, err := time.ParseDuration(f.PerVideoTimeout)
		if err != nil {
			return nil, fmt.Errorf("config: %s: parse per_video_timeout: %w", path, err)
		}
		cfg.PerVideoTimeout = v
	}

	// Strings: non-empty values override defaults so partial configs keep
	// usable fallbacks for keys they predate.
	if f.Cookie != "" {
		cfg.Cookie = f.Cookie
	}
	if f.SessionID != "" {
		cfg.SessionID = f.SessionID
	}
	if f.EnvFile != "" {
		cfg.EnvFile = f.EnvFile
	}
	if f.DefaultDownloadLocation != "" {
		cfg.DefaultDownloadLocation = f.DefaultDownloadLocation
	}
	cfg.PreferYTDLP = f.PreferYTDLP
	cfg.InstallYTDLP = f.InstallYTDLP

	if t := f.Theme; t != nil {
		if t.Accent != "" {
			cfg.Theme.AccentHex = t.Accent
		}
		cfg.Theme.Dark = t.Dark
		if t.Radius != 0 {
			cfg.Theme.Radius = t.Radius
		}
		if t.Spacing != 0 {
			cfg.Theme.Spacing = t.Spacing
		}
	}
	return cfg, nil
}

func (c *Config) toFile() configFile {
	return configFile{
		Cookie:                  c.Cookie,
		SessionID:               c.SessionID,
		EnvFile:                 c.EnvFile,
		DefaultDownloadLocation: c.DefaultDownloadLocation,
		Workers:                 c.Workers,
		Delay:                   c.Delay.String(),
		PerVideoTimeout:         c.PerVideoTimeout.String(),
		PreferYTDLP:             c.PreferYTDLP,
		InstallYTDLP:            c.InstallYTDLP,
		Theme: &themeFile{
			Accent:  c.Theme.AccentHex,
			Dark:    c.Theme.Dark,
			Radius:  c.Theme.Radius,
			Spacing: c.Theme.Spacing,
		},
	}
}
