// Package ytdlp wraps the yt-dlp binary: detection, profile fetching,
// single-video download with progress.
//
// Owned by workstream B (progress.md). Contract: docs/implementation-plan.md §3.
// M0 skeleton: bodies return ErrNotImplemented until B1–B3 land.
package ytdlp

import (
	"context"
	"errors"

	"nameless/internal/config"
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
func Detect() (Binary, error) { return Binary{}, ErrNotImplemented }

type VideoMeta struct {
	ID, URL, Title string
	Position       int
}

type Progress struct {
	Pct               float64
	Downloaded, Total int64
}

type ProgressFunc func(p Progress)

type Fetcher interface {
	// FetchProfile streams the profile's videos, calling onMeta per entry.
	FetchProfile(ctx context.Context, cfg *config.Config, profileURL string, onMeta func(VideoMeta)) error
}

type Downloader interface {
	// Download runs one video to completion, reporting progress, and returns
	// the final output path.
	Download(ctx context.Context, cfg *config.Config, videoURL, outDir string, onProgress ProgressFunc) (finalPath string, err error)
}

func NewFetcher(bin Binary) Fetcher { return nil }

func NewDownloader(bin Binary) Downloader { return nil }
