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
	// Workers is the number of videos downloaded at once (concurrent
	// downloads). Default 3, maximum 5.
	DefaultWorkers          = 3
	MaxWorkers              = 5
	MaxConcurrentDownloads  = MaxWorkers
	DefaultDelay            = time.Second
	DefaultPerVideoTimeout  = 5 * time.Minute
	DefaultDownloadLocation = "./downloads"
	DefaultNamingTemplate   = "%(upload_date)s_%(id)s_%(title)s.%(ext)s"
	DefaultVideoQuality     = "best"
	DefaultEnvFile          = ".env"
	DefaultAccentHex        = "#fafafa"
	DefaultRadius           = float32(8) // DIP
	DefaultSpacing          = float32(4) // mygo spacing unit
	// ConnectionsPerDownload is how many parallel connections a single
	// video file may use (yt-dlp -N / Range chunks in tikdownloader).
	// This is NOT concurrent downloads. Default 5.
	DefaultConnectionsPerDownload = 5
	MaxConnectionsPerDownload     = 16
)

// ThemeConfig is the persisted theme state; internal/theme adapts it to ui.Theme.
type ThemeConfig struct {
	AccentHex string // "#4f46e5"
	Dark      bool
	Radius    float32 // DIP, default 12
	Spacing   float32 // mygo spacing unit, default 4 (3 compact … 6 roomy)
}

// Config is the full application settings file.
type Config struct {
	Cookie, SessionID, EnvFile string // as in existing config.json
	DefaultDownloadLocation    string
	NamingTemplate             string
	VideoQuality               string
	WriteThumbnail             bool
	WriteInfoJSON              bool
	WriteSubtitles             bool
	CookiesFromBrowser         string
	YTDLPPath                  string
	// Workers is the number of concurrent downloads (videos in flight at
	// once): default 3, max 5. On disk it may appear as "workers" (legacy)
	// or "concurrent_downloads" (preferred); both are kept in sync on save.
	Workers int
	// ConnectionsPerDownload is parallel connections per single file:
	// default 5. Distinct from Workers.
	ConnectionsPerDownload    int
	Delay                     time.Duration // parsed from "1s"
	PerVideoTimeout           time.Duration // parsed from "5m"
	PreferYTDLP, InstallYTDLP bool
	Theme                     ThemeConfig
	TikDownloaderAPI          string // tikdownloader.io API endpoint
	TikDownloaderCookies      string // cookies for API auth
	TikDownloaderFormat       string // format selector (mp4, hd, etc.)
}

// Default returns the settings used when config.json is absent or incomplete.
func Default() *Config {
	return &Config{
		EnvFile:                 DefaultEnvFile,
		DefaultDownloadLocation: DefaultDownloadLocation,
		NamingTemplate:          DefaultNamingTemplate,
		VideoQuality:            DefaultVideoQuality,
		WriteThumbnail:          true,
		Workers:                 DefaultWorkers,
		ConnectionsPerDownload:  DefaultConnectionsPerDownload,
		Delay:                   DefaultDelay,
		PerVideoTimeout:         DefaultPerVideoTimeout,
		Theme: ThemeConfig{
			AccentHex: DefaultAccentHex,
			Dark:      true,
			Radius:    DefaultRadius,
			Spacing:   DefaultSpacing,
		},
		TikDownloaderAPI:     "https://tikdownloader.io",
		TikDownloaderCookies: "",
		TikDownloaderFormat:  "mp4",
	}
}

// EffectiveConcurrentDownloads returns Workers clamped to 1..MaxWorkers
// (default 3), so legacy configs with workers > 5 cannot spawn more than
// 5 concurrent downloads even before Validate runs.
func (c *Config) EffectiveConcurrentDownloads() int {
	if c == nil || c.Workers < 1 {
		return DefaultWorkers
	}
	if c.Workers > MaxWorkers {
		return MaxWorkers
	}
	return c.Workers
}

// EffectiveConnectionsPerDownload returns ConnectionsPerDownload clamped
// to 1..MaxConnectionsPerDownload (default 5).
func (c *Config) EffectiveConnectionsPerDownload() int {
	if c == nil || c.ConnectionsPerDownload < 1 {
		return DefaultConnectionsPerDownload
	}
	if c.ConnectionsPerDownload > MaxConnectionsPerDownload {
		return MaxConnectionsPerDownload
	}
	return c.ConnectionsPerDownload
}

// themeFile mirrors the "theme" object in config.json.
type themeFile struct {
	Accent  string  `json:"accent"`
	Dark    bool    `json:"dark"`
	Radius  float32 `json:"radius"`
	Spacing float32 `json:"spacing"`
}

// tikFile mirrors the "tikdownloader" object in config.json.
type tikFile struct {
	APIEndpoint string `json:"api_endpoint,omitempty"`
	APICookies  string `json:"api_cookies,omitempty"`
	Format      string `json:"format,omitempty"`
}

// configFile mirrors the on-disk config.json shape (snake_case keys).
type configFile struct {
	Cookie                  string `json:"cookie,omitempty"`
	SessionID               string `json:"session_id,omitempty"`
	EnvFile                 string `json:"env_file,omitempty"`
	DefaultDownloadLocation string `json:"default_download_location"`
	NamingTemplate          string `json:"naming_template,omitempty"`
	VideoQuality            string `json:"video_quality,omitempty"`
	WriteThumbnail          bool   `json:"write_thumbnail,omitempty"`
	WriteInfoJSON           bool   `json:"write_info_json,omitempty"`
	WriteSubtitles          bool   `json:"write_subtitles,omitempty"`
	CookiesFromBrowser      string `json:"cookies_from_browser,omitempty"`
	YTDLPPath               string `json:"ytdlp_path,omitempty"`
	// Workers is the legacy key; ConcurrentDownloads is the preferred
	// name for the same value (videos in flight at once, max 5).
	Workers                int        `json:"workers"`
	ConcurrentDownloads    int        `json:"concurrent_downloads,omitempty"`
	ConnectionsPerDownload int        `json:"connections_per_download"`
	Delay                  string     `json:"delay"`
	PerVideoTimeout        string     `json:"per_video_timeout"`
	PreferYTDLP            bool       `json:"prefer_ytdlp,omitempty"`
	InstallYTDLP           bool       `json:"install_ytdlp,omitempty"`
	Theme                  *themeFile `json:"theme,omitempty"`
	TikDownloader          *tikFile   `json:"tikdownloader,omitempty"`
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

// Validate checks ranges: Workers (concurrent downloads) 1..5,
// ConnectionsPerDownload 1..16, durations > 0.
func (c *Config) Validate() error {
	if c.Workers < 1 || c.Workers > MaxWorkers {
		return fmt.Errorf("config: workers (concurrent downloads) must be between 1 and %d, got %d", MaxWorkers, c.Workers)
	}
	if c.ConnectionsPerDownload < 1 || c.ConnectionsPerDownload > MaxConnectionsPerDownload {
		return fmt.Errorf("config: connections_per_download must be between 1 and %d, got %d", MaxConnectionsPerDownload, c.ConnectionsPerDownload)
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

	// Preferred key wins when both are present and different.
	if f.ConcurrentDownloads != 0 {
		cfg.Workers = f.ConcurrentDownloads
	} else if f.Workers != 0 {
		cfg.Workers = f.Workers
	}
	if f.ConnectionsPerDownload != 0 {
		cfg.ConnectionsPerDownload = f.ConnectionsPerDownload
	}
	if f.TikDownloader != nil {
		if f.TikDownloader.APIEndpoint != "" {
			cfg.TikDownloaderAPI = f.TikDownloader.APIEndpoint
		}
		if f.TikDownloader.APICookies != "" {
			cfg.TikDownloaderCookies = f.TikDownloader.APICookies
		}
		if f.TikDownloader.Format != "" {
			cfg.TikDownloaderFormat = f.TikDownloader.Format
		}
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
	if f.NamingTemplate != "" {
		cfg.NamingTemplate = f.NamingTemplate
	}
	if f.VideoQuality != "" {
		cfg.VideoQuality = f.VideoQuality
	}
	cfg.WriteThumbnail = f.WriteThumbnail
	cfg.WriteInfoJSON = f.WriteInfoJSON
	cfg.WriteSubtitles = f.WriteSubtitles
	if f.CookiesFromBrowser != "" {
		cfg.CookiesFromBrowser = f.CookiesFromBrowser
	}
	if f.YTDLPPath != "" {
		cfg.YTDLPPath = f.YTDLPPath
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
		NamingTemplate:          c.NamingTemplate,
		VideoQuality:            c.VideoQuality,
		WriteThumbnail:          c.WriteThumbnail,
		WriteInfoJSON:           c.WriteInfoJSON,
		WriteSubtitles:          c.WriteSubtitles,
		CookiesFromBrowser:      c.CookiesFromBrowser,
		YTDLPPath:               c.YTDLPPath,
		Workers:                 c.Workers,
		ConcurrentDownloads:     c.Workers,
		ConnectionsPerDownload:  c.ConnectionsPerDownload,
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
		TikDownloader: &tikFile{
			APIEndpoint: c.TikDownloaderAPI,
			APICookies:  c.TikDownloaderCookies,
			Format:      c.TikDownloaderFormat,
		},
	}
}
