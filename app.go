package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"nameless/internal/config"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	_ "modernc.org/sqlite"
)

type screen int

const (
	screenMain screen = iota
	screenProfile
	screenSettings
)

type video struct {
	id     int64
	url    string
	title  string
	status string // pending/downloading/done/failed/paused
	path   string
}

type profile struct {
	id   int64
	url  string
	name string
}

type app struct {
	// mu guards the pause flag shared between the UI thread and the interim
	// download worker; cond wakes the worker on resume. All other state is
	// mutated only on the UI thread (view actions and win.Update closures).
	mu   sync.Mutex
	cond *sync.Cond

	db      *sql.DB
	win     *mygo.Window
	screen  screen
	cfg     *config.Config
	cfgPath string

	// main
	profileURL string
	outDir     string
	fetching   bool
	fetchErr   string
	profiles   []profile

	// profile view
	currentProfile *profile
	videos         []video
	downloading    bool
	paused         bool // guarded by mu

	// settings UI mirrors: persistent bindings for float/slider and
	// duration inputs (ui.Slider binds *float64; durations bind as text).
	// Owned by WS-D; see progress.md contract changes.
	radiusF64  float64
	workersF64 float64
	spacingSel string
	delayStr   string
	timeoutStr string
}

func defaultConfig() *config.Config {
	return &config.Config{
		DefaultDownloadLocation: "./downloads",
		Workers:                 3,
		Delay:                   time.Second,
		PerVideoTimeout:         5 * time.Minute,
		Theme: config.ThemeConfig{
			AccentHex: "#4f46e5",
			Radius:    12,
			Spacing:   4,
		},
	}
}

// enterSettings prepares the settings bindings before the screen is shown.
// WS-D may extend it (e.g. spacingSel) when the full settings UI lands.
func (a *app) enterSettings() {
	a.radiusF64 = float64(a.cfg.Theme.Radius)
	a.workersF64 = float64(a.cfg.Workers)
	a.delayStr = a.cfg.Delay.String()
	a.timeoutStr = a.cfg.PerVideoTimeout.String()
	a.screen = screenSettings
}

func (a *app) initDB() {
	dir, _ := os.UserConfigDir()
	path := filepath.Join(dir, "nameless", "data.db")
	os.MkdirAll(filepath.Dir(path), 0755)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}
	a.db = db
	db.Exec(`CREATE TABLE IF NOT EXISTS profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT)`)
	db.Exec(`CREATE TABLE IF NOT EXISTS videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT)`)
}

// loadProfiles reads the profile list. UI thread only.
func (a *app) loadProfiles() {
	rows, err := a.db.Query("SELECT id,url,name FROM profiles")
	if err != nil {
		return
	}
	defer rows.Close()
	a.profiles = nil
	for rows.Next() {
		var p profile
		if err := rows.Scan(&p.id, &p.url, &p.name); err != nil {
			return
		}
		a.profiles = append(a.profiles, p)
	}
}

// loadVideos queries a profile's videos; safe on any thread (DB only, no app state).
func (a *app) loadVideos(profileID int64) []video {
	rows, err := a.db.Query("SELECT id,url,title,status,path FROM videos WHERE profile_id=?", profileID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var vids []video
	for rows.Next() {
		var v video
		if err := rows.Scan(&v.id, &v.url, &v.title, &v.status, &v.path); err != nil {
			return vids
		}
		vids = append(vids, v)
	}
	return vids
}

// applyProfile switches to the profile screen. UI thread only.
func (a *app) applyProfile(p profile, vids []video) {
	pcopy := p
	a.currentProfile = &pcopy
	a.screen = screenProfile
	a.videos = vids
}

func (a *app) fetchProfile() {
	if a.profileURL == "" {
		a.fetchErr = "Enter a profile URL"
		return
	}
	if a.fetching {
		return
	}
	a.fetching = true
	a.fetchErr = ""
	url := a.profileURL
	go func() {
		name := url
		if len(name) > 20 {
			name = name[len(name)-20:]
		}
		p := profile{url: url, name: name}
		res, err := a.db.Exec("INSERT INTO profiles(url,name) VALUES(?,?)", p.url, p.name)
		if err != nil {
			a.win.Update(func() {
				a.fetching = false
				a.fetchErr = "Could not save profile: " + err.Error()
			})
			return
		}
		p.id, _ = res.LastInsertId()
		vids := a.loadVideos(p.id) // DB read off the UI thread
		a.win.Update(func() {
			a.fetching = false
			a.profiles = append(a.profiles, p)
			a.applyProfile(p, vids)
		})
	}()
}

// openProfile navigates to a profile by value (never a pointer into a slice
// that may be appended to later). UI thread only.
func (a *app) openProfile(p profile) {
	a.applyProfile(p, a.loadVideos(p.id))
}

func (a *app) startDownload() {
	if a.currentProfile == nil {
		return
	}
	a.setPaused(false)
	a.downloading = true
	// Snapshot: the worker reads only vids and reports through win.Update,
	// so UI-thread mutations of a.videos can never race with it.
	vids := make([]video, len(a.videos))
	copy(vids, a.videos)
	go func() {
		for i := range vids {
			a.waitWhilePaused()
			id := vids[i].id
			a.win.Update(func() { a.setVideoStatus(id, "downloading") })
			// simulate yt-dlp download (replaced by internal/queue in M3)
			a.db.Exec("UPDATE videos SET status=? WHERE id=?", "done", id)
			a.win.Update(func() { a.setVideoStatus(id, "done") })
		}
		a.win.Update(func() { a.downloading = false })
	}()
}

// setVideoStatus updates one video by id; no-op for ids not on screen. UI thread only.
func (a *app) setVideoStatus(id int64, status string) {
	for i := range a.videos {
		if a.videos[i].id == id {
			a.videos[i].status = status
			return
		}
	}
}

func (a *app) retryFailed() {
	for i := range a.videos {
		if a.videos[i].status == "failed" {
			a.videos[i].status = "pending"
		}
	}
}

// setPaused toggles the interim pause flag and wakes the worker on resume.
func (a *app) setPaused(v bool) {
	a.mu.Lock()
	if a.cond == nil {
		a.cond = sync.NewCond(&a.mu)
	}
	a.paused = v
	if !v {
		a.cond.Broadcast()
	}
	a.mu.Unlock()
}

// waitWhilePaused blocks the (interim) worker without busy-looping while paused.
func (a *app) waitWhilePaused() {
	a.mu.Lock()
	if a.cond == nil {
		a.cond = sync.NewCond(&a.mu)
	}
	for a.paused {
		a.cond.Wait()
	}
	a.mu.Unlock()
}

// main wires config, DB, and the main window (original main.go was merged
// here during the E0 split — see progress.md).
func main() {
	cfgPath := "config.json"
	cfg, err := config.LoadOrCreate(cfgPath)
	if err != nil || cfg == nil {
		cfg = defaultConfig() // skeleton/internal error: run on defaults (plan §3)
	}
	a := &app{cfg: cfg, cfgPath: cfgPath}
	a.initDB()
	a.loadProfiles()

	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "nameless",
			Width:     720,
			Height:    560,
			MinWidth:  360,
			MinHeight: 280,
			StateKey:  "main",
			Content:   ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
