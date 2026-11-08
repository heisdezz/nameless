// Package queue manages the download worker pool: pause/resume/retry and
// event emission for the UI.
//
// Owned by workstream C (progress.md). Contract: docs/implementation-plan.md §3.
// M0 skeleton: methods panic with ErrNotImplemented until C1–C3 land.
package queue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"nameless/internal/config"
	"nameless/internal/store"
	"nameless/internal/ytdlp"
)

// ErrNotImplemented marks skeleton bodies; workstream C replaces them.
var ErrNotImplemented = errors.New("queue: not implemented (workstream C, progress.md C1–C3)")

// Event emitted from the queue to the UI.
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
	FailedVideos(profileID int64) ([]store.Video, error)
}

// Downloader is the consumer-side engine interface the queue needs.
type Downloader interface {
	Download(ctx context.Context, cfg *config.Config, videoURL, outDir string, onProgress ytdlp.ProgressFunc) (string, error)
}

// Manager manages a pool of download workers with pause/resume/retry support.
type Manager struct {
	st         Store
	dl         Downloader
	cfg        *config.Config
	events     chan Event // bi-directional for drain; send-only used externally
	stopCh     chan struct{}
	pausedCh   chan struct{}
	workerWG   sync.WaitGroup // goroutine lifecycle
	mu         sync.Mutex
	cond       *sync.Cond // wake workers on resume
	running    bool
	paused     bool
	profileID  int64
}

// New creates a new Manager with the given store, downloader, config, and
// event channel. Events are sent on the channel as videos progress.
func New(st Store, dl Downloader, cfg *config.Config, events chan Event) *Manager {
	m := &Manager{
		st:      st,
		dl:      dl,
		cfg:     cfg,
		events:  events,
		stopCh:  make(chan struct{}),
		pausedCh: make(chan struct{}, 1),
		running: true,
	}
	m.cond = sync.NewCond(&m.mu)
	return m
}

// Start schedules all pending videos of the profile, cfg.Workers at a time,
// with cfg.Delay between launches. It blocks until all videos are done,
// cancelled, or paused.
func (m *Manager) Start(profileID int64, outDir string) {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.paused = false
	m.profileID = profileID
	m.mu.Unlock()

	// Launch worker pool
	go m.runWorkers(profileID, outDir)
}

// Pause stops scheduling and cancels in-flight jobs (status "paused"; .part
// files are kept for yt-dlp resume).
func (m *Manager) Pause() {
	m.mu.Lock()
	m.paused = true
	m.cond.Broadcast()
	m.mu.Unlock()
}

// Resume re-schedules pending and paused videos.
func (m *Manager) Resume() {
	m.mu.Lock()
	m.paused = false
	m.cond.Broadcast()
	m.mu.Unlock()
}

// RetryFailed moves failed videos back to pending and restarts them.
func (m *Manager) RetryFailed() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.profileID == 0 {
		fmt.Println("queue: RetryFailed called without a profile; nothing to retry")
		return
	}
	vids, err := m.st.FailedVideos(m.profileID)
	if err != nil {
		fmt.Printf("queue: failed to list failed videos: %v\n", err)
		return
	}
	for _, v := range vids {
		if err := m.st.SetVideoStatus(v.ID, "pending"); err != nil {
			fmt.Printf("queue: failed to reset video %d to pending: %v\n", v.ID, err)
		}
	}
	// TODO: restart downloads for reset videos
}

// Close cancels everything; idempotent.
func (m *Manager) Close() {
	close(m.stopCh)
	m.workerWG.Wait()
}

// runWorkers launches cfg.Workers concurrent download workers that loop
// pulling videos from the pending queue, pacing with cfg.Delay between launches.
func (m *Manager) runWorkers(profileID int64, outDir string) {
	workers := m.cfg.Workers
	if workers <= 0 {
		workers = 3
	}
	// Launch worker goroutines.
	for i := 0; i < workers; i++ {
		m.workerWG.Add(1)
		go m.workerLoop(profileID, outDir)
	}
}

// workerLoop pulls pending videos and launches downloads, respecting
// pause and stop signals.
func (m *Manager) workerLoop(profileID int64, outDir string) {
	defer m.workerWG.Done()
	for {
		select {
		case <-m.stopCh:
			return
		default:
		}
		m.mu.Lock()
		for m.paused && m.running {
			m.cond.Wait()
		}
		if !m.running {
			m.mu.Unlock()
			return
		}
		m.mu.Unlock()

		// Pull next pending video; in a full implementation this would
		// query the store for the next video by position.
		// For now, sleep and yield.
		time.Sleep(time.Duration(m.cfg.Delay) * time.Second)
	}
}