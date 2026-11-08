// Package store is the SQLite data layer: profiles, videos, migrations.
//
// Owned by workstream A (progress.md). Contract: docs/implementation-plan.md §3.
//
// Open runs additive migrations (plan §3) that converge both a fresh database
// and the legacy v1 schema produced by the prototype (profiles(id,url,name),
// videos(id,profile_id,url,title,status,path)) onto the v2 shape. Existing
// tables are only ever ALTERed for missing columns, so migrations are
// idempotent and never lose data. Errors are wrapped with %w; nothing here
// calls log.Fatal.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // registers the pure-Go "sqlite" driver (CGO_ENABLED=0 safe)

	"nameless/internal/ytdlp"
)

// Status values stored in videos.status (plan §3).
const (
	StatusPending     = "pending"
	StatusDownloading = "downloading"
	StatusPaused      = "paused"
	StatusDone        = "done"
	StatusFailed      = "failed"
)

// schemaVersion is the version stamped into schema_version once migrations ran.
const schemaVersion = 2

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

// Open opens (creating if needed) the database at path, applies pragmas and
// migrations, and verifies the connection.
func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	// A single connection serializes UI-pump and queue-worker statements so
	// in-app contention can never surface as SQLITE_BUSY.
	sqldb.SetMaxOpenConns(1)
	db := &DB{db: sqldb}
	if err := db.configure(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	if err := db.migrate(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("store: open %s: %w", path, err)
	}
	return db, nil
}

type DB struct{ db *sql.DB }

func (db *DB) Close() error {
	if err := db.db.Close(); err != nil {
		return fmt.Errorf("store: close: %w", err)
	}
	return nil
}

// configure sets pragmas useful for a desktop app: a busy timeout for any
// contention with external processes, and WAL journaling so the queue's
// writes don't block UI reads. WAL is a best effort — filesystems that
// cannot support it simply keep their journal mode.
func (db *DB) configure() error {
	var timeout int
	if err := db.db.QueryRow(`PRAGMA busy_timeout = 5000`).Scan(&timeout); err != nil {
		return fmt.Errorf("set busy_timeout: %w", err)
	}
	var mode string
	if err := db.db.QueryRow(`PRAGMA journal_mode = WAL`).Scan(&mode); err != nil {
		return fmt.Errorf("set journal_mode: %w", err)
	}
	return nil
}

// migrate brings any database — fresh, legacy v1, or already v2 — to the v2
// shape (plan §3) using idempotent statements and column detection.
func (db *DB) migrate() error {
	// v1 base tables in the exact shape the prototype's initDB created;
	// no-ops on databases that already have them.
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT)`,
		`CREATE TABLE IF NOT EXISTS videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT)`,
		`CREATE TABLE IF NOT EXISTS schema_version(version INTEGER NOT NULL)`,
	} {
		if _, err := db.db.Exec(q); err != nil {
			return fmt.Errorf("create base tables: %w", err)
		}
	}

	// v2 additive columns, only applied when missing (legacy v1 databases).
	for _, c := range []struct{ table, name, decl string }{
		{"profiles", "fetched_at", "INTEGER"},
		{"profiles", "video_count", "INTEGER DEFAULT 0"},
		{"videos", "pct", "REAL DEFAULT 0"},
		{"videos", "err", "TEXT DEFAULT ''"},
		{"videos", "downloaded_bytes", "INTEGER DEFAULT 0"},
		{"videos", "total_bytes", "INTEGER DEFAULT 0"},
		{"videos", "added_at", "INTEGER"},
		{"videos", "updated_at", "INTEGER"},
		{"videos", "position", "INTEGER"},
	} {
		if err := db.addColumn(c.table, c.name, c.decl); err != nil {
			return err
		}
	}

	if _, err := db.db.Exec(`CREATE INDEX IF NOT EXISTS idx_videos_profile ON videos(profile_id)`); err != nil {
		return fmt.Errorf("create index idx_videos_profile: %w", err)
	}
	// Stamp the version (single statement, idempotent).
	if _, err := db.db.Exec(`INSERT INTO schema_version(version) SELECT ? WHERE NOT EXISTS(SELECT 1 FROM schema_version WHERE version >= ?)`, schemaVersion, schemaVersion); err != nil {
		return fmt.Errorf("stamp schema_version: %w", err)
	}
	return nil
}

// addColumn ALTERs table if column name is not already present, detected via
// PRAGMA table_info.
func (db *DB) addColumn(table, name, decl string) error {
	rows, err := db.db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", table, err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var (
			cid     int
			col     string
			typ     string
			notNull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &col, &typ, &notNull, &dflt, &pk); err != nil {
			return fmt.Errorf("inspect %s: scan: %w", table, err)
		}
		if col == name {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("inspect %s: %w", table, err)
	}
	if found {
		return nil
	}
	if _, err := db.db.Exec(`ALTER TABLE ` + table + ` ADD COLUMN ` + name + ` ` + decl); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, name, err)
	}
	return nil
}

func (db *DB) CreateProfile(url, name string) (Profile, error) {
	res, err := db.db.Exec(`INSERT INTO profiles(url, name, fetched_at, video_count) VALUES (?, ?, NULL, 0)`, url, name)
	if err != nil {
		return Profile{}, fmt.Errorf("store: create profile %s: %w", url, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Profile{}, fmt.Errorf("store: create profile %s: last insert id: %w", url, err)
	}
	// FetchedAt stays zero until the first fetch fills it (UpsertVideos).
	return Profile{ID: id, URL: url, Name: name}, nil
}

func (db *DB) Profiles() ([]Profile, error) {
	rows, err := db.db.Query(`SELECT id, url, name, fetched_at, video_count FROM profiles ORDER BY id DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: list profiles: %w", err)
	}
	defer rows.Close()
	var out []Profile
	for rows.Next() {
		var p Profile
		var fetched, count sql.NullInt64
		if err := rows.Scan(&p.ID, &p.URL, &p.Name, &fetched, &count); err != nil {
			return nil, fmt.Errorf("store: scan profile: %w", err)
		}
		p.FetchedAt = timeFromStamp(fetched)
		p.VideoCount = int(count.Int64)
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list profiles: %w", err)
	}
	return out, nil
}

func (db *DB) DeleteProfile(id int64) error {
	tx, err := db.db.Begin()
	if err != nil {
		return fmt.Errorf("store: delete profile %d: begin: %w", id, err)
	}
	if _, err := tx.Exec(`DELETE FROM videos WHERE profile_id = ?`, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("store: delete videos of profile %d: %w", id, err)
	}
	if _, err := tx.Exec(`DELETE FROM profiles WHERE id = ?`, id); err != nil {
		tx.Rollback()
		return fmt.Errorf("store: delete profile %d: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: delete profile %d: commit: %w", id, err)
	}
	return nil
}

// UpsertVideos inserts new videos and updates existing ones (dedupe by URL),
// preserving playlist order. Duplicate URLs within one batch keep the first
// occurrence. Returns n, the number of newly added rows. The profile's
// video_count and fetched_at are refreshed to reflect the completed fetch.
func (db *DB) UpsertVideos(profileID int64, vids []ytdlp.VideoMeta) (n int, err error) {
	tx, err := db.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("store: upsert videos of profile %d: begin: %w", profileID, err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// Map the profile's existing rows by URL for dedupe.
	rows, err := tx.Query(`SELECT id, url FROM videos WHERE profile_id = ?`, profileID)
	if err != nil {
		return 0, fmt.Errorf("store: upsert videos of profile %d: list existing: %w", profileID, err)
	}
	existing := make(map[string]int64, len(vids))
	for rows.Next() {
		var id int64
		var url string
		if err := rows.Scan(&id, &url); err != nil {
			rows.Close()
			return 0, fmt.Errorf("store: upsert videos of profile %d: scan existing: %w", profileID, err)
		}
		existing[url] = id
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("store: upsert videos of profile %d: list existing: %w", profileID, err)
	}
	rows.Close()

	now := time.Now().Unix()
	inserted := make(map[string]bool, len(vids))
	const insertQ = `INSERT INTO videos(profile_id, url, title, status, path, pct, err, downloaded_bytes, total_bytes, added_at, updated_at, position)
VALUES (?, ?, ?, ?, '', 0, '', 0, 0, ?, ?, ?)`
	for _, m := range vids {
		if id, ok := existing[m.URL]; ok {
			if _, err := tx.Exec(`UPDATE videos SET title = ?, position = ? WHERE id = ?`, m.Title, m.Position, id); err != nil {
				return n, fmt.Errorf("store: upsert videos: update %s: %w", m.URL, err)
			}
			continue
		}
		if inserted[m.URL] {
			continue // duplicate URL in the same batch: keep the first occurrence
		}
		if _, err := tx.Exec(insertQ, profileID, m.URL, m.Title, StatusPending, now, now, m.Position); err != nil {
			return n, fmt.Errorf("store: upsert videos: insert %s: %w", m.URL, err)
		}
		inserted[m.URL] = true
		n++
	}

	if _, err := tx.Exec(`UPDATE profiles SET video_count = (SELECT COUNT(*) FROM videos WHERE profile_id = ?), fetched_at = ? WHERE id = ?`, profileID, now, profileID); err != nil {
		return n, fmt.Errorf("store: upsert videos: refresh profile %d: %w", profileID, err)
	}
	if err := tx.Commit(); err != nil {
		return n, fmt.Errorf("store: upsert videos of profile %d: commit: %w", profileID, err)
	}
	return n, nil
}

// Videos returns a profile's videos ordered by position then id.
func (db *DB) Videos(profileID int64) ([]Video, error) {
	rows, err := db.db.Query(`SELECT `+videoColumns+` FROM videos WHERE profile_id = ? ORDER BY position, id`, profileID)
	if err != nil {
		return nil, fmt.Errorf("store: list videos of profile %d: %w", profileID, err)
	}
	defer rows.Close()
	var out []Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan video: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list videos of profile %d: %w", profileID, err)
	}
	return out, nil
}

func (db *DB) SetVideoStatus(id int64, status string) error {
	if _, err := db.db.Exec(`UPDATE videos SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now().Unix(), id); err != nil {
		return fmt.Errorf("store: set status of video %d: %w", id, err)
	}
	return nil
}

func (db *DB) SetVideoProgress(id int64, pct float64, downloaded, total int64) error {
	if _, err := db.db.Exec(`UPDATE videos SET pct = ?, downloaded_bytes = ?, total_bytes = ?, updated_at = ? WHERE id = ?`, pct, downloaded, total, time.Now().Unix(), id); err != nil {
		return fmt.Errorf("store: set progress of video %d: %w", id, err)
	}
	return nil
}

func (db *DB) SetVideoFailed(id int64, errMsg, path string) error {
	if _, err := db.db.Exec(`UPDATE videos SET status = ?, err = ?, path = ?, updated_at = ? WHERE id = ?`, StatusFailed, errMsg, path, time.Now().Unix(), id); err != nil {
		return fmt.Errorf("store: set video %d failed: %w", id, err)
	}
	return nil
}

// ResetInterrupted moves downloading/paused rows back to pending at startup
// so interrupted runs resume.
func (db *DB) ResetInterrupted() error {
	if _, err := db.db.Exec(`UPDATE videos SET status = ?, updated_at = ? WHERE status IN (?, ?)`, StatusPending, time.Now().Unix(), StatusDownloading, StatusPaused); err != nil {
		return fmt.Errorf("store: reset interrupted videos: %w", err)
	}
	return nil
}

func (db *DB) FailedVideos(profileID int64) ([]Video, error) {
	rows, err := db.db.Query(`SELECT `+videoColumns+` FROM videos WHERE profile_id = ? AND status = ? ORDER BY position, id`, profileID, StatusFailed)
	if err != nil {
		return nil, fmt.Errorf("store: list failed videos of profile %d: %w", profileID, err)
	}
	defer rows.Close()
	var out []Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, fmt.Errorf("store: scan video: %w", err)
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: list failed videos of profile %d: %w", profileID, err)
	}
	return out, nil
}

const videoColumns = `id, profile_id, url, title, status, path, pct, err, downloaded_bytes, total_bytes, added_at, updated_at`

// scanVideo reads one row of videoColumns, tolerating NULLs (legacy rows may
// predate some columns).
func scanVideo(rows *sql.Rows) (Video, error) {
	var v Video
	var profileID sql.NullInt64
	var url, title, status, path, errMsg sql.NullString
	var pct sql.NullFloat64
	var dl, total, added, updated sql.NullInt64
	if err := rows.Scan(&v.ID, &profileID, &url, &title, &status, &path, &pct, &errMsg, &dl, &total, &added, &updated); err != nil {
		return Video{}, err
	}
	v.ProfileID = profileID.Int64
	v.URL, v.Title, v.Status, v.Path = url.String, title.String, status.String, path.String
	v.Err = errMsg.String
	v.Pct = pct.Float64
	v.Downloaded, v.Total = dl.Int64, total.Int64
	v.AddedAt, v.UpdatedAt = timeFromStamp(added), timeFromStamp(updated)
	return v, nil
}

// timeFromStamp converts a unix-seconds column (NULL on never-fetched rows) to
// a time.Time; zero values map to the zero time.
func timeFromStamp(s sql.NullInt64) time.Time {
	if !s.Valid || s.Int64 <= 0 {
		return time.Time{}
	}
	return time.Unix(s.Int64, 0)
}
