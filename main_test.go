package main

import (
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

	tt := ui.NewTester(a.view, 480, 360)

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
	tt = ui.NewTester(a.view, 480, 360)

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
