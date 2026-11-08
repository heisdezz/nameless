// Package store is the SQLite data layer: profiles, videos, migrations.
//
// Owned by workstream A (progress.md). Contract: docs/implementation-plan.md §3.
// M0 skeleton: bodies return ErrNotImplemented until A3–A5 land.
package store

import (
	"database/sql"
	"errors"
	"time"

	"nameless/internal/ytdlp"
)

// ErrNotImplemented marks skeleton bodies; workstream A replaces them.
var ErrNotImplemented = errors.New("store: not implemented (workstream A, progress.md A3–A5)")

type Profile struct {
	ID         int64
	URL, Name  string
	FetchedAt  time.Time
	VideoCount int
}

type Video struct {
	ID, ProfileID      int64
	URL, Title, Path   string
	Status             string // pending|downloading|paused|done|failed
	Pct                float64
	Err                string
	Downloaded, Total  int64 // bytes
	AddedAt, UpdatedAt time.Time
}

func Open(path string) (*DB, error) { return nil, ErrNotImplemented }

type DB struct{ db *sql.DB }

func (db *DB) Close() error { return ErrNotImplemented }

func (db *DB) CreateProfile(url, name string) (Profile, error) { return Profile{}, ErrNotImplemented }
func (db *DB) Profiles() ([]Profile, error)                    { return nil, ErrNotImplemented }
func (db *DB) DeleteProfile(id int64) error                    { return ErrNotImplemented }

// UpsertVideos inserts new videos and updates existing ones (dedupe by URL),
// preserving playlist order.
func (db *DB) UpsertVideos(profileID int64, vids []ytdlp.VideoMeta) (n int, err error) {
	return 0, ErrNotImplemented
}

// Videos returns a profile's videos ordered by position then id.
func (db *DB) Videos(profileID int64) ([]Video, error) { return nil, ErrNotImplemented }

func (db *DB) SetVideoStatus(id int64, status string) error { return ErrNotImplemented }
func (db *DB) SetVideoProgress(id int64, pct float64, downloaded, total int64) error {
	return ErrNotImplemented
}
func (db *DB) SetVideoFailed(id int64, errMsg, path string) error { return ErrNotImplemented }

// ResetInterrupted moves downloading/paused rows back to pending at startup
// so interrupted runs resume.
func (db *DB) ResetInterrupted() error { return ErrNotImplemented }

func (db *DB) FailedVideos(profileID int64) ([]Video, error) { return nil, ErrNotImplemented }
