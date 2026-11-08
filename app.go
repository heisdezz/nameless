package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	screenProfilesList
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

type toastInfo struct {
	title       string
	description string
	toastType   string // "error", "success", "info"
}

type app struct {
	// mu guards the pause flag shared between the UI thread and the interim
	// download worker; cond wakes the worker on resume. All other state is
	// strictly UI-thread only.
	mu     sync.Mutex
	cond   *sync.Cond
	win    *mygo.Window
	db     *sql.DB
	screen screen

	// persistent config loaded from config.json (or defaults if missing/corrupt).
	// Theme values are applied via c.SetTheme on each view pass.
	cfg     *config.Config
	cfgPath string

	// main
	profileURL string
	outDir     string
	fetching   bool
	fetchErr   string
	profiles   []profile
	sidebarFilter string

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

	// yt-dlp detection state for the Settings screen (WS-D); ytdlpVersion
	// is filled by a background Detect and shown via win.Update.
	ytdlpVersion string
	detecting    bool

	// toasts holds transient notifications (errors, success alerts) dispatched
	// to the UI context on the next view pass.
	toasts []toastInfo
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

// showToast enqueues a transient notification dispatched to ui.Context on the
// next view pass. Thread-safe; wakes the window via win.Update if attached.
func (a *app) showToast(title, desc string, toastType string) {
	a.mu.Lock()
	a.toasts = append(a.toasts, toastInfo{title: title, description: desc, toastType: toastType})
	a.mu.Unlock()
	if a.win != nil {
		a.win.Update(func() {})
	}
}

// effectiveOutputDir returns the resolved output directory: the active
// text input if set, otherwise the configured DefaultDownloadLocation,
// falling back to "./downloads".
func (a *app) effectiveOutputDir() string {
	if a.outDir != "" {
		return a.outDir
	}
	if a.cfg != nil && a.cfg.DefaultDownloadLocation != "" {
		return a.cfg.DefaultDownloadLocation
	}
	return "./downloads"
}

// openFolder opens the given file or directory in the system file manager
// via c.Services().OpenURL, copies the resolved directory path to the
// clipboard, and shows an informational toast.
func (a *app) openFolder(c *ui.Context, target string) {
	if target == "" {
		target = a.effectiveOutputDir()
	}
	dir := resolveFolder(target)
	abs, err := filepath.Abs(dir)
	if err == nil {
		dir = abs
	}
	u := folderURL(dir)
	c.Services().OpenURL(u)
	c.Services().WriteClipboard(dir)
	a.showToast("Opened folder", dir, "info")
}

// resolveFolder returns the enclosing directory if target is a file or
// path with a file extension, otherwise target itself.
func resolveFolder(target string) string {
	if target == "" {
		return "."
	}
	fi, err := os.Stat(target)
	if err == nil {
		if !fi.IsDir() {
			return filepath.Dir(target)
		}
		return target
	}
	if filepath.Ext(target) != "" {
		return filepath.Dir(target)
	}
	return target
}

// folderURL formats a file path into a file:// URL.
func folderURL(dir string) string {
	if dir == "" {
		dir = "."
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	slash := filepath.ToSlash(abs)
	if !strings.HasPrefix(slash, "/") {
		slash = "/" + slash
	}
	return "file://" + slash
}

// overviewStats returns aggregated metrics for the Home dashboard:
// total profiles count, total downloaded videos count, and total videos count.
func (a *app) overviewStats() (profilesCount int, downloadedCount int, totalCount int) {
	profilesCount = len(a.profiles)
	if a.db != nil {
		var done, total int
		row := a.db.QueryRow("SELECT COUNT(CASE WHEN status='done' THEN 1 END), COUNT(*) FROM videos")
		if err := row.Scan(&done, &total); err == nil {
			return profilesCount, done, total
		}
	}
	for _, v := range a.videos {
		if v.status == "done" {
			downloadedCount++
		}
		totalCount++
	}
	return profilesCount, downloadedCount, totalCount
}

// recentDownloads returns the most recently completed downloads up to limit.
func (a *app) recentDownloads(limit int) []video {
	if limit <= 0 {
		limit = 5
	}
	if a.db != nil {
		rows, err := a.db.Query("SELECT id,url,title,status,path FROM videos WHERE status='done' ORDER BY id DESC LIMIT ?", limit)
		if err == nil {
			defer rows.Close()
			var vids []video
			for rows.Next() {
				var v video
				if err := rows.Scan(&v.id, &v.url, &v.title, &v.status, &v.path); err != nil {
					break
				}
				vids = append(vids, v)
			}
			if len(vids) > 0 || len(a.videos) == 0 {
				return vids
			}
		}
	}
	var vids []video
	for i := len(a.videos) - 1; i >= 0; i-- {
		if a.videos[i].status == "done" {
			vids = append(vids, a.videos[i])
			if len(vids) >= limit {
				break
			}
		}
	}
	return vids
}


// enterSettings prepares the settings bindings before the screen is shown.
func (a *app) enterSettings() {
	if a.outDir == "" && a.cfg != nil {
		a.outDir = a.cfg.DefaultDownloadLocation
	}
	a.radiusF64 = float64(a.cfg.Theme.Radius)
	a.workersF64 = float64(a.cfg.Workers)
	a.spacingSel = spacingLabel(a.cfg.Theme.Spacing)
	a.delayStr = a.cfg.Delay.String()
	a.timeoutStr = a.cfg.PerVideoTimeout.String()
	a.screen = screenSettings
}

// spacingLabel / spacingUnit map the theme spacing unit (plan §3: 3 compact,
// 4 default, 5 roomy) to the Settings select labels. WS-D consumes these in
// ui_settings.go (contract change log 2026-11-08).
func spacingLabel(unit float32) string {
	switch unit {
	case 3:
		return "Compact"
	case 5:
		return "Roomy"
	default:
		return "Default"
	}
}

func spacingUnit(label string) float32 {
	switch label {
	case "Compact":
		return 3
	case "Roomy":
		return 5
	default:
		return 4
	}
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
		a.showToast("Error", "Enter a profile URL", "error")
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
				a.showToast("Fetch Error", a.fetchErr, "error")
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
			p := filepath.Join(a.effectiveOutputDir(), fmt.Sprintf("video_%d.mp4", id))
			a.db.Exec("UPDATE videos SET status=?, path=? WHERE id=?", "done", p, id)
			a.win.Update(func() { a.setVideoDone(id, p) })
		}
		a.win.Update(func() {
			a.downloading = false
			done, total := 0, len(a.videos)
			for _, v := range a.videos {
				if v.status == "done" {
					done++
				}
			}
			if done == total && total > 0 {
				a.showToast("Download Complete", fmt.Sprintf("All %d videos downloaded", total), "success")
			}
		})
	}()
}

// setVideoStatus updates one video by id; no-op for ids not on screen. UI thread only.
func (a *app) setVideoStatus(id int64, status string) {
	for i := range a.videos {
		if a.videos[i].id == id {
			a.videos[i].status = status
			if status == "failed" {
				a.showToast("Download Failed", a.videos[i].title, "error")
			}
			return
		}
	}
}

// setVideoDone marks a video done and sets its final downloaded path. UI thread only.
func (a *app) setVideoDone(id int64, path string) {
	for i := range a.videos {
		if a.videos[i].id == id {
			a.videos[i].status = "done"
			a.videos[i].path = path
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
	outDir := ""
	if cfg != nil {
		outDir = cfg.DefaultDownloadLocation
	}
	a := &app{cfg: cfg, cfgPath: cfgPath, outDir: outDir}
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
