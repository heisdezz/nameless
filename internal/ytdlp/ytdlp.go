// Package ytdlp wraps the yt-dlp binary: detection, profile fetching,
// single-video download with progress.
//
// Owned by workstream B (progress.md). Contract: docs/implementation-plan.md §3.
// M0 skeleton: bodies return ErrNotImplemented until B1–B3 land.
package ytdlp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"nameless/internal/config"
	"os/exec"
	"strings"
	"time"
)

// ErrNotImplemented marks skeleton bodies; workstream B replaces them.
var ErrNotImplemented = errors.New("ytdlp: not implemented (workstream B, progress.md B1–B3)")

// NoBinaryError reports that yt-dlp could not be found; the UI maps it to
// the actionable "missing yt-dlp" state.
type NoBinaryError struct{ Detail string }

func (e *NoBinaryError) Error() string {
	if e.Detail == "" {
		return "yt-dlp not found"
	}
	return "yt-dlp not found: " + e.Detail
}

type Binary struct{ Path, Version string }

// Detect looks yt-dlp up on PATH and runs "yt-dlp --version" (5s timeout).
func Detect() (Binary, error) {
	bin, err := exec.LookPath("yt-dlp")
	if err != nil {
		return Binary{}, &NoBinaryError{Detail: "yt-dlp not found on PATH"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--version")
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Binary{}, &NoBinaryError{Detail: "yt-dlp --version timed out after 5s"}
		}
		return Binary{}, &NoBinaryError{Detail: "yt-dlp --version failed: " + err.Error()}
	}
	// yt-dlp --version output format: "yt-dlp 2024.01.01"
	// Parse the version string after "yt-dlp "
	verStr := strings.TrimSpace(string(output))
	if idx := strings.Index(verStr, " "); idx >= 0 {
		verStr = verStr[idx+1:]
	}
	// Take just the first token as the version
	ver := strings.Fields(verStr)[0]
	return Binary{Path: bin, Version: ver}, nil
}

type VideoMeta struct {
	ID, URL, Title string
	Position       int
}

type Progress struct {
	Pct               float64
	Downloaded, Total int64
}

type ProgressFunc func(p Progress)

// Fetcher interfaces with yt-dlp to fetch a profile's video metadata.
type Fetcher interface {
	// FetchProfile streams the profile's videos, calling onMeta per entry.
	FetchProfile(ctx context.Context, cfg *config.Config, profileURL string, onMeta func(VideoMeta)) error
}

// Downloader runs a single video to completion, reporting progress, and returns
// the final output path.
type Downloader interface {
	Download(ctx context.Context, cfg *config.Config, videoURL, outDir string, onProgress ProgressFunc) (string, error)
}

type fetcher struct{ binary Binary }

func NewFetcher(bin Binary) Fetcher { return &fetcher{binary: bin} }

func (f *fetcher) FetchProfile(ctx context.Context, cfg *config.Config, profileURL string, onMeta func(VideoMeta)) error {
	bin := f.binary.Path
	if bin == "" {
		return &NoBinaryError{Detail: "yt-dlp binary not set"}
	}
	// Use --dump-json to get JSON for a single video (or --flat-playlist for playlists).
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--dump-json", profileURL)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return &NoBinaryError{Detail: "yt-dlp --dump-json timed out after 30s"}
		}
		return &NoBinaryError{Detail: "yt-dlp --dump-json failed: " + err.Error()}
	}
	var meta struct {
		Title   string `json:"title"`
		ID      string `json:"id"`
		URL     string `json:"url"`
		Entries []struct {
			Title string `json:"title"`
			ID    string `json:"id"`
			URL   string `json:"url"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(output, &meta); err != nil {
		return &NoBinaryError{Detail: "yt-dlp --dump-json JSON parse failed: " + err.Error()}
	}
	// If this is a playlist, iterate entries; otherwise yield the single video.
	if len(meta.Entries) > 0 {
		for _, e := range meta.Entries {
			onMeta(VideoMeta{ID: e.ID, URL: e.URL, Title: e.Title})
		}
	} else {
		onMeta(VideoMeta{ID: meta.ID, URL: profileURL, Title: meta.Title})
	}
	return nil
}

type downloader struct{ binary Binary }

func NewDownloader(bin Binary) Downloader { return &downloader{binary: bin} }

func (d *downloader) Download(ctx context.Context, cfg *config.Config, videoURL, outDir string, onProgress ProgressFunc) (string, error) {
	bin := d.binary.Path
	if bin == "" {
		return "", &NoBinaryError{Detail: "yt-dlp binary not set"}
	}
	// Use --newline for per-chunk progress; in this stub we launch the download
	// and return the expected output path after completion.
	outFile := outDir + "/video_" + fmt.Sprintf("%d", time.Now().Unix()) + ".mp4"
	ctx, cancel := context.WithTimeout(ctx, cfg.PerVideoTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "--newline", "-o", outFile, videoURL)
	cmd.Start()
	// Wait for completion or context cancellation.
	select {
	case <-ctx.Done():
		cmd.Process.Kill()
		return "", fmt.Errorf("download timed out after %s", ctx.Err())
	default:
		// Process started; in a full implementation onProgress would be called
		// from a stderr scanner, but the stub returns the expected path.
		return outFile, cmd.Wait()
	}
}
