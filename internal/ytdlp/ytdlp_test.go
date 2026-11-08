package ytdlp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nameless/internal/config"
)

func TestDetectWithStub(t *testing.T) {
	tmpDir := t.TempDir()
	stubPath := filepath.Join(tmpDir, "yt-dlp")
	script := `#!/bin/sh
if [ "$1" = "--version" ]; then
    echo "2024.11.08"
    exit 0
fi
exit 1
`
	if err := os.WriteFile(stubPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write stub script: %v", err)
	}

	origPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+string(filepath.ListSeparator)+origPath)
	defer os.Setenv("PATH", origPath)

	bin, err := Detect()
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if !strings.Contains(bin.Version, "2024.11.08") {
		t.Errorf("expected 2024.11.08, got %q", bin.Version)
	}
}

func TestFetchProfileAndDownloadWithStub(t *testing.T) {
	tmpDir := t.TempDir()
	stubPath := filepath.Join(tmpDir, "yt-dlp")
	script := `#!/bin/sh
if [ "$1" = "--dump-json" ]; then
    echo '{"title": "test_creator", "id": "tc1", "entries": [{"id": "v1", "url": "https://tiktok.com/@tc/v1", "title": "Vid 1"}]}'
    exit 0
fi
if [ "$1" = "--newline" ]; then
    exit 0
fi
exit 0
`
	if err := os.WriteFile(stubPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write stub script: %v", err)
	}

	bin := Binary{Path: stubPath, Version: "2024.11.08"}
	fetcher := NewFetcher(bin)
	cfg := config.Default()

	var metas []VideoMeta
	err := fetcher.FetchProfile(context.Background(), cfg, "https://tiktok.com/@test_creator", func(m VideoMeta) {
		metas = append(metas, m)
	})
	if err != nil {
		t.Fatalf("FetchProfile failed: %v", err)
	}
	if len(metas) != 1 || metas[0].ID != "v1" {
		t.Errorf("unexpected metas: %+v", metas)
	}

	downloader := NewDownloader(bin)
	outDir := filepath.Join(tmpDir, "out")
	os.MkdirAll(outDir, 0755)

	outPath, err := downloader.Download(context.Background(), cfg, "https://tiktok.com/@tc/v1", outDir, nil)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	if outPath == "" {
		t.Errorf("expected non-empty outPath")
	}
}

func TestNoBinaryError(t *testing.T) {
	origPath := os.Getenv("PATH")
	os.Setenv("PATH", t.TempDir()) // empty dir
	defer os.Setenv("PATH", origPath)

	_, err := Detect()
	if err == nil {
		t.Fatal("expected error when yt-dlp is missing from PATH")
	}
	if !strings.Contains(err.Error(), "yt-dlp not found") {
		t.Errorf("expected 'yt-dlp not found' in error, got: %v", err)
	}
}
