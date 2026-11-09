package main

import (
	"os"
	"path/filepath"
	"testing"

	"nameless/internal/config"
	"nameless/internal/tikdownloader"
	"nameless/internal/ytdlp"
)

func TestAuthArgsPrecedence(t *testing.T) {
	cfg := config.Default()
	if got := ytdlp.AuthArgs(cfg); len(got) != 0 {
		t.Fatalf("default cfg should have no auth args, got %v", got)
	}
	cfg.SessionID = "abc123"
	if got := ytdlp.AuthArgs(cfg); len(got) != 2 || got[0] != "--add-header" {
		t.Fatalf("sessionid args: %v", got)
	}
	cfg.Cookie = "/tmp/cookies.txt"
	if got := ytdlp.AuthArgs(cfg); len(got) != 2 || got[0] != "--cookies" {
		t.Fatalf("cookie file should win over sessionid: %v", got)
	}
	cfg.CookiesFromBrowser = "firefox"
	if got := ytdlp.AuthArgs(cfg); len(got) != 2 || got[0] != "--cookies-from-browser" {
		t.Fatalf("browser should win: %v", got)
	}
}

func TestEffectiveTikFormat(t *testing.T) {
	cfg := config.Default() // TikFormat mp4 (default), VideoQuality best
	if got := effectiveTikFormat(cfg); got != "mp4" {
		t.Fatalf("default: got %q", got)
	}
	cfg.VideoQuality = "720p"
	if got := effectiveTikFormat(cfg); got != "720p" {
		t.Fatalf("quality should drive format when tik format is default: got %q", got)
	}
	cfg.TikDownloaderFormat = "1080p"
	if got := effectiveTikFormat(cfg); got != "1080p" {
		t.Fatalf("explicit tik format should win: got %q", got)
	}
}

func TestRenameWithTemplate(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "tiktok_123.mp4")
	if err := os.WriteFile(src, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.NamingTemplate = "%(upload_date)s_%(id)s_%(title)s.%(ext)s"
	a := &app{cfg: cfg}
	got := a.renameWithTemplate(cfg, src, tikdownloader.VideoMeta{ID: "123", Title: "Hi There", UploadDate: "20261109"})
	if filepath.Base(got) != "20261109_123_Hi_There.mp4" {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Fatalf("renamed file missing: %v", err)
	}
	// Empty template keeps the original name.
	if got2 := a.renameWithTemplate(&config.Config{}, src, tikdownloader.VideoMeta{ID: "1"}); got2 != src {
		t.Fatalf("empty template should keep path, got %q", got2)
	}
}

func TestDownloadSidecarsNoopWhenDisabled(t *testing.T) {
	cfg := config.Default() // WriteThumbnail true by default!
	cfg.WriteThumbnail = false
	a := &app{cfg: cfg}
	if err := a.downloadSidecars(cfg, "https://example.com/v", "/tmp/x.mp4"); err != nil {
		t.Fatalf("disabled sidecars should be a no-op: %v", err)
	}
}
