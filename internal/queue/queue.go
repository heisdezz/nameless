// Package queue manages the download worker pool: pause/resume/retry and
// event emission for the UI.
//
// Owned by workstream C (progress.md). Contract: docs/implementation-plan.md §3.
// M0 skeleton: methods panic with ErrNotImplemented until C1–C3 land.
package queue

import (
	"context"
	"errors"

	"nameless/internal/config"
	"nameless/internal/store"
	"nameless/internal/ytdlp"
)

// ErrNotImplemented marks skeleton bodies; workstream C replaces them.
var ErrNotImplemented = errors.New("queue: not implemented (workstream C, progress.md C1–C3)")

type Event struct {
	VideoID   int64
	Status    string // pending|downloading|paused|done|failed
	Pct       float64
	Err       string
	FinalPath string
}

// Store is the consumer-side data interface the queue needs.
type Store interface {
	Videos(profileID int64) ([]store.Video, error)
	SetVideoStatus(id int64, status string) error
	SetVideoProgress(id int64, pct float64, dl, total int64) error
	SetVideoFailed(id int64, msg, path string) error
}

// Downloader is the consumer-side engine interface the queue needs.
type Downloader interface {
	Download(ctx context.Context, cfg *config.Config, videoURL, outDir string, onProgress ytdlp.ProgressFunc) (string, error)
}

type Manager struct{}

func New(st Store, dl Downloader, cfg *config.Config, events chan<- Event) *Manager {
	return &Manager{}
}

// Start schedules all pending videos of the profile, cfg.Workers at a time,
// cfg.Delay between launches.
func (m *Manager) Start(profileID int64, outDir string) { panic(ErrNotImplemented) }

// Pause stops scheduling and cancels in-flight jobs (status "paused"; .part
// files are kept for yt-dlp resume).
func (m *Manager) Pause() { panic(ErrNotImplemented) }

// Resume re-schedules pending and paused videos.
func (m *Manager) Resume() { panic(ErrNotImplemented) }

func (m *Manager) RetryFailed() { panic(ErrNotImplemented) }

// Close cancels everything; idempotent.
func (m *Manager) Close() { panic(ErrNotImplemented) }
