package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nameless/internal/config"
	"nameless/internal/ytdlp"

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
	id         int64
	url        string
	name       string
	avatarURL  string
	avatarPath string
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
	profileURL    string
	outDir        string
	fetching      bool
	fetchErr      string
	profiles      []profile
	sidebarFilter string
	imgCache      map[int64]ui.ImageSource
	avatarsDir    string
	gridCols      string

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
	ytdlpMissing bool
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

// profileVideosCount returns the number of videos associated with the profile.
func (a *app) profileVideosCount(profileID int64) int {
	if a.db != nil {
		var count int
		if err := a.db.QueryRow("SELECT COUNT(*) FROM videos WHERE profile_id=?", profileID).Scan(&count); err == nil {
			return count
		}
	}
	if a.currentProfile != nil && a.currentProfile.id == profileID {
		return len(a.videos)
	}
	return 0
}

// deleteProfile removes a profile and its video records from the database.
func (a *app) deleteProfile(profileID int64) {
	if a.db != nil {
		a.db.Exec("DELETE FROM videos WHERE profile_id=?", profileID)
		a.db.Exec("DELETE FROM profiles WHERE id=?", profileID)
	}
	for i, p := range a.profiles {
		if p.id == profileID {
			a.profiles = append(a.profiles[:i], a.profiles[i+1:]...)
			break
		}
	}
	if a.currentProfile != nil && a.currentProfile.id == profileID {
		a.currentProfile = nil
		a.videos = nil
		a.screen = screenProfilesList
	}
	a.showToast("Profile Removed", "Profile was deleted from archive", "info")
}

// detectYTDLP probes system PATH for yt-dlp binary (plan §4, milestone 3 task E5).
func (a *app) detectYTDLP() {
	bin, err := ytdlp.Detect()
	if err != nil {
		a.ytdlpMissing = true
		a.ytdlpVersion = "yt-dlp Missing"
		a.showToast("yt-dlp Missing", "yt-dlp was not found on system PATH", "error")
	} else {
		a.ytdlpMissing = false
		a.ytdlpVersion = "yt-dlp " + bin.Version
		a.showToast("yt-dlp Ready", "Detected "+a.ytdlpVersion, "success")
	}
}

func (a *app) openInstallDocs(c *ui.Context) {
	c.Services().OpenURL("https://github.com/yt-dlp/yt-dlp#installation")
}

// profileImage returns the decoded ImageSource for a profile, or nil if unavailable.
func (a *app) profileImage(p profile) ui.ImageSource {
	if a.imgCache == nil {
		a.imgCache = make(map[int64]ui.ImageSource)
	}
	if src, ok := a.imgCache[p.id]; ok {
		return src
	}
	if p.avatarPath != "" {
		if data, err := os.ReadFile(p.avatarPath); err == nil {
			if src, err := ui.DecodeImage(data); err == nil {
				a.imgCache[p.id] = src
				return src
			}
		}
	}
	return nil
}

// enterSettings prepares the settings bindings before the screen is shown.
func (a *app) enterSettings() {
	if a.cfg == nil {
		a.cfg = defaultConfig()
	}
	if a.outDir == "" {
		a.outDir = a.cfg.DefaultDownloadLocation
	}
	a.radiusF64 = float64(a.cfg.Theme.Radius)
	a.workersF64 = float64(a.cfg.Workers)
	a.spacingSel = spacingLabel(a.cfg.Theme.Spacing)
	a.delayStr = a.cfg.Delay.String()
	a.timeoutStr = a.cfg.PerVideoTimeout.String()
	a.screen = screenSettings
}

// saveSettings persists the in-memory settings to config.json.
func (a *app) saveSettings() {
	if a.cfg == nil {
		return
	}
	if a.outDir != "" {
		a.cfg.DefaultDownloadLocation = a.outDir
	}
	if a.workersF64 >= 1 && a.workersF64 <= 16 {
		a.cfg.Workers = int(a.workersF64)
	}
	if d, err := time.ParseDuration(a.delayStr); err == nil && d > 0 {
		a.cfg.Delay = d
	}
	if to, err := time.ParseDuration(a.timeoutStr); err == nil && to > 0 {
		a.cfg.PerVideoTimeout = to
	}
	if a.radiusF64 >= 0 {
		a.cfg.Theme.Radius = float32(a.radiusF64)
	}
	cfgPath := a.cfgPath
	if cfgPath == "" {
		cfgPath = "config.json"
	}
	if err := a.cfg.Save(cfgPath); err != nil {
		a.showToast("Save Error", err.Error(), "error")
	} else {
		a.showToast("Settings Saved", "Preferences updated in "+filepath.Base(cfgPath), "success")
	}
}

// resetSettingsToDefault restores all options to initial factory defaults.
func (a *app) resetSettingsToDefault() {
	a.cfg = config.Default()
	a.outDir = a.cfg.DefaultDownloadLocation
	a.radiusF64 = float64(a.cfg.Theme.Radius)
	a.workersF64 = float64(a.cfg.Workers)
	a.spacingSel = spacingLabel(a.cfg.Theme.Spacing)
	a.delayStr = a.cfg.Delay.String()
	a.timeoutStr = a.cfg.PerVideoTimeout.String()
	a.showToast("Reset Defaults", "All settings restored to defaults", "info")
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
	if dir == "" {
		dir = "."
	}
	appDir := filepath.Join(dir, "nameless")
	os.MkdirAll(appDir, 0755)
	a.avatarsDir = filepath.Join(appDir, "avatars")
	os.MkdirAll(a.avatarsDir, 0755)
	path := filepath.Join(appDir, "data.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Fatal(err)
	}
	a.db = db
	db.Exec(`CREATE TABLE IF NOT EXISTS profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT, avatar_url TEXT, avatar_path TEXT)`)
	db.Exec(`ALTER TABLE profiles ADD COLUMN avatar_url TEXT`)
	db.Exec(`ALTER TABLE profiles ADD COLUMN avatar_path TEXT`)
	db.Exec(`CREATE TABLE IF NOT EXISTS videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT)`)
	// Reset interrupted downloads on boot (plan §3, task E8)
	db.Exec(`UPDATE videos SET status='pending' WHERE status IN ('downloading', 'paused')`)
	// Probe for yt-dlp binary (plan §4, task E5)
	if bin, err := ytdlp.Detect(); err == nil {
		a.ytdlpMissing = false
		a.ytdlpVersion = "yt-dlp " + bin.Version
	} else {
		a.ytdlpMissing = true
		a.ytdlpVersion = "yt-dlp Missing"
	}
}

// loadProfiles reads the profile list. UI thread only.
func (a *app) loadProfiles() {
	if a.db == nil {
		return
	}
	a.db.Exec("ALTER TABLE profiles ADD COLUMN avatar_url TEXT")
	a.db.Exec("ALTER TABLE profiles ADD COLUMN avatar_path TEXT")
	rows, err := a.db.Query("SELECT id,url,name,COALESCE(avatar_url, ''),COALESCE(avatar_path, '') FROM profiles")
	if err != nil {
		return
	}
	defer rows.Close()
	a.profiles = nil
	for rows.Next() {
		var p profile
		if err := rows.Scan(&p.id, &p.url, &p.name, &p.avatarURL, &p.avatarPath); err != nil {
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

func parseProfileName(u string) string {
	clean := strings.TrimRight(strings.TrimSpace(u), "/")
	if idx := strings.LastIndex(clean, "/"); idx >= 0 {
		sub := clean[idx+1:]
		if sub != "" {
			return sub
		}
	}
	if len(clean) > 20 {
		return clean[len(clean)-20:]
	}
	return clean
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
		name := parseProfileName(url)
		p := profile{url: url, name: name}

		a.db.Exec("ALTER TABLE profiles ADD COLUMN avatar_url TEXT")
		a.db.Exec("ALTER TABLE profiles ADD COLUMN avatar_path TEXT")

		res, err := a.db.Exec("INSERT INTO profiles(url, name, avatar_url, avatar_path) VALUES(?, ?, '', '')", p.url, p.name)
		if err != nil {
			updateFn := func() {
				a.fetching = false
				a.fetchErr = "Could not save profile: " + err.Error()
				a.showToast("Fetch Error", a.fetchErr, "error")
			}
			if a.win != nil {
				a.win.Update(updateFn)
			} else {
				updateFn()
			}
			return
		}
		p.id, _ = res.LastInsertId()

		// Metadata & Profile Image Fetch via yt-dlp
		var avatarURL string
		var avatarPath string
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "yt-dlp", "--flat-playlist", "-J", "--playlist-items", "1", url)
		out, err := cmd.Output()
		if err == nil && len(out) > 0 {
			var meta struct {
				Title      string `json:"title"`
				Uploader   string `json:"uploader"`
				Thumbnail  string `json:"thumbnail"`
				Thumbnails []struct {
					URL string `json:"url"`
				} `json:"thumbnails"`
				Entries []struct {
					Title      string `json:"title"`
					Thumbnails []struct {
						URL string `json:"url"`
					} `json:"thumbnails"`
				} `json:"entries"`
			}
			if json.Unmarshal(out, &meta) == nil {
				if meta.Title != "" {
					p.name = meta.Title
				} else if meta.Uploader != "" {
					p.name = meta.Uploader
				}
				if meta.Thumbnail != "" {
					avatarURL = meta.Thumbnail
				} else if len(meta.Thumbnails) > 0 && meta.Thumbnails[0].URL != "" {
					avatarURL = meta.Thumbnails[0].URL
				} else if len(meta.Entries) > 0 && len(meta.Entries[0].Thumbnails) > 0 {
					avatarURL = meta.Entries[0].Thumbnails[0].URL
				}
			}
		}

		if avatarURL != "" {
			p.avatarURL = avatarURL
			avatarsDir := a.avatarsDir
			if avatarsDir == "" {
				dir, _ := os.UserConfigDir()
				if dir == "" {
					dir = "."
				}
				avatarsDir = filepath.Join(dir, "nameless", "avatars")
			}
			os.MkdirAll(avatarsDir, 0755)
			destFile := filepath.Join(avatarsDir, fmt.Sprintf("profile_%d.jpg", p.id))
			req, rerr := http.NewRequestWithContext(ctx, "GET", avatarURL, nil)
			if rerr == nil {
				client := &http.Client{Timeout: 10 * time.Second}
				resp, derr := client.Do(req)
				if derr == nil && resp.StatusCode == http.StatusOK {
					imgData, rerr2 := io.ReadAll(resp.Body)
					resp.Body.Close()
					if rerr2 == nil && len(imgData) > 0 {
						if os.WriteFile(destFile, imgData, 0644) == nil {
							avatarPath = destFile
							p.avatarPath = avatarPath
						}
					}
				}
			}
		}

		a.db.Exec("UPDATE profiles SET name=?, avatar_url=?, avatar_path=? WHERE id=?", p.name, p.avatarURL, p.avatarPath, p.id)

		vids := a.loadVideos(p.id)
		doneFn := func() {
			a.fetching = false
			if p.avatarPath != "" {
				if data, err := os.ReadFile(p.avatarPath); err == nil {
					if src, err := ui.DecodeImage(data); err == nil {
						if a.imgCache == nil {
							a.imgCache = make(map[int64]ui.ImageSource)
						}
						a.imgCache[p.id] = src
					}
				}
			}
			a.profiles = append(a.profiles, p)
			a.applyProfile(p, vids)
			a.showToast("Profile Fetched", fmt.Sprintf("Loaded profile %s", p.name), "success")
		}

		if a.win != nil {
			a.win.Update(doneFn)
		} else {
			doneFn()
		}
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
	if a.ytdlpMissing {
		a.showToast("yt-dlp Missing", "Install yt-dlp first to start downloads", "error")
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
