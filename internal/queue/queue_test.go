package queue

import (
	"context"
	"sync"
	"testing"
	"time"

	"nameless/internal/config"
	"nameless/internal/store"
	"nameless/internal/tikdownloader"
)

type fakeStore struct {
	mu     sync.Mutex
	videos []store.Video
}

func (s *fakeStore) Videos(profileID int64) ([]store.Video, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]store.Video(nil), s.videos...), nil
}

func (s *fakeStore) SetVideoStatus(id int64, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.videos {
		if s.videos[i].ID == id {
			s.videos[i].Status = status
		}
	}
	return nil
}

func (s *fakeStore) SetVideoProgress(id int64, pct float64, dl, total int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.videos {
		if s.videos[i].ID == id {
			s.videos[i].Pct = pct
			s.videos[i].Downloaded = dl
			s.videos[i].Total = total
		}
	}
	return nil
}

func (s *fakeStore) SetVideoFailed(id int64, msg, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.videos {
		if s.videos[i].ID == id {
			s.videos[i].Status = store.StatusFailed
			s.videos[i].Err = msg
			s.videos[i].Path = path
		}
	}
	return nil
}

func (s *fakeStore) FailedVideos(profileID int64) ([]store.Video, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []store.Video
	for _, v := range s.videos {
		if v.Status == store.StatusFailed {
			out = append(out, v)
		}
	}
	return out, nil
}

type fakeDownloader struct{}

func (d *fakeDownloader) DownloadVideo(ctx context.Context, videoURL, outDir string) (*tikdownloader.DownloadResult, error) {
	return &tikdownloader.DownloadResult{Success: true, Path: outDir + "/video.mp4"}, nil
}

func (d *fakeDownloader) GetSupportedFormats() []string { return []string{"mp4"} }

func TestQueuePauseResumeRace(t *testing.T) {
	fs := &fakeStore{
		videos: []store.Video{
			{ID: 1, ProfileID: 1, Status: "pending"},
			{ID: 2, ProfileID: 1, Status: "failed"},
		},
	}
	fd := &fakeDownloader{}
	cfg := config.Default()
	cfg.Workers = 2
	cfg.Delay = 10 * time.Millisecond

	events := make(chan Event, 100)
	q := New(fs, fd, cfg, events)

	q.Start(1, t.TempDir())
	time.Sleep(20 * time.Millisecond)

	q.Pause()
	time.Sleep(20 * time.Millisecond)

	q.Resume()
	time.Sleep(20 * time.Millisecond)

	q.RetryFailed()

	q.Close()
}
