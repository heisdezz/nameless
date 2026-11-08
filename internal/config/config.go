// Package config persists user settings in config.json.
//
// Owned by workstream A (progress.md). Contract: docs/implementation-plan.md §3.
// This is an M0 skeleton: bodies return ErrNotImplemented until A1/A2 land.
package config

import (
	"errors"
	"time"
)

// ErrNotImplemented marks skeleton bodies; workstream A replaces them.
var ErrNotImplemented = errors.New("config: not implemented (workstream A, progress.md A1/A2)")

type ThemeConfig struct {
	AccentHex string // "#4f46e5"
	Dark      bool
	Radius    float32 // DIP, default 12
	Spacing   float32 // mygo spacing unit, default 4 (3 compact … 6 roomy)
}

type Config struct {
	Cookie, SessionID, EnvFile string
	DefaultDownloadLocation    string
	Workers                    int
	Delay                      time.Duration
	PerVideoTimeout            time.Duration
	PreferYTDLP, InstallYTDLP  bool
	Theme                      ThemeConfig
}

// Load reads path; a missing file yields defaults, not an error.
func Load(path string) (*Config, error) { return nil, ErrNotImplemented }

// LoadOrCreate reads path, writing defaults first if the file is absent.
func LoadOrCreate(path string) (*Config, error) { return nil, ErrNotImplemented }

func (c *Config) Save(path string) error { return ErrNotImplemented }

// Validate checks ranges: Workers 1..16, durations > 0.
func (c *Config) Validate() error { return ErrNotImplemented }
