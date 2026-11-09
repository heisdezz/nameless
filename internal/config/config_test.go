package config

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadOrCreateAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	// Test LoadOrCreate with missing file
	cfg, err := LoadOrCreate(path)
	if err != nil {
		t.Fatalf("LoadOrCreate failed: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Modify and save
	cfg.Workers = 5
	cfg.Delay = 2 * time.Second
	cfg.PerVideoTimeout = 10 * time.Minute
	cfg.Theme.AccentHex = "#ff0000"

	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify
	cfg2, err := Load(path)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if cfg2.Workers != 5 {
		t.Errorf("expected workers=5, got %d", cfg2.Workers)
	}
	if cfg2.Delay != 2*time.Second {
		t.Errorf("expected delay=2s, got %v", cfg2.Delay)
	}
	if cfg2.PerVideoTimeout != 10*time.Minute {
		t.Errorf("expected timeout=10m, got %v", cfg2.PerVideoTimeout)
	}
	if cfg2.Theme.AccentHex != "#ff0000" {
		t.Errorf("expected accent #ff0000, got %s", cfg2.Theme.AccentHex)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Workers != 3 {
		t.Errorf("expected default workers (concurrent downloads)=3, got %d", cfg.Workers)
	}
	if cfg.ConnectionsPerDownload != 5 {
		t.Errorf("expected default connections_per_download=5, got %d", cfg.ConnectionsPerDownload)
	}
	if cfg.EffectiveConcurrentDownloads() != 3 {
		t.Errorf("expected effective concurrent=3, got %d", cfg.EffectiveConcurrentDownloads())
	}
	// Max concurrent downloads is 5; legacy values clamp for runtime use
	// but must fail Validate.
	cfg.Workers = 6
	if cfg.EffectiveConcurrentDownloads() != 5 {
		t.Errorf("expected effective concurrent clamped to 5, got %d", cfg.EffectiveConcurrentDownloads())
	}
	if err := cfg.Validate(); err == nil {
		t.Errorf("expected Validate to reject workers=6 (> max 5)")
	}
	cfg.Workers = 3
	if cfg.Delay != time.Second {
		t.Errorf("expected default delay=1s, got %v", cfg.Delay)
	}
	if cfg.PerVideoTimeout != 5*time.Minute {
		t.Errorf("expected default per_video_timeout=5m, got %v", cfg.PerVideoTimeout)
	}
	if cfg.Theme.AccentHex != "#fafafa" {
		t.Errorf("expected default accent #fafafa, got %s", cfg.Theme.AccentHex)
	}
	if cfg.DefaultDownloadLocation != "./downloads" {
		t.Errorf("expected default download location ./downloads, got %s", cfg.DefaultDownloadLocation)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("default config should validate: %v", err)
	}
}
