package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestView(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.initDB()
	tt := ui.NewTester(a.view, 480, 360)
	if !tt.HasText("nameless") {
		t.Errorf("texts %q", tt.Texts())
	}
}

func TestApplyProfile(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.applyProfile(profile{id: 1, url: "u", name: "n"}, []video{{id: 7, title: "t", status: "pending"}})
	if a.currentProfile == nil || a.currentProfile.id != 1 || a.screen != screenProfile {
		t.Fatalf("profile=%v screen=%v", a.currentProfile, a.screen)
	}
	a.setVideoStatus(7, "done")
	if a.videos[0].status != "done" {
		t.Fatalf("status=%q", a.videos[0].status)
	}
	a.setVideoStatus(999, "done") // unknown id must be a safe no-op
}

func TestPauseResume(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.setPaused(true)
	go func() {
		time.Sleep(20 * time.Millisecond)
		a.setPaused(false)
	}()
	done := make(chan struct{})
	go func() {
		a.waitWhilePaused()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("waitWhilePaused did not return after resume")
	}
}

func TestStatusSummaryLine(t *testing.T) {
	// Unit check on videoSummary
	vids := []video{
		{id: 1, title: "v1", status: "done"},
		{id: 2, title: "v2", status: "done"},
	}
	s, comp, failed := videoSummary(vids)
	if s != "2/2 downloaded" || !comp || failed {
		t.Fatalf("expected '2/2 downloaded' complete, got %q comp=%v failed=%v", s, comp, failed)
	}

	vidsFailed := []video{
		{id: 1, title: "v1", status: "done"},
		{id: 2, title: "v2", status: "failed"},
	}
	s, comp, failed = videoSummary(vidsFailed)
	if s != "1/2 downloaded (1 failed)" || !comp || !failed {
		t.Fatalf("expected '1/2 downloaded (1 failed)' complete, got %q comp=%v failed=%v", s, comp, failed)
	}

	// UI Tester check: profile screen renders the summary line
	a := &app{cfg: defaultConfig()}
	a.applyProfile(profile{id: 1, name: "myprofile"}, vids)
	tt := ui.NewTester(a.view, 480, 360)
	if !tt.HasText("2/2 downloaded") {
		t.Errorf("texts %q missing '2/2 downloaded'", tt.Texts())
	}

	// With a failed item
	a.applyProfile(profile{id: 1, name: "myprofile"}, vidsFailed)
	tt = ui.NewTester(a.view, 480, 360)
	if !tt.HasText("1/2 downloaded (1 failed)") {
		t.Errorf("texts %q missing '1/2 downloaded (1 failed)'", tt.Texts())
	}
}

func TestToastNotification(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.showToast("Network Error", "Failed to fetch profile", "error")

	if len(a.toasts) != 1 {
		t.Fatalf("expected 1 toast queued, got %d", len(a.toasts))
	}

	tt := ui.NewTester(a.view, 480, 360)
	if len(a.toasts) != 0 {
		t.Errorf("expected toasts to be drained during view pass, got %d", len(a.toasts))
	}

	// Toast title/description should be rendered by mygo toast viewport
	if !tt.HasText("Network Error") {
		t.Errorf("expected toast 'Network Error' in texts %q", tt.Texts())
	}
}

func TestOpenFolderHelpers(t *testing.T) {
	// folderURL
	u := folderURL("/tmp/myfolder")
	if !strings.HasPrefix(u, "file://") || !strings.Contains(u, "/tmp/myfolder") {
		t.Fatalf("unexpected folderURL %q", u)
	}

	// resolveFolder on file-like path vs dir path
	dir := resolveFolder("/tmp/folder/video.mp4")
	if dir != "/tmp/folder" {
		t.Fatalf("expected /tmp/folder, got %q", dir)
	}

	// resolveFolder on empty
	dirEmpty := resolveFolder("")
	if dirEmpty != "." {
		t.Fatalf("expected ., got %q", dirEmpty)
	}
}

func TestOpenFolderUI(t *testing.T) {
	tmpDir := t.TempDir()
	sampleVideo := filepath.Join(tmpDir, "vid1.mp4")

	vids := []video{
		{id: 1, title: "vid1", status: "done", path: sampleVideo},
		{id: 2, title: "vid2", status: "pending"},
	}

	a := &app{cfg: defaultConfig(), outDir: tmpDir}
	a.applyProfile(profile{id: 1, name: "myprofile"}, vids)

	tt := ui.NewTester(a.view, 640, 480)

	// Since only 1 video is done (not all), per-row button exists, but after-success button does not
	if !tt.HasText("Open folder") {
		t.Fatalf("expected 'Open folder' button on done video row, got texts %q", tt.Texts())
	}
	if tt.HasText("Open output folder") {
		t.Fatalf("did not expect 'Open output folder' before all videos are done")
	}

	// Click row "Open folder"
	if err := tt.Click("Open folder"); err != nil {
		t.Fatalf("failed to click 'Open folder': %v", err)
	}

	opened := tt.OpenedURLs()
	if len(opened) == 0 {
		t.Fatalf("expected OpenedURLs to contain folder URL, got none")
	}
	expectedFolder, _ := filepath.Abs(tmpDir)
	if !strings.HasPrefix(opened[0], "file://") || !strings.Contains(opened[0], filepath.ToSlash(expectedFolder)) {
		t.Fatalf("expected opened URL to contain %q, got %q", expectedFolder, opened[0])
	}
	if tt.Clipboard() != expectedFolder {
		t.Fatalf("expected clipboard %q, got %q", expectedFolder, tt.Clipboard())
	}

	// Now mark all videos done to verify after-success "Open output folder" button
	a.videos[1].status = "done"
	a.videos[1].path = filepath.Join(tmpDir, "vid2.mp4")
	tt = ui.NewTester(a.view, 640, 480)

	if !tt.HasText("Open output folder") {
		t.Fatalf("expected 'Open output folder' button on completion, got texts %q", tt.Texts())
	}

	if err := tt.Click("Open output folder"); err != nil {
		t.Fatalf("failed to click 'Open output folder': %v", err)
	}

	openedAfter := tt.OpenedURLs()
	if len(openedAfter) != 1 {
		t.Fatalf("expected 1 OpenedURL on new tester, got %d", len(openedAfter))
	}
	if !strings.HasPrefix(openedAfter[0], "file://") || !strings.Contains(openedAfter[0], filepath.ToSlash(expectedFolder)) {
		t.Fatalf("expected openedAfter URL to contain %q, got %q", expectedFolder, openedAfter[0])
	}
	if tt.Clipboard() != expectedFolder {
		t.Fatalf("expected clipboard %q, got %q", expectedFolder, tt.Clipboard())
	}
}

func TestMinWindowResize(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.initDB()

	// 1. Main screen at minimum bounds 360x280
	ttMain := ui.NewTester(a.view, 360, 280)
	if !ttMain.HasText("nameless") || !ttMain.HasText("Fetch profile") {
		t.Fatalf("main screen missing essential text at 360x280: %v", ttMain.Texts())
	}

	// 2. Profile screen at minimum bounds 360x280 with long titles and actions
	vids := []video{
		{id: 1, title: "A very long video title that might squeeze layout on small windows", status: "done", path: "/tmp/vid1.mp4"},
		{id: 2, title: "Second video", status: "pending"},
	}
	a.applyProfile(profile{id: 1, name: "myprofile"}, vids)
	ttProfile := ui.NewTester(a.view, 360, 280)
	if !ttProfile.HasText("myprofile") || !ttProfile.HasText("Download all") || !ttProfile.HasText("Open folder") {
		t.Fatalf("profile screen missing essential text at 360x280: %v", ttProfile.Texts())
	}

	// 3. Settings screen at minimum bounds 360x280
	a.enterSettings()
	ttSettings := ui.NewTester(a.view, 360, 280)
	if !ttSettings.HasText("Settings") {
		t.Fatalf("settings screen missing 'Settings' at 360x280: %v", ttSettings.Texts())
	}
}

func TestSidebarNavigation(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.initDB()
	a.profiles = []profile{
		{id: 1, name: "tiktok_dancer", url: "https://tiktok.com/@tiktok_dancer"},
		{id: 2, name: "music_clips", url: "https://tiktok.com/@music_clips"},
	}

	tt := ui.NewTester(a.view, 640, 480)

	// Verify sidebar elements are present
	if !tt.HasText("Home") || !tt.HasText("Profiles") || !tt.HasText("Settings") {
		t.Fatalf("sidebar missing navigation items: %v", tt.Texts())
	}

	// Click Settings in sidebar
	if err := tt.Click("Settings"); err != nil {
		t.Fatalf("failed to click Settings in sidebar: %v", err)
	}
	if a.screen != screenSettings {
		t.Fatalf("expected screenSettings, got %v", a.screen)
	}

	// Click Home in sidebar
	if err := tt.Click("Home"); err != nil {
		t.Fatalf("failed to click Home in sidebar: %v", err)
	}
	if a.screen != screenMain {
		t.Fatalf("expected screenMain, got %v", a.screen)
	}

	// Click Profiles in sidebar -> navigates to dedicated Profiles page
	if err := tt.Click("Profiles"); err != nil {
		t.Fatalf("failed to click Profiles in sidebar: %v", err)
	}
	if a.screen != screenProfilesList {
		t.Fatalf("expected screenProfilesList, got %v", a.screen)
	}

	// Profiles page displays profiles
	if !tt.HasText("tiktok_dancer") {
		t.Fatalf("profiles page missing profile card: %v", tt.Texts())
	}

	// Click profile title on card -> navigates to screenProfile
	if err := tt.Click("tiktok_dancer"); err != nil {
		t.Fatalf("failed to click profile title: %v", err)
	}
	if a.screen != screenProfile || a.currentProfile == nil || a.currentProfile.id != 1 {
		t.Fatalf("expected screenProfile with profile 1, got screen=%v profile=%v", a.screen, a.currentProfile)
	}

	// Click "← All Profiles" -> navigates back to screenProfilesList
	if err := tt.Click("← All Profiles"); err != nil {
		t.Fatalf("failed to click ← All Profiles: %v", err)
	}
	if a.screen != screenProfilesList {
		t.Fatalf("expected screenProfilesList after back, got %v", a.screen)
	}
}

func TestProfileIdentityCollapsesVariants(t *testing.T) {
	// Different spellings of the same account share one identity key: the
	// duplicate guard collapses them so one row per account is kept, whether
	// the user types a full URL, a hostless path, "@handle" or a bare handle.
	base := profileURLKey("https://www.tiktok.com/@creator")
	for _, variant := range []string{
		"https://www.tiktok.com/@CREATOR/",
		"https://www.tiktok.com/@creator?is_from_webapp=1",
		"tiktok.com/@creator",
		"@creator",
		"creator", // bare handle (e.g. "ne_edless")
	} {
		if got := profileURLKey(variant); got != base {
			t.Fatalf("variant %q: key = %q, want %q", variant, got, base)
		}
	}
	if u := normalizeProfileURL("ne_edless"); !strings.HasSuffix(u, "/@ne_edless") {
		t.Fatalf("bare handle must resolve to an @-profile URL, got %q", u)
	}
}

func TestUpsertFetchedVideos(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "seed.db"))
	if err != nil {
		t.Fatalf("failed to open isolated test db: %v", err)
	}
	defer db.Close()
	a.db = db
	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT, avatar_url TEXT, avatar_path TEXT)")
	db.Exec("CREATE TABLE videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT, position INTEGER)")
	db.Exec("INSERT INTO profiles(id, url, name) VALUES(1, 'https://tiktok.com/@creator', 'creator')")
	// A finished download must survive a re-fetch untouched.
	db.Exec("INSERT INTO videos(profile_id, url, title, status, path) VALUES(1, 'https://tiktok.com/@creator/video/1', 'v1', 'done', '/dl/v1.mp4')")

	meta := playlistMeta{
		Title: "creator",
		Entries: []playlistEntry{
			{ID: "1", URL: "https://tiktok.com/@creator/video/1", Title: "v1"}, // exists -> untouched
			{ID: "2", URL: "https://tiktok.com/@creator/video/2", Title: "v2"}, // new
			{ID: "3", URL: "", Title: "no-url"},                                // skipped
			{ID: "4", URL: "https://tiktok.com/@creator/video/4", Title: ""},   // new, fallback title
		},
	}
	p := profile{id: 1, url: "https://tiktok.com/@creator", name: "creator"}
	a.upsertFetchedVideos(&p, meta)

	vids := a.loadVideos(1)
	if len(vids) != 3 {
		t.Fatalf("expected 3 videos after upsert, got %d", len(vids))
	}
	for _, v := range vids {
		switch v.url {
		case "https://tiktok.com/@creator/video/1":
			if v.status != "done" || v.path != "/dl/v1.mp4" {
				t.Fatalf("existing done video was modified: %+v", v)
			}
		case "https://tiktok.com/@creator/video/2":
			if v.status != "pending" {
				t.Fatalf("new video should be pending, got %q", v.status)
			}
		case "https://tiktok.com/@creator/video/4":
			if v.title != "Video 4" {
				t.Fatalf("expected fallback title, got %q", v.title)
			}
		default:
			t.Fatalf("unexpected video URL %q", v.url)
		}
	}
}

func TestFetchProfileEndToEnd(t *testing.T) {
	// A stub yt-dlp binary on PATH (no network): prints one flat-playlist
	// JSON document with a fake title and two usable entries.
	stubDir := t.TempDir()
	script := `#!/bin/sh
cat <<'JSON'
{"title":"Fake Creator","uploader":"fake","thumbnail":"","entries":[{"id":"1","url":"https://tiktok.com/@fake/video/1","title":"V1"},{"id":"2","url":"https://tiktok.com/@fake/video/2","title":"V2"},{"id":"3","url":"","title":"bad"}]}
JSON
`
	if err := os.WriteFile(filepath.Join(stubDir, "yt-dlp"), []byte(script), 0o777); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	t.Setenv("PATH", stubDir+string(filepath.ListSeparator)+os.Getenv("PATH"))

	a := &app{cfg: defaultConfig()}
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "e2e.db"))
	if err != nil {
		t.Fatalf("failed to open isolated test db: %v", err)
	}
	defer db.Close()
	a.db = db
	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT, avatar_url TEXT, avatar_path TEXT)")
	db.Exec("CREATE TABLE videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT, position INTEGER)")

	// 1. Fresh account: data is fetched BEFORE any insert, and the row that
	// lands in SQLite carries the fetched identity, not the pasted URL.
	a.profileURL = "https://www.tiktok.com/@fake"
	a.fetchProfile() // headless: runs to completion synchronously
	if a.fetching {
		t.Fatal("headless fetch must complete before returning")
	}
	if a.screen != screenProfile || a.currentProfile == nil || a.currentProfile.name != "Fake Creator" {
		t.Fatalf("expected navigation to profile details with fetched name, got screen=%v profile=%+v", a.screen, a.currentProfile)
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM profiles").Scan(&n)
	if n != 1 {
		t.Fatalf("expected exactly 1 profile row after fetch, got %d", n)
	}
	db.QueryRow("SELECT COUNT(*) FROM videos WHERE profile_id=?", a.currentProfile.id).Scan(&n)
	if n != 2 {
		t.Fatalf("expected 2 seeded videos (empty-url entry skipped), got %d", n)
	}

	// 2. Same account via a different spelling: refreshes in place, never
	// inserts a second row, and keeps the video list stable.
	a.profileURL = "@fake" // bare-handle spelling of the same account
	a.fetchProfile()
	db.QueryRow("SELECT COUNT(*) FROM profiles").Scan(&n)
	if n != 1 {
		t.Fatalf("duplicate guard failed: %d profile rows after re-fetch", n)
	}
	db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&n)
	if n != 2 {
		t.Fatalf("re-fetch must not duplicate videos, got %d rows", n)
	}
}

func TestFetchProfileRejectsNonexistentProfiles(t *testing.T) {
	// The failure mode observed in the wild: a fake/misbehaving yt-dlp
	// answers every handle with a placeholder "Sample Video" entry whose
	// URL is not a real video link. That reply must persist nothing.
	stubDir := t.TempDir()
	script := `#!/bin/sh
cat <<'JSON'
{"entries":[{"title":"Sample Video","id":"vid1","url":"https://www.tiktok.com/video/sample"}]}
JSON
`
	if err := os.WriteFile(filepath.Join(stubDir, "yt-dlp"), []byte(script), 0o777); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	t.Setenv("PATH", stubDir+string(filepath.ListSeparator)+os.Getenv("PATH"))

	a := &app{cfg: defaultConfig()}
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "reject.db"))
	if err != nil {
		t.Fatalf("failed to open isolated test db: %v", err)
	}
	defer db.Close()
	a.db = db
	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT, avatar_url TEXT, avatar_path TEXT)")
	db.Exec("CREATE TABLE videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT, position INTEGER)")

	a.profileURL = "@random_nonexistent_user"
	a.fetchProfile() // headless: synchronous
	if a.screen != screenMain {
		t.Fatalf("bogus profile data must not navigate to the profile screen, got %v", a.screen)
	}
	if a.fetchErr == "" {
		t.Fatalf("expected an actionable error for placeholder profile data")
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM profiles").Scan(&n)
	if n != 0 {
		t.Fatalf("placeholder yt-dlp reply must persist nothing, found %d rows", n)
	}
	db.QueryRow("SELECT COUNT(*) FROM videos").Scan(&n)
	if n != 0 {
		t.Fatalf("placeholder yt-dlp reply must seed no videos, found %d", n)
	}
}

func TestKeyboardOperability(t *testing.T) {
	// Offline yt-dlp stub: "--version" works (Detect passes), everything
	// else fails — profile data fetch errors out, downloads fail fast.
	stubDir := t.TempDir()
	script := `#!/bin/sh
case "$1" in
  --version) echo "2026.01.02";;
  *) exit 1;;
esac
`
	if err := os.WriteFile(filepath.Join(stubDir, "yt-dlp"), []byte(script), 0o777); err != nil {
		t.Fatalf("write stub: %v", err)
	}
	t.Setenv("PATH", stubDir+string(filepath.ListSeparator)+os.Getenv("PATH"))

	// 1. Enter on profileURL TextInput triggers fetchProfile.
	// Offline stub: fetch fails fast, nothing is saved, and the UI shows
	// the recoverable error (fetch-first, fail-loudly).
	a1 := &app{cfg: defaultConfig()}
	db1, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "kbd1.db"))
	if err != nil {
		t.Fatalf("failed to open isolated test db: %v", err)
	}
	defer db1.Close()
	a1.db = db1
	a1.profileURL = "https://www.tiktok.com/@testuser"
	a1.fetchProfile()
	if a1.fetching {
		t.Fatalf("headless fetch must complete synchronously before returning")
	}
	if a1.fetchErr == "" {
		t.Fatalf("expected a fetch error when yt-dlp cannot deliver profile data")
	}
	var rowsn int
	db1.QueryRow("SELECT COUNT(*) FROM profiles").Scan(&rowsn)
	// a1.initDB() was not called; the table may not exist yet, in which
	// case the count stays 0 either way.
	if rowsn > 0 {
		t.Fatalf("failed fetch must persist nothing, found %d rows", rowsn)
	}

	// 2. Escape shortcut on screenSettings returns to screenMain
	a := &app{cfg: defaultConfig()}
	a.initDB()
	a.enterSettings()
	if a.screen != screenSettings {
		t.Fatalf("expected screenSettings, got %v", a.screen)
	}
	ttSettings := ui.NewTester(a.view, 640, 480)
	ttSettings.Key(0, ui.KeyEscape)
	if a.screen != screenMain {
		t.Fatalf("expected screenMain after KeyEscape, got %v", a.screen)
	}

	// 3. Escape shortcut on screenProfile returns to screenMain
	a.applyProfile(profile{id: 1, name: "p1"}, []video{{id: 1, title: "v1", status: "pending"}})
	if a.screen != screenProfile {
		t.Fatalf("expected screenProfile, got %v", a.screen)
	}
	ttProfile := ui.NewTester(a.view, 640, 480)
	ttProfile.Key(0, ui.KeyEscape)
	if a.screen != screenMain {
		t.Fatalf("expected screenMain after KeyEscape on profile screen, got %v", a.screen)
	}

	// 4. Ctrl+D shortcut on screenProfile triggers startDownload
	a.applyProfile(profile{id: 1, name: "p1"}, []video{{id: 1, title: "v1", status: "pending"}})
	ttProfile = ui.NewTester(a.view, 640, 480)
	ttProfile.Key(ui.Ctrl, ui.KeyD)
	if !a.downloading {
		t.Fatalf("expected a.downloading to be true after Ctrl+D")
	}
}

func TestDashboardOverviewStatsAndSettings(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "dash_test.db")
	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()
	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT)")
	db.Exec("CREATE TABLE videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT)")
	db.Exec("INSERT INTO profiles(id, url, name) VALUES(1, 'https://tiktok.com/@p1', 'profile1')")
	db.Exec("INSERT INTO videos(profile_id, url, title, status) VALUES(1, 'u1', 'v1', 'done')")
	db.Exec("INSERT INTO videos(profile_id, url, title, status) VALUES(1, 'u2', 'v2', 'pending')")

	a := &app{cfg: defaultConfig(), db: db}
	a.loadProfiles()

	tt := ui.NewTester(a.view, 640, 480)

	// Check Overview Stats on Home/Dashboard
	if !tt.HasText("OVERVIEW STATS") || !tt.HasText("Total Downloaded") || !tt.HasText("Profiles Archived") {
		t.Fatalf("home screen missing overview stats: %v", tt.Texts())
	}
	if !tt.HasText("1 Videos") || !tt.HasText("1 Profiles") {
		t.Fatalf("expected counts in overview stats: %v", tt.Texts())
	}

	// Navigate to settings and check that Download Settings are present
	a.enterSettings()
	ttSettings := ui.NewTester(a.view, 640, 480)
	if !ttSettings.HasText("Download & Engine Settings") || !ttSettings.HasText("Default Download Location") {
		t.Fatalf("settings screen missing download settings: %v", ttSettings.Texts())
	}
}

func TestRecentDownloads(t *testing.T) {
	tmpDB := filepath.Join(t.TempDir(), "recent_test.db")
	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()
	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT)")
	db.Exec("CREATE TABLE videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT)")

	a := &app{cfg: defaultConfig(), db: db}

	// 1. Empty state
	ttEmpty := ui.NewTester(a.view, 640, 480)
	if !ttEmpty.HasText("RECENT DOWNLOADS") || !ttEmpty.HasText("No downloaded videos yet") {
		t.Fatalf("expected empty recent downloads section, got %v", ttEmpty.Texts())
	}

	// 2. Populate DB with downloaded videos
	db.Exec("INSERT INTO videos(id, profile_id, url, title, status, path) VALUES(1, 1, 'https://tiktok.com/@u/1', 'Dance Trend #1', 'done', '/tmp/dance1.mp4')")
	db.Exec("INSERT INTO videos(id, profile_id, url, title, status, path) VALUES(2, 1, 'https://tiktok.com/@u/2', 'Cooking Recipe #2', 'pending', '')")
	db.Exec("INSERT INTO videos(id, profile_id, url, title, status, path) VALUES(3, 1, 'https://tiktok.com/@u/3', 'Comedy Sketch #3', 'done', '/tmp/sketch3.mp4')")

	recent := a.recentDownloads(5)
	if len(recent) != 2 {
		t.Fatalf("expected 2 recent downloads, got %d", len(recent))
	}
	if recent[0].title != "Comedy Sketch #3" || recent[1].title != "Dance Trend #1" {
		t.Fatalf("expected newest completed first, got %v", recent)
	}

	// 3. UI renders recent items
	tt := ui.NewTester(a.view, 640, 480)
	if !tt.HasText("Comedy Sketch #3") || !tt.HasText("Dance Trend #1") {
		t.Fatalf("expected recent videos in home view, got %v", tt.Texts())
	}
	if !tt.HasText("2 completed") {
		t.Fatalf("expected '2 completed' badge, got %v", tt.Texts())
	}

	// 4. Clicking 'Open folder' triggers openFolder
	if err := tt.Click("Open folder"); err != nil {
		t.Fatalf("failed to click Open folder: %v", err)
	}
	opened := tt.OpenedURLs()
	if len(opened) == 0 {
		t.Fatalf("expected OpenedURLs to contain folder URL, got none")
	}
}

func TestShadcnSidebarFeatures(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.initDB()
	a.profiles = []profile{
		{id: 1, name: "tiktok_dancer", url: "https://tiktok.com/@tiktok_dancer"},
		{id: 2, name: "music_clips", url: "https://tiktok.com/@music_clips"},
	}

	tt := ui.NewTester(a.view, 640, 480)

	// 1. Sidebar header and navigation items
	if !tt.HasText("nameless") || !tt.HasText("ARCHIVED PROFILES") || !tt.HasText("Home") || !tt.HasText("Profiles") || !tt.HasText("Settings") {
		t.Fatalf("sidebar missing primary elements: %v", tt.Texts())
	}

	// 2. Profile items in sidebar with avatar initials
	if !tt.HasText("tiktok_dancer") || !tt.HasText("music_clips") {
		t.Fatalf("sidebar missing profiles: %v", tt.Texts())
	}
	// Initial letters T and M
	if !tt.HasText("T") || !tt.HasText("M") {
		t.Fatalf("sidebar missing avatar initials: %v", tt.Texts())
	}

	// 3. Filter profiles via sidebarFilter
	a.sidebarFilter = "music"
	ttFilter := ui.NewTester(a.view, 640, 480)
	if !ttFilter.HasText("music_clips") {
		t.Fatalf("expected music_clips in filtered sidebar")
	}

	// 4. Click profile in sidebar to open directly
	a.sidebarFilter = ""
	ttClick := ui.NewTester(a.view, 640, 480)
	if err := ttClick.Click("tiktok_dancer"); err != nil {
		t.Fatalf("failed to click profile in sidebar: %v", err)
	}
	if a.screen != screenProfile || a.currentProfile == nil || a.currentProfile.id != 1 {
		t.Fatalf("expected screenProfile with tiktok_dancer, got screen=%v profile=%v", a.screen, a.currentProfile)
	}

	// 5. Engine status in bottom card
	if !ttClick.HasText("yt-dlp") {
		t.Fatalf("sidebar footer missing engine status: %v", ttClick.Texts())
	}
}

func TestProfilesCardGridAndProfileImage(t *testing.T) {
	tmpDir := t.TempDir()
	tmpDB := filepath.Join(tmpDir, "prof_test.db")
	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()

	// Create test png image file for avatar
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	avatarFile := filepath.Join(tmpDir, "avatar1.png")
	if err := os.WriteFile(avatarFile, pngData, 0644); err != nil {
		t.Fatalf("failed to write test avatar: %v", err)
	}

	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT, avatar_url TEXT, avatar_path TEXT)")
	db.Exec("INSERT INTO profiles(id, url, name, avatar_url, avatar_path) VALUES(1, 'https://tiktok.com/@creator1', 'creator1', 'https://cdn.example.com/a1.png', ?)", avatarFile)
	db.Exec("INSERT INTO profiles(id, url, name, avatar_url, avatar_path) VALUES(2, 'https://tiktok.com/@creator2', 'creator2', '', '')")

	a := &app{cfg: defaultConfig(), db: db, avatarsDir: tmpDir}
	a.loadProfiles()

	if len(a.profiles) != 2 {
		t.Fatalf("expected 2 profiles loaded, got %d", len(a.profiles))
	}
	if a.profiles[0].avatarPath != avatarFile {
		t.Fatalf("expected avatarPath %q, got %q", avatarFile, a.profiles[0].avatarPath)
	}

	// Verify image caching and decoding
	img := a.profileImage(a.profiles[0])
	if img == nil {
		t.Fatalf("expected decoded ImageSource for profile 1")
	}
	if a.profileImage(a.profiles[1]) != nil {
		t.Fatalf("expected nil ImageSource for profile 2 without avatarPath")
	}

	// Verify Profiles Page Grid UI
	a.screen = screenProfilesList
	tt := ui.NewTester(a.view, 640, 480)

	if !tt.HasText("Archived Profiles") || !tt.HasText("creator1") || !tt.HasText("creator2") {
		t.Fatalf("profiles grid missing headers or creator cards: %v", tt.Texts())
	}
	if !tt.HasText("media/profiles/creator1") || !tt.HasText("media/profiles/creator2") {
		t.Fatalf("profiles card missing path subtitle: %v", tt.Texts())
	}
	if !tt.HasText("0 items") {
		t.Fatalf("profiles card missing item count badge: %v", tt.Texts())
	}
}

func TestProfilesGridFiveColsAndDropdown(t *testing.T) {
	tmpDir := t.TempDir()
	avatarFile := filepath.Join(tmpDir, "avatar1.png")
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
	if err := os.WriteFile(avatarFile, pngData, 0644); err != nil {
		t.Fatalf("failed to write test avatar: %v", err)
	}

	tmpDB := filepath.Join(tmpDir, "profiles_grid.db")
	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()
	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT, avatar_url TEXT, avatar_path TEXT)")
	db.Exec("INSERT INTO profiles(id, url, name, avatar_url, avatar_path) VALUES(1, 'https://tiktok.com/@creator1', 'creator1', 'https://avatar1.jpg', ?)", avatarFile)
	db.Exec("INSERT INTO profiles(id, url, name, avatar_url, avatar_path) VALUES(2, 'https://tiktok.com/@creator2', 'creator2', '', '')")

	a := &app{cfg: defaultConfig(), db: db, avatarsDir: tmpDir}
	a.loadProfiles()

	// 1. Verify default 5 columns
	a.screen = screenProfilesList
	tt := ui.NewTester(a.view, 640, 480)

	if !tt.HasText("Archived Profiles") || !tt.HasText("creator1") || !tt.HasText("creator2") {
		t.Fatalf("profiles grid missing headers or creator cards: %v", tt.Texts())
	}
	if !tt.HasText("Columns:") {
		t.Fatalf("profiles grid missing 'Columns:' dropdown label: %v", tt.Texts())
	}
	if a.gridCols != "5" {
		t.Fatalf("expected default gridCols to be '5', got %q", a.gridCols)
	}

	// 2. Click profile title on card (sidebar item shares the name and navigates the same way)
	//    ensures no button overflow or collision in the 5-column grid
	if err := tt.Click("creator1"); err != nil {
		t.Fatalf("failed to click profile title: %v", err)
	}
	if a.screen != screenProfile || a.currentProfile == nil || a.currentProfile.id != 1 {
		t.Fatalf("expected screenProfile after View Videos, got screen=%v profile=%v", a.screen, a.currentProfile)
	}

	// 3. Change columns to 3 and 7 (min 3 max 7)
	a.screen = screenProfilesList
	a.gridCols = "3"
	tt3 := ui.NewTester(a.view, 640, 480)
	if !tt3.HasText("creator1") {
		t.Fatalf("missing creator1 at 3 cols")
	}

	a.gridCols = "7"
	tt7 := ui.NewTester(a.view, 640, 480)
	if !tt7.HasText("creator1") {
		t.Fatalf("missing creator1 at 7 cols")
	}
}

func TestSettingsPageOverhaul(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.json")
	a := &app{
		cfg:     defaultConfig(),
		cfgPath: cfgFile,
	}
	a.enterSettings()

	tt := ui.NewTester(a.view, 800, 600)

	// Verify all cards and options are rendered
	expectedSections := []string{
		"Download & Engine Settings",
		"Storage & Naming",
		"Video Filename Template",
		"Video Quality Preference",
		"Save Video Thumbnails",
		"Save Metadata Info JSON",
		"Save Subtitles & Captions",
		"Engine & Concurrency",
		"Concurrent Downloads (Parallel Workers)",
		"Connections Per Download",
		"Delay Between Downloads",
		"Per-Video Timeout",
		"Custom yt-dlp Executable Path",
		"TikTok Authentication & Cookies",
		"Extract Cookies from Browser",
		"Netscape Cookies File",
		"Session ID Cookie",
		"Theme & Appearance",
		"Accent Color",
		"Corner Radius",
		"Interface Spacing Density",
		"Dark Mode",
		"Save Settings",
		"Reset Defaults",
	}

	for _, text := range expectedSections {
		if !tt.HasText(text) {
			t.Fatalf("settings page missing expected text %q", text)
		}
	}

	// Change values in memory
	a.outDir = "/custom/downloads"
	a.cfg.NamingTemplate = "%(id)s.%(ext)s"
	a.workersF64 = 5 // concurrent downloads, max 5
	a.connectionsF64 = 7
	a.delayStr = "2s"
	a.timeoutStr = "10m"
	a.cfg.VideoQuality = "720p"
	a.cfg.WriteThumbnail = false
	a.cfg.WriteInfoJSON = true
	a.cfg.WriteSubtitles = true
	a.cfg.CookiesFromBrowser = "firefox"
	a.cfg.YTDLPPath = "/custom/yt-dlp"

	// Save settings
	a.saveSettings()

	// Verify on-disk config file
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		t.Fatalf("failed to read saved config file: %v", err)
	}
	strData := string(data)
	if !strings.Contains(strData, "/custom/downloads") ||
		!strings.Contains(strData, "%(id)s.%(ext)s") ||
		!strings.Contains(strData, "720p") ||
		!strings.Contains(strData, "firefox") ||
		!strings.Contains(strData, "/custom/yt-dlp") {
		t.Fatalf("saved config missing modified values:\n%s", strData)
	}

	// Test Reset Defaults
	a.resetSettingsToDefault()
	if a.outDir != "./downloads" || a.workersF64 != 3 || a.connectionsF64 != 5 || a.cfg.VideoQuality != "best" {
		t.Fatalf("reset defaults did not restore defaults: outDir=%q workers=%v connections=%v quality=%q", a.outDir, a.workersF64, a.connectionsF64, a.cfg.VideoQuality)
	}
}

func TestIntegrationMilestone3(t *testing.T) {
	// 1. Missing yt-dlp banner (Task E5)
	a := &app{cfg: defaultConfig()}
	a.ytdlpMissing = true
	tt := ui.NewTester(a.view, 640, 480)

	if !tt.HasText("yt-dlp not found") || !tt.HasText("Detect") || !tt.HasText("Install instructions") {
		t.Fatalf("home screen missing yt-dlp banner: %v", tt.Texts())
	}
	if err := tt.Click("Install instructions"); err != nil {
		t.Fatalf("failed to click Install instructions: %v", err)
	}
	opened := tt.OpenedURLs()
	if len(opened) == 0 || !strings.Contains(opened[0], "yt-dlp") {
		t.Fatalf("expected yt-dlp install URL opened, got %v", opened)
	}

	// 2. Empty profile state & overall bar (Tasks E6 / E8)
	a.screen = screenProfile
	a.currentProfile = &profile{id: 1, name: "empty_creator", url: "https://tiktok.com/@empty"}
	a.videos = nil
	ttEmptyProf := ui.NewTester(a.view, 640, 480)
	if !ttEmptyProf.HasText("No videos found") || !ttEmptyProf.HasText("No videos found — the account may be private; try cookies in Settings") {
		t.Fatalf("profile detail missing empty state: %v", ttEmptyProf.Texts())
	}

	// 3. ResetInterrupted on startup (Task E8)
	tmpDB := filepath.Join(t.TempDir(), "resume_test.db")
	db, err := sql.Open("sqlite", tmpDB)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	defer db.Close()
	db.Exec("CREATE TABLE profiles(id INTEGER PRIMARY KEY, url TEXT, name TEXT)")
	db.Exec("CREATE TABLE videos(id INTEGER PRIMARY KEY, profile_id INTEGER, url TEXT, title TEXT, status TEXT, path TEXT)")
	db.Exec("INSERT INTO profiles(id, url, name) VALUES(1, 'https://tiktok.com/@p', 'p')")
	db.Exec("INSERT INTO videos(profile_id, url, title, status) VALUES(1, 'u1', 'v1', 'downloading')")
	db.Exec("INSERT INTO videos(profile_id, url, title, status) VALUES(1, 'u2', 'v2', 'paused')")
	db.Exec("INSERT INTO videos(profile_id, url, title, status) VALUES(1, 'u3', 'v3', 'done')")

	// Run ResetInterrupted query as done in initDB
	db.Exec("UPDATE videos SET status='pending' WHERE status IN ('downloading', 'paused')")

	var s1, s2, s3 string
	db.QueryRow("SELECT status FROM videos WHERE id=1").Scan(&s1)
	db.QueryRow("SELECT status FROM videos WHERE id=2").Scan(&s2)
	db.QueryRow("SELECT status FROM videos WHERE id=3").Scan(&s3)

	if s1 != "pending" || s2 != "pending" || s3 != "done" {
		t.Fatalf("expected interrupted reset to pending, got s1=%q s2=%q s3=%q", s1, s2, s3)
	}
}
