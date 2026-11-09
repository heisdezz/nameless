package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
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
	"nameless/internal/tikdownloader"
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
	screenPlayer
)

type video struct {
	id         int64
	url        string
	title      string
	status     string // pending/downloading/done/failed/paused
	path       string
	pct        float64
	downloaded int64
	total      int64
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

	// Profile detail loading/pagination state. profileLoading shows the
	// loader until the async DB load finishes; profileLoadGen discards
	// stale loads when the user navigates quickly.
	profileLoading     bool
	profileLoadGen     int
	profileSearch      string    // debounced (applied) query
	profileSearchInput string    // raw field contents, edited per keystroke
	profileSearchSeen  string    // last raw value observed by viewProfile
	profileSearchAt    time.Time // when the raw value last changed
	profileSort        string
	profilePage        int
	profilePageSize    int
	thumbInflight      map[int64]bool

	// Profiles list state: cached per-profile video counts (one GROUP BY
	// query instead of a query per card per frame), decoded avatar cache,
	// and async loading/pagination state mirroring the profile page.
	profileCounts    map[int64]int
	avatarCache      map[int64]*ui.Bitmap
	avatarInflight   map[int64]bool
	profilesLoading  bool
	profilesPage     int
	profilesPageSize int

	// settings UI mirrors: persistent bindings for float/slider and
	// duration inputs (ui.Slider binds *float64; durations bind as text).
	// Owned by WS-D; see progress.md contract changes.
	radiusF64      float64
	workersF64     float64 // concurrent downloads (1..5, default 3)
	connectionsF64 float64 // connections per download (default 5)
	spacingSel     string
	delayStr       string
	timeoutStr     string
	saveByUsername bool // Save downloads by username toggle

	// yt-dlp detection state for the Settings screen (WS-D); ytdlpVersion
	// is filled by a background Detect and shown via win.Update.
	ytdlpVersion string
	ytdlpMissing bool
	detecting    bool
	ytdlBin      ytdlp.Binary

	// App-managed yt-dlp: binDir is nameless-owned storage for the
	// self-downloaded binary (config "ytdlp_path"); ytdlpInstalling guards
	// the background download; ytdlpInstallStatus/UpdateAvailable reflect
	// Settings/home UX state.
	binDir               string
	ytdlpInstalling      bool
	ytdlpInstallStatus   string
	ytdlpUpdateProgress  string
	ytdlpUpdateAvailable string
	ytdlpUpdateStatus    string

	// toasts holds transient notifications (errors, success alerts) dispatched
	// to the UI context on the next view pass.
	toasts []toastInfo

	// player view and thumbnails
	profileGridCols string
	currentVideoID  int64
	// player is the in-app system video player for the current video
	// (nil when the video is not downloaded, missing, or the system has
	// no media framework); playerVideoID is the video it plays and
	// playerPath the file the player is open over.
	player        *ui.VideoPlayer
	playerVideoID int64
	playerPath    string
	// playerSeek is the slider binding for the playhead position, in
	// seconds (app-owned so dragging persists across view rebuilds).
	playerSeek float64
	// playerFullScreen marks fullscreen video mode: the window goes
	// fullscreen and the player view collapses to just the viewport.
	playerFullScreen bool
	randomVideoIDs   []int64
	thumbsDir        string
	thumbCache       map[int64]*ui.Bitmap
	videoDurations   map[int64]string

	// animation state for video progress bars
	animTargetProgress map[int64]float64
	animFrame          int

	// animation state for theme transitions
	themeOld   config.ThemeConfig
	themeNew   config.ThemeConfig
	themeAnim  float32 // progress 0..1 across frames
	themeDirty bool    // set true when theme values change
}

func defaultConfig() *config.Config {
	return &config.Config{
		DefaultDownloadLocation: "./downloads",
		Workers:                 3, // concurrent downloads, max 5
		ConnectionsPerDownload:  5, // connections per single file
		Delay:                   time.Second,
		PerVideoTimeout:         5 * time.Minute,
		Theme: config.ThemeConfig{
			AccentHex: "#4f46e5",
			Radius:    12,
			Spacing:   4,
		},
		TikDownloaderAPI:    "https://tikdownloader.io",
		TikDownloaderFormat: "mp4",
		SaveByUsername:      false,
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

// sanitizeSegment makes a string safe for use as a directory name,
// mirroring tikdownloader's internal sanitizer logic.
func sanitizeSegment(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('_')
		default:
		}
	}
	out := strings.Trim(b.String(), "_")
	if len(out) > 80 {
		out = strings.Trim(out[:80], "_")
	}
	return out
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
		a.stopPlayer()
		a.currentProfile = nil
		a.videos = nil
		a.screen = screenProfilesList
	}
	a.refreshProfileCounts()
	a.showToast("Profile Removed", "Profile was deleted from archive", "info")
}

// detectYTDLP re-probes yt-dlp (custom path first, then PATH) and
// refreshes the Settings engine status. Plan §4, milestone 3 task E5.
func (a *app) detectYTDLP() {
	if a.probeYtdlp() {
		a.showToast("yt-dlp Ready", "Detected "+a.ytdlpVersion, "success")
	} else {
		a.showToast("yt-dlp Missing", "yt-dlp was not found (custom path or system PATH)", "error")
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
	a.stopPlayer() // leaving the player screen (if we were on it)
	if a.cfg == nil {
		a.cfg = defaultConfig()
	}
	if a.outDir == "" {
		a.outDir = a.cfg.DefaultDownloadLocation
	}
	// Initialize save-by-username toggle from config
	a.saveByUsername = a.cfg.SaveByUsername
	a.radiusF64 = float64(a.cfg.Theme.Radius)
	a.workersF64 = float64(a.cfg.EffectiveConcurrentDownloads())
	a.connectionsF64 = float64(a.cfg.EffectiveConnectionsPerDownload())
	a.spacingSel = spacingLabel(a.cfg.Theme.Spacing)
	a.delayStr = a.cfg.Delay.String()
	a.timeoutStr = a.cfg.PerVideoTimeout.String()

	// Capture current theme as "old" for transition animation
	a.themeOld = a.cfg.Theme
	a.themeNew = a.cfg.Theme
	a.themeAnim = 0
	a.themeDirty = true

	a.screen = screenSettings
}

// persistConfig writes the current settings to config.json (best effort).
// Used by flows that don't go through the Settings page, e.g. recording the
// managed yt-dlp path after an auto-install.
func (a *app) persistConfig() {
	if a.cfg == nil {
		return
	}
	cfgPath := a.cfgPath
	if cfgPath == "" {
		cfgPath = "config.json"
	}
	if err := a.cfg.Save(cfgPath); err != nil {
		a.showToast("Save Error", err.Error(), "error")
	}
}

// saveSettings persists the in-memory settings to config.json.
func (a *app) saveSettings() {
	if a.cfg == nil {
		return
	}
	if a.outDir != "" {
		a.cfg.DefaultDownloadLocation = a.outDir
	}
	// Concurrent downloads: clamp to 1..5 (default 3), never 16.
	if a.workersF64 >= 1 {
		a.cfg.Workers = int(a.workersF64)
		if a.cfg.Workers > config.MaxWorkers {
			a.cfg.Workers = config.MaxWorkers
		}
	}
	// Connections per download: separate limit, default 5, max 16.
	if a.connectionsF64 >= 1 {
		a.cfg.ConnectionsPerDownload = int(a.connectionsF64)
		if a.cfg.ConnectionsPerDownload > config.MaxConnectionsPerDownload {
			a.cfg.ConnectionsPerDownload = config.MaxConnectionsPerDownload
		}
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
	a.cfg.SaveByUsername = a.saveByUsername
	// Track theme change for animation
	a.themeNew = a.cfg.Theme
	a.themeDirty = true
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
	a.workersF64 = float64(a.cfg.EffectiveConcurrentDownloads())
	a.connectionsF64 = float64(a.cfg.EffectiveConnectionsPerDownload())
	a.spacingSel = spacingLabel(a.cfg.Theme.Spacing)
	a.delayStr = a.cfg.Delay.String()
	a.timeoutStr = a.cfg.PerVideoTimeout.String()
	// Reset save-by-username toggle
	a.saveByUsername = false
	// Reset theme animation state
	a.themeNew = a.cfg.Theme
	a.themeAnim = 0
	a.themeDirty = true
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
	os.MkdirAll(appDir, 0o755)
	a.avatarsDir = filepath.Join(appDir, "avatars")
	os.MkdirAll(a.avatarsDir, 0o755)
	a.thumbsDir = filepath.Join(appDir, "thumbs")
	os.MkdirAll(a.thumbsDir, 0o755)
	a.binDir = filepath.Join(appDir, "bin")
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
	db.Exec(`ALTER TABLE videos ADD COLUMN position INTEGER`)
	// Reset interrupted downloads on boot (plan §3, task E8)
	db.Exec(`UPDATE videos SET status='pending' WHERE status IN ('downloading', 'paused')`)
	// Probe for yt-dlp binary (plan §4, task E5): prefer the app-managed
	// binary recorded in config.json, then PATH; when neither exists, the
	// launch prompt offers to download nameless' own copy into its storage.
	if a.probeYtdlp() {
		// Refresh an app-managed install opportunistically? The user must
		// opt in via Settings (no surprise downloads on boot).
	} else {
		a.showToast("yt-dlp not found",
			"Install it system-wide, or let nameless download and manage its own copy — see the banner on Home or Settings → Engine",
			"error")
	}
}

// ytdlpBinDir returns the app-owned storage for the managed yt-dlp binary.
func (a *app) ytdlpBinDir() string {
	if a.binDir != "" {
		return a.binDir
	}
	dir, _ := os.UserConfigDir()
	if dir == "" {
		dir = "."
	}
	return filepath.Join(dir, "nameless", "bin")
}

// probeYtdlp sets the yt-dlp detection state: an app-managed binary
// (config "ytdlp_path") is preferred, then PATH. Reports whether yt-dlp
// is usable. Startup/main thread only.
func (a *app) probeYtdlp() bool {
	if a.cfg != nil && a.cfg.YTDLPPath != "" {
		if bin, err := ytdlp.DetectAt(a.cfg.YTDLPPath); err == nil {
			a.ytdlpMissing = false
			a.ytdlpVersion = "yt-dlp " + bin.Version
			a.ytdlBin = bin
			return true
		}
		// The recorded binary is gone/broken: fall through to PATH and
		// treat the managed copy as benefiting from a re-download.
	}
	if bin, err := ytdlp.Detect(); err == nil {
		a.ytdlpMissing = false
		a.ytdlpVersion = "yt-dlp " + bin.Version
		a.ytdlBin = bin
		return true
	}
	a.ytdlpMissing = true
	a.ytdlpVersion = "yt-dlp Missing"
	a.ytdlBin = ytdlp.Binary{}
	return false
}

// onUI runs fn on the UI thread when a window exists; headless runs (tests)
// execute inline so shared state stays on the caller goroutine.
func (a *app) onUI(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
		return
	}
	fn()
}

// downloadYtdlp installs (or, when one is already managed, updates /
// re-downloads) the standalone yt-dlp binary into nameless' own storage,
// records the path under config "ytdlp_path", and re-probes detection.
func (a *app) downloadYtdlp() {
	if a.ytdlpInstalling {
		return
	}
	refresh := a.ytdlBin.Path != ""
	body := func() {
		ver, verr := ytdlp.LatestVersion(context.Background())
		a.onUI(func() {
			if verr == nil {
				a.ytdlpUpdateProgress = ""
				a.ytdlpInstallStatus = fmt.Sprintf("Downloading yt-dlp %s…", ver)
			} else {
				a.ytdlpInstallStatus = "Resolving latest yt-dlp release…"
			}
		})
		bin, err := ytdlp.InstallBinary(context.Background(), a.ytdlpBinDir(), func(done, total int64) {
			a.onUI(func() {
				if total > 0 {
					a.ytdlpUpdateProgress = fmt.Sprintf("%d%%", int(100*float64(done)/float64(total)))
				} else {
					a.ytdlpUpdateProgress = fmt.Sprintf("%d kB", done/1024)
				}
			})
		})
		a.onUI(func() {
			a.ytdlpInstalling = false
			a.ytdlpUpdateProgress = ""
			if err != nil {
				a.ytdlpInstallStatus = ""
				a.showToast("yt-dlp Download Failed", err.Error(), "error")
				return
			}
			a.ytdlBin = bin
			a.ytdlpMissing = false
			a.ytdlpVersion = "yt-dlp " + bin.Version
			if a.cfg == nil {
				a.cfg = defaultConfig()
			}
			a.cfg.YTDLPPath = bin.Path // config entry for the managed binary
			a.cfg.InstallYTDLP = true  // managed by nameless, update/redownload in Settings
			a.persistConfig()
			if refresh {
				a.ytdlpInstallStatus = "yt-dlp re-downloaded"
				a.showToast("yt-dlp Updated", fmt.Sprintf("nameless-managed yt-dlp re-downloaded (%s)", bin.Version), "success")
			} else {
				a.ytdlpInstallStatus = "yt-dlp installed by nameless"
				a.showToast("yt-dlp Installed", fmt.Sprintf("Downloaded yt-dlp %s into nameless storage", bin.Version), "success")
			}
			a.ytdlpUpdateAvailable = ""
		})
	}
	if a.win == nil {
		// Headless (tests): synchronous, race-free.
		a.ytdlpInstalling = true
		body()
		return
	}
	a.ytdlpInstalling = true
	go body()
}

// checkYtdlpUpdate compares the active yt-dlp against the latest official
// release and flags the outcome for the Settings UI and toasts.
func (a *app) checkYtdlpUpdate() {
	if a.ytdlpInstalling {
		return
	}
	if a.ytdlBin.Path == "" {
		a.showToast("No yt-dlp yet", "Install/download yt-dlp first, then check for updates", "error")
		return
	}
	cur := a.ytdlBin.Version
	body := func() {
		latest, err := ytdlp.LatestVersion(context.Background())
		a.onUI(func() {
			switch {
			case err != nil:
				a.showToast("Update Check Failed", err.Error(), "error")
			case cur == "" || ytdlp.IsNewerVersion(cur, latest):
				a.ytdlpUpdateAvailable = latest
				a.showToast("Update Available", fmt.Sprintf("yt-dlp %s is available — press Re-download to update", latest), "info")
			default:
				a.ytdlpUpdateAvailable = ""
				a.ytdlpUpdateStatus = "yt-dlp is up to date"
				a.showToast("Up to Date", fmt.Sprintf("yt-dlp %s is the latest release", cur), "success")
			}
		})
	}
	if a.win == nil {
		body()
		return
	}
	go body()
}

// refreshProfileCounts recomputes per-profile video counts with a single
// GROUP BY query. Called on startup and when the profiles list is shown,
// never per frame.
func (a *app) refreshProfileCounts() {
	a.profileCounts = make(map[int64]int)
	if a.db == nil {
		return
	}
	rows, err := a.db.Query("SELECT profile_id, COUNT(*) FROM videos GROUP BY profile_id")
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, n int64
		if err := rows.Scan(&id, &n); err == nil {
			a.profileCounts[id] = int(n)
		}
	}
}

// showProfilesList navigates to the profiles directory, refreshing the
// list and counts off the UI thread behind the loading state.
func (a *app) showProfilesList() {
	a.screen = screenProfilesList
	a.profilesLoading = true
	if a.profilesPageSize <= 0 {
		a.profilesPageSize = 20
	}
	go func() {
		a.loadProfiles()
		a.onUI(func() { a.profilesLoading = false })
	}()
}

// loadAvatar decodes one profile avatar into the cache (once, in-flight
// guarded) and kicks a redraw. Replaces per-frame file reads in the view.
func (a *app) loadAvatar(id int64, avatarPath string) {
	if avatarPath == "" {
		return
	}
	a.mu.Lock()
	if a.avatarCache != nil {
		if _, ok := a.avatarCache[id]; ok {
			a.mu.Unlock()
			return
		}
	}
	if a.avatarInflight == nil {
		a.avatarInflight = make(map[int64]bool)
	}
	if a.avatarInflight[id] {
		a.mu.Unlock()
		return
	}
	a.avatarInflight[id] = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.avatarInflight, id)
		a.mu.Unlock()
	}()
	data, err := os.ReadFile(avatarPath)
	if err != nil {
		return
	}
	bmp, err := ui.DecodeBitmap(data)
	if err != nil {
		return
	}
	a.mu.Lock()
	if a.avatarCache == nil {
		a.avatarCache = make(map[int64]*ui.Bitmap)
	}
	a.avatarCache[id] = bmp
	a.mu.Unlock()
	if a.win != nil {
		a.win.Update(func() {})
	}
}

// loadProfiles reads the profile list. UI thread only.
func (a *app) loadProfiles() {
	if a.db == nil {
		return
	}
	a.db.Exec("ALTER TABLE profiles ADD COLUMN avatar_url TEXT")
	a.db.Exec("ALTER TABLE profiles ADD COLUMN avatar_path TEXT")
	rows, err := a.db.Query("SELECT id,url,COALESCE(name, ''),COALESCE(avatar_url, ''),COALESCE(avatar_path, '') FROM profiles")
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
	rows.Close()
	a.refreshProfileCounts()
	for _, p := range a.profiles {
		if p.avatarPath != "" {
			go a.loadAvatar(p.id, p.avatarPath)
		}
	}
}

// loadVideos queries a profile's videos; safe on any thread (DB only, no app state).
func (a *app) loadVideos(profileID int64) []video {
	rows, err := a.db.Query("SELECT id,url,title,status,COALESCE(path, '') FROM videos WHERE profile_id=?", profileID)
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
	a.stopPlayer() // leaving the player screen (if we were on it)
	pcopy := p
	a.currentProfile = &pcopy
	a.screen = screenProfile
	a.videos = vids
	for _, v := range vids {
		if v.path != "" {
			go a.ensureThumbnail(v.id, v.path)
		}
	}
}

// normalizeProfileURL canonicalizes any accepted profile input to a canonical
// TikTok profile URL so equivalent variants compare equal:
//   - "https://www.tiktok.com/@user/extra?x=y" → "https://tiktok.com/@user/extra"
//   - "tiktok.com/@user" → "https://tiktok.com/@user"
//   - "@user" and bare "user" → "https://tiktok.com/@user"
//
// Handle case is preserved (use profileURLKey for equality checks).
func normalizeProfileURL(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	u = strings.SplitN(u, "?", 2)[0]
	u = strings.SplitN(u, "#", 2)[0]
	u = strings.TrimRight(u, "/")
	switch {
	case strings.HasPrefix(u, "http://") || strings.HasPrefix(u, "https://"):
		idx := strings.Index(u, "://")
		rest := u[idx+3:]
		host, path := rest, ""
		if p := strings.Index(rest, "/"); p >= 0 {
			host, path = rest[:p], rest[p:]
		}
		return u[:idx] + "://" + strings.TrimPrefix(strings.ToLower(host), "www.") + path
	case strings.HasPrefix(u, "@"):
		return "https://tiktok.com/" + u
	case strings.Contains(u, "/"):
		// Host/path without a scheme: prefix the canonical scheme.
		return "https://" + u
	default:
		// Bare token: treat it as an @handle ("ne_edless" → @ne_edless).
		return "https://tiktok.com/@" + u
	}
}

// profileURLKey is the comparison key for duplicate detection: the @handle
// part is case-insensitive on TikTok.
func profileURLKey(u string) string {
	return strings.ToLower(normalizeProfileURL(u))
}

// playlistMeta is the subset of yt-dlp's `--flat-playlist -J` output the
// fetch flow consumes: channel-level info plus flat video entries.
type playlistMeta struct {
	Title      string `json:"title"`
	Uploader   string `json:"uploader"`
	Thumbnail  string `json:"thumbnail"`
	Thumbnails []struct {
		URL string `json:"url"`
	} `json:"thumbnails"`
	Entries []playlistEntry `json:"entries"`
}

type playlistEntry struct {
	ID         string `json:"id"`
	URL        string `json:"url"`
	Title      string `json:"title"`
	Thumbnails []struct {
		URL string `json:"url"`
	} `json:"thumbnails"`
}

// validVideoURL reports whether u looks like an actual video link. TikTok
// video pages end in "/video/<numeric id>", so placeholder replies (e.g.
// ".../video/sample") and non-video entries are rejected — a bogus or
// failing yt-dlp response can then never pass as a fetched profile.
func validVideoURL(u string) bool {
	u = strings.TrimSpace(u)
	idx := strings.LastIndex(u, "/video/")
	if idx < 0 {
		return false
	}
	id := strings.TrimRight(u[idx+len("/video/"):], "/")
	if id == "" {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// upsertFetchedVideos reconciles the profile's videos with the fetched
// playlist: inserts new entries in playlist order and keeps existing rows as
// they are, so statuses and finished downloads survive a re-fetch.
// Database only; no app state touched. Returns the number added.
func (a *app) upsertFetchedVideos(p *profile, meta playlistMeta) int {
	if a.db == nil {
		return 0
	}
	known := make(map[string]bool)
	for _, v := range a.loadVideos(p.id) {
		known[v.url] = true
	}
	n := 0
	for i, e := range meta.Entries {
		if e.URL == "" || known[e.URL] || !validVideoURL(e.URL) {
			continue
		}
		title := e.Title
		if title == "" {
			title = "Video " + e.ID
		}
		a.db.Exec("INSERT INTO videos(profile_id, url, title, status, position) VALUES(?, ?, ?, 'pending', ?)", p.id, e.URL, title, i)
		n++
	}
	return n
}

// existingProfile looks up an already-saved profile by canonical URL. It
// normalizes both sides so rows saved before normalization (raw pasted
// strings) still match. Returns (0, "") when the account was never added.
func (a *app) existingProfile(url string) (int64, string) {
	if a.db == nil {
		return 0, ""
	}
	key := profileURLKey(url)
	rows, err := a.db.Query("SELECT id, url, name FROM profiles")
	if err != nil {
		return 0, ""
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var u, name string
		if err := rows.Scan(&id, &u, &name); err != nil {
			return 0, ""
		}
		if profileURLKey(u) == key {
			return id, name
		}
	}
	return 0, ""
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
	url := normalizeProfileURL(a.profileURL)
	if url == "" {
		a.fetchErr = "Enter a valid profile URL"
		a.showToast("Error", "Enter a valid profile URL", "error")
		return
	}

	// Same account, one row: if it's already saved, typing it again
	// *refreshes* the existing profile (name, avatar, video list resolve
	// here) instead of creating a duplicate row.
	existingID, _ := a.existingProfile(url)

	a.fetching = true
	a.fetchErr = ""

	run := func() {
		// Report a fetch failure; nothing has been persisted yet, so a
		// failed fetch never leaves a stub profile row in SQLite.
		fail := func(msg string) {
			a.onUI(func() {
				a.fetching = false
				a.fetchErr = msg
				a.showToast("Fetch Error", msg, "error")
			})
		}

		// 1. Fetch the user's actual data via yt-dlp FIRST (flat playlist
		// JSON: channel info, avatar and every video entry).
		var avatarURL string
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		fetchArgs := append([]string{"--flat-playlist", "-J"}, ytdlp.AuthArgs(a.cfg)...)
		fetchArgs = append(fetchArgs, url)
		cmd := exec.CommandContext(ctx, a.ytDLPBin(a.cfg), fetchArgs...)
		out, err := cmd.Output()
		if err != nil {
			msg := "yt-dlp failed to fetch profile info: " + err.Error()
			var ee *exec.Error
			if errors.As(err, &ee) && errors.Is(ee.Err, exec.ErrNotFound) {
				msg = "yt-dlp is not installed — install it, press Detect, then retry"
			} else if ctx.Err() == context.DeadlineExceeded {
				msg = "Fetching profile info timed out — check the URL and network, then retry"
			}
			fail(msg)
			return
		}
		if len(out) == 0 {
			fail("yt-dlp returned no profile data for this URL — check the URL, then retry")
			return
		}
		var meta playlistMeta
		if json.Unmarshal(out, &meta) != nil {
			fail("Could not parse yt-dlp profile data — update yt-dlp, then retry")
			return
		}
		// A reply counts as a real profile only if yt-dlp identified the
		// account (title/uploader) or it contains at least one genuine
		// video link. Anything else — empty result, placeholder entries —
		// is treated as "profile not found" and persists nothing.
		validVideos := 0
		for _, e := range meta.Entries {
			if validVideoURL(e.URL) {
				validVideos++
			}
		}
		if meta.Title == "" && meta.Uploader == "" && validVideos == 0 {
			fail("yt-dlp found no profile or videos for this URL — check the handle and retry")
			return
		}
		name := meta.Title
		if name == "" {
			name = meta.Uploader
		}
		if name == "" {
			name = parseProfileName(url)
		}
		if meta.Thumbnail != "" {
			avatarURL = meta.Thumbnail
		} else if len(meta.Thumbnails) > 0 && meta.Thumbnails[0].URL != "" {
			avatarURL = meta.Thumbnails[0].URL
		} else if len(meta.Entries) > 0 && len(meta.Entries[0].Thumbnails) > 0 {
			avatarURL = meta.Entries[0].Thumbnails[0].URL
		}
		p := profile{url: url, name: name, avatarURL: avatarURL, id: existingID}

		// 3. Persist: update an existing row in place, insert only when the
		// account is new — one row per account, ever.
		a.db.Exec("ALTER TABLE profiles ADD COLUMN avatar_url TEXT")
		a.db.Exec("ALTER TABLE profiles ADD COLUMN avatar_path TEXT")

		if p.id == 0 {
			res, err := a.db.Exec("INSERT INTO profiles(url, name, avatar_url, avatar_path) VALUES(?, ?, '', '')", p.url, p.name)
			if err != nil {
				fail("Could not save profile: " + err.Error())
				return
			}
			p.id, _ = res.LastInsertId()
		} else {
			// Refresh: adopt the canonical URL form and the fetched name.
			a.db.Exec("UPDATE profiles SET url=?, name=? WHERE id=?", p.url, p.name, p.id)
		}

		// Profile Image download using the id issued above.
		var avatarPath string

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

		// 4. Reconcile the video list with the fetched entries: new videos
		// are inserted pending at their playlist position, existing rows are
		// kept so finished downloads survive a re-fetch.
		a.upsertFetchedVideos(&p, meta)

		vids := a.loadVideos(p.id)
		refreshed := existingID != 0
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
			// Replace the in-memory entry on refresh; append only new profiles.
			replaced := false
			for i := range a.profiles {
				if a.profiles[i].id == p.id {
					a.profiles[i] = p
					replaced = true
					break
				}
			}
			if !replaced {
				a.profiles = append(a.profiles, p)
			}
			a.applyProfile(p, vids)
			if refreshed {
				a.showToast("Profile Updated", fmt.Sprintf("Refreshed %s — %d videos found", p.name, len(vids)), "success")
			} else {
				a.showToast("Profile Fetched", fmt.Sprintf("Loaded profile %s", p.name), "success")
			}
			// Kick off the batch download of every fetched video right
			// after landing on the profile screen (real UI sessions only).
			if a.win != nil && !a.ytdlpMissing {
				a.startDownload()
			}
		}

		if a.win != nil {
			a.win.Update(doneFn)
		} else {
			doneFn()
		}
	}

	if a.win == nil {
		// Headless (tests): there is no window that drains posts, so run
		// the whole fetch inline on the caller thread — deterministic and
		// race-free to observe. The real app always runs it off-thread.
		run()
		return
	}
	go run()
}

// openProfile navigates to a profile by value (never a pointer into a slice
// that may be appended to later). UI thread only.
// The video list is loaded off the UI thread: the profile screen shows
// its loading state immediately instead of freezing on the DB query
// (which is slow with many videos) and thumbnail extraction.
func (a *app) openProfile(p profile) {
	pcopy := p
	a.currentProfile = &pcopy
	a.screen = screenProfile
	a.videos = nil
	a.profilePage = 1
	a.profileSearch = ""
	a.profileSearchInput = ""
	a.profileSearchSeen = ""
	a.profileSort = "Default"
	if a.profilePageSize <= 0 {
		a.profilePageSize = 20
	}
	a.profileLoading = true
	a.profileLoadGen++
	gen := a.profileLoadGen
	profileID := p.id
	go func() {
		vids := a.loadVideos(profileID)
		a.onUI(func() {
			if gen != a.profileLoadGen {
				return // user navigated away; drop this stale load
			}
			a.videos = vids
			a.profileLoading = false
			for _, v := range vids {
				if v.path != "" {
					go a.ensureThumbnail(v.id, v.path)
				}
			}
		})
	}()
}

func (a *app) startDownload() {
	if a.currentProfile == nil {
		return
	}
	// yt-dlp is still needed for fetching profile metadata (video URLs)
	if a.ytdlpMissing {
		a.showToast("yt-dlp Missing", "Install yt-dlp first to fetch profiles, then press Detect", "error")
		return
	}
	if a.downloading {
		return
	}
	cfg := a.cfg
	if cfg == nil {
		cfg = defaultConfig()
	}
	// Use tikdownloader.io for actual video downloading. Connections per
	// download (default 5) is separate from concurrent downloads below.
	dl := tikdownloader.New(cfg.TikDownloaderAPI, cfg.TikDownloaderCookies, effectiveTikFormat(cfg), cfg.EffectiveConnectionsPerDownload())
	outDir := a.effectiveOutputDir()
	// If enabled, save videos in per-uploader subdirectories
	// We'll compute the per-video directory after fetching metadata.
	vidsSnapshot := make([]video, len(a.videos))
	copy(vidsSnapshot, a.videos)
	a.setPaused(false)
	a.downloading = true

	go func() {
		// Fetch profile with yt-dlp to get video URLs and metadata
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		fetchArgs := append([]string{"--flat-playlist", "-J"}, ytdlp.AuthArgs(cfg)...)
		fetchArgs = append(fetchArgs, a.currentProfile.url)
		cmd := exec.CommandContext(ctx, a.ytDLPBin(cfg), fetchArgs...)
		out, err := cmd.Output()
		if err != nil {
			a.showToast("Fetch Error", "yt-dlp failed to fetch profile info", "error")
			a.downloading = false
			return
		}

		// Parse the flat playlist JSON to get video entries
		var meta struct {
			Title    string `json:"title"`
			Uploader string `json:"uploader"`
			Entries  []struct {
				ID         string `json:"id"`
				URL        string `json:"url"`
				Title      string `json:"title"`
				UploadDate string `json:"upload_date"`
			} `json:"entries"`
		}
		if json.Unmarshal(out, &meta) != nil {
			a.showToast("Parse Error", "Could not parse yt-dlp profile data", "error")
			a.downloading = false
			return
		}

		// Determine per-uploader directory if SaveByUsername is enabled
		var uploaderDir string
		if cfg.SaveByUsername && meta.Uploader != "" {
			// Sanitize the uploader name for use as a directory
			uploaderDir = filepath.Join(outDir, sanitizeSegment(meta.Uploader))
			if err := os.MkdirAll(uploaderDir, 0o755); err != nil {
				a.showToast("Directory Error", fmt.Sprintf("Failed to create uploader subdirectory: %v", err), "error")
				a.downloading = false
				return
			}
		}

		// Download videos using tikdownloader.io, up to
		// EffectiveConcurrentDownloads (default 3, max 5) at once.
		// ConnectionsPerDownload is handled inside tikdownloader per file.
		type job struct {
			id       int64
			videoURL string
			meta     tikdownloader.VideoMeta
		}
		jobs := make(chan job)
		var wg sync.WaitGroup
		workers := cfg.EffectiveConcurrentDownloads()
		for w := 0; w < workers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := range jobs {
					a.waitWhilePaused()
					id := j.id
					a.onUI(func() { a.setVideoStatus(id, "downloading") })
					// Per-video timeout from Settings (default 5m).
					dlCtx, dlCancel := context.WithTimeout(context.Background(), cfg.PerVideoTimeout)
					// Use uploader subdirectory if configured
					dlTargetDir := outDir
					if uploaderDir != "" {
						dlTargetDir = uploaderDir
					}
					result, err := dl.DownloadVideo(dlCtx, j.videoURL, dlTargetDir)
					dlCancel()
					if err != nil {
						if a.db != nil {
							a.db.Exec("UPDATE videos SET status=?, err=? WHERE id=?", "failed", err.Error(), id)
						}
						a.onUI(func() { a.setVideoStatus(id, "failed") })
						continue
					}
					// Apply the naming template, then fetch sidecars.
					finalPath := a.renameWithTemplate(cfg, result.Path, j.meta)
					_ = a.downloadSidecars(cfg, j.videoURL, finalPath)
					if a.db != nil {
						a.db.Exec("UPDATE videos SET status=?, path=? WHERE id=?", "done", finalPath, id)
					}
					a.onUI(func() { a.setVideoDone(id, finalPath) })
					// Delay between downloads from Settings.
					if cfg.Delay > 0 {
						a.waitWhilePaused()
						time.Sleep(cfg.Delay)
					}
				}
			}()
		}
		for i := range meta.Entries {
			entry := meta.Entries[i]
			if entry.URL == "" {
				continue
			}
			// Map to the on-screen video by URL, falling back to position.
			id := int64(i + 1)
			for _, v := range vidsSnapshot {
				if v.url == entry.URL {
					id = v.id
					break
				}
			}
			jobs <- job{id: id, videoURL: entry.URL, meta: tikdownloader.VideoMeta{ID: entry.ID, Title: entry.Title, Uploader: meta.Title, UploadDate: entry.UploadDate}}
		}
		close(jobs)
		wg.Wait()

		a.downloading = false
		a.showToast("Download Complete", fmt.Sprintf("All videos downloaded via tikdownloader.io (%d entries)", len(meta.Entries)), "success")
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
			go a.ensureThumbnail(id, path)
			return
		}
	}
}

// setVideoProgress updates a video's progress percentage and byte counts. UI thread only.
func (a *app) setVideoProgress(id int64, pct float64, downloaded int64) {
	for i := range a.videos {
		if a.videos[i].id == id {
			a.videos[i].pct = pct
			a.videos[i].downloaded = downloaded
			// total is inferred or set separately
			return
		}
	}
}

// retryFailed sets all failed videos to pending. UI thread only.
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

// advanceVideoProgress animates one frame of video progress.
// Should be called at the start of each view loop iteration when a download
// is in progress. It advances the progress for the given video id by one frame.
func (a *app) advanceVideoProgress(id int64) {
	if a.animTargetProgress == nil {
		return
	}
	target, hasTarget := a.animTargetProgress[id]
	if !hasTarget {
		return
	}

	// Simple frame-based animation: advance 1/20 of the way per frame
	// (assuming ~60fps, this gives ~20 frames = ~333ms, close to AnimationDuration 200ms)
	const framesForFullAnimation = 20
	a.animFrame++

	// Calculate progress: from 0 to 1 over framesForFullAnimation frames
	progress := float64(a.animFrame) / float64(framesForFullAnimation)

	// Clamp to [0, 1]
	if progress > 1.0 {
		progress = 1.0
		a.animFrame = 0              // reset for next animation
		a.animTargetProgress[id] = 0 // clear target
	}

	// Compute intermediate percentage
	pct := progress * target
	a.win.Update(func() {
		a.setVideoProgress(id, pct, int64(progress*100)) // approximate total/downloaded
	})
}

// animateVideoProgress animates a video's progress percentage from pctFrom to pctTo
// using MyGo's frame-based animation system. The animation advances each frame
// via the view loop, aiming to reach pctTo over AnimationDuration (200ms).
// Call this once per video when starting a download, then the view loop will
// advance the progress each frame via advanceVideoProgress.
func (a *app) animateVideoProgress(id int64, pctFrom, pctTo float64) {
	// Set the target progress for this video
	if a.animTargetProgress == nil {
		a.animTargetProgress = make(map[int64]float64)
	}
	a.animTargetProgress[id] = pctTo

	// Mark that we start animation from pctFrom
	// The view loop will compute intermediate values each frame
	a.animFrame = 0 // reset frame counter
}

// ensureThumbnail generates a video thumbnail using ffmpeg if not already cached.
// thumbSem limits concurrent ffmpeg thumbnail extractions.
var thumbSem = make(chan struct{}, 3)

func (a *app) ensureThumbnail(id int64, videoPath string) {
	if videoPath == "" {
		return
	}
	// One in-flight job per video: viewProfile runs every frame, and
	// without this guard it spawned a new ffmpeg process per frame.
	a.mu.Lock()
	if a.thumbCache != nil {
		if _, ok := a.thumbCache[id]; ok {
			a.mu.Unlock()
			return
		}
	}
	if a.thumbInflight == nil {
		a.thumbInflight = make(map[int64]bool)
	}
	if a.thumbInflight[id] {
		a.mu.Unlock()
		return
	}
	a.thumbInflight[id] = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		delete(a.thumbInflight, id)
		a.mu.Unlock()
	}()
	thumbSem <- struct{}{}
	defer func() { <-thumbSem }()
	if a.thumbsDir == "" {
		dir, _ := os.UserConfigDir()
		if dir == "" {
			dir = "."
		}
		a.thumbsDir = filepath.Join(dir, "nameless", "thumbs")
		_ = os.MkdirAll(a.thumbsDir, 0o755)
	}
	thumbPath := filepath.Join(a.thumbsDir, fmt.Sprintf("thumb_%d.jpg", id))
	if _, err := os.Stat(thumbPath); err == nil {
		a.loadThumbnail(id, thumbPath)
		a.probeDuration(id, videoPath)
		return
	}
	if _, err := os.Stat(videoPath); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-ss", "00:00:01", "-i", videoPath, "-frames:v", "1", "-update", "1", thumbPath)
	if err := cmd.Run(); err == nil {
		a.loadThumbnail(id, thumbPath)
	}
	a.probeDuration(id, videoPath)
}

// loadThumbnail decodes a thumbnail file into the in-memory cache and triggers a redraw.
func (a *app) loadThumbnail(id int64, thumbPath string) {
	data, err := os.ReadFile(thumbPath)
	if err != nil {
		return
	}
	bmp, err := ui.DecodeBitmap(data)
	if err != nil {
		return
	}
	a.mu.Lock()
	if a.thumbCache == nil {
		a.thumbCache = make(map[int64]*ui.Bitmap)
	}
	a.thumbCache[id] = bmp
	a.mu.Unlock()
	if a.win != nil {
		a.win.Update(func() {})
	}
}

// probeDuration probes the video duration in seconds via ffprobe and formats as M:SS.
func (a *app) probeDuration(id int64, videoPath string) {
	a.mu.Lock()
	if a.videoDurations != nil {
		if _, ok := a.videoDurations[id]; ok {
			a.mu.Unlock()
			return
		}
	}
	a.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", videoPath)
	out, err := cmd.Output()
	if err != nil {
		return
	}
	secStr := strings.TrimSpace(string(out))
	var sec float64
	if _, err := fmt.Sscanf(secStr, "%f", &sec); err == nil && sec > 0 {
		m := int(sec) / 60
		s := int(sec) % 60
		durStr := fmt.Sprintf("%d:%02d", m, s)
		a.mu.Lock()
		if a.videoDurations == nil {
			a.videoDurations = make(map[int64]string)
		}
		a.videoDurations[id] = durStr
		a.mu.Unlock()
		if a.win != nil {
			a.win.Update(func() {})
		}
	}
}

// playExternal launches the video in mpv (preferred) or xdg-open.
func (a *app) playExternal(videoPath string) {
	if videoPath == "" {
		a.showToast("Cannot play", "Video has not been downloaded yet", "error")
		return
	}
	if _, err := os.Stat(videoPath); err != nil {
		a.showToast("File missing", "Video file not found at "+videoPath, "error")
		return
	}
	go func() {
		if mpvPath, err := exec.LookPath("mpv"); err == nil {
			_ = exec.Command(mpvPath, videoPath).Run()
			return
		}
		if xdgPath, err := exec.LookPath("xdg-open"); err == nil {
			_ = exec.Command(xdgPath, videoPath).Run()
			return
		}
	}()
	a.showToast("Playing video", "Opened in system player", "info")
}
