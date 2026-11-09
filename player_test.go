package main

import (
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestProfileDetailsGridAndColumns(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.initDB()

	vids := []video{
		{id: 1, title: "Video One", status: "done", path: "/tmp/vid1.mp4"},
		{id: 2, title: "Video Two", status: "downloading", pct: 0.5},
		{id: 3, title: "Video Three", status: "failed"},
		{id: 4, title: "Video Four", status: "pending"},
		{id: 5, title: "Video Five", status: "done", path: "/tmp/vid5.mp4"},
	}
	a.applyProfile(profile{id: 1, name: "cool_creator"}, vids)

	tt := ui.NewTester(a.view, 800, 600)

	// Check grid header and column selector
	if !tt.HasText("Videos") || !tt.HasText("Columns:") {
		t.Fatalf("expected 'Videos' and 'Columns:' on profile details screen, got %v", tt.Texts())
	}

	// Default columns is 4
	if a.profileGridCols != "4" {
		t.Errorf("expected default profileGridCols to be '4', got %q", a.profileGridCols)
	}

	// Change columns to 3
	a.profileGridCols = "3"
	tt = ui.NewTester(a.view, 800, 600)
	if !tt.HasText("3") {
		t.Errorf("expected columns selector to reflect '3'")
	}

	// Change columns to 7 (maximum)
	a.profileGridCols = "7"
	tt = ui.NewTester(a.view, 800, 600)
	if !tt.HasText("7") {
		t.Errorf("expected columns selector to reflect '7'")
	}

	// Verify video cards render titles and actions (the explicit Watch
	// button was removed; cards open via cover/title clicks)
	if !tt.HasText("Video One") || !tt.HasText("Open folder") {
		t.Fatalf("expected video card texts, got %v", tt.Texts())
	}
}

func TestPlayerPageNavigationAndLayout(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.initDB()

	vids := []video{
		{id: 10, title: "First Track", status: "done", path: "/tmp/vid10.mp4"},
		{id: 11, title: "Second Track", status: "done", path: "/tmp/vid11.mp4"},
		{id: 12, title: "Third Track", status: "done", path: "/tmp/vid12.mp4"},
		{id: 13, title: "Fourth Track", status: "pending"},
		{id: 14, title: "Fifth Track", status: "pending"},
	}
	a.applyProfile(profile{id: 1, name: "music_maker"}, vids)

	tt := ui.NewTester(a.view, 1024, 768)

	// 1. Click a video card title to enter Player page (OnWatch fires from
	// the cover/title since the explicit Watch button was removed)
	if err := tt.Click("First Track"); err != nil {
		t.Fatalf("failed to click First Track card title: %v", err)
	}

	if a.screen != screenPlayer {
		t.Fatalf("expected screenPlayer after clicking Watch, got %v", a.screen)
	}
	if a.currentVideoID != 10 {
		t.Fatalf("expected currentVideoID 10, got %d", a.currentVideoID)
	}

	// 2. Verify Player Page elements
	// Header: "← Back to Profile", "music_maker", "Player", "Play in System Player"
	if !tt.HasText("← Back to Profile") || !tt.HasText("music_maker") || !tt.HasText("Play in System Player") {
		t.Fatalf("player page header missing essential elements: %v", tt.Texts())
	}

	// Main player viewport: First Track title
	if !tt.HasText("First Track") {
		t.Fatalf("player viewport missing video title 'First Track': %v", tt.Texts())
	}

	// Up Next section: "Up Next", and the upcoming videos (Track 2, 3, 4)
	if !tt.HasText("Up Next") || !tt.HasText("Second Track") {
		t.Fatalf("player page missing 'Up Next' section: %v", tt.Texts())
	}

	// Random section: "More From Profile (Random Order)", "Shuffle"
	if !tt.HasText("More From Profile (Random Order)") || !tt.HasText("Shuffle") {
		t.Fatalf("player page missing random order section: %v", tt.Texts())
	}

	// 3. Click "Second Track" in Up Next to switch player video
	if err := tt.Click("Second Track"); err != nil {
		t.Fatalf("failed to click Second Track in Up Next: %v", err)
	}
	if a.currentVideoID != 11 {
		t.Fatalf("expected currentVideoID 11 after selecting Second Track, got %d", a.currentVideoID)
	}

	// 4. Test Play/Pause toggle: no in-app player exists (the video files
	// are not on disk in this test), so toggling is a safe no-op.
	a.togglePlayer()
	if a.player != nil {
		t.Fatalf("expected no in-app player for missing files")
	}

	// 5. Click "← Back to Profile" to return to profile grid
	if err := tt.Click("← Back to Profile"); err != nil {
		t.Fatalf("failed to click Back to Profile: %v", err)
	}
	if a.screen != screenProfile {
		t.Fatalf("expected screenProfile after clicking Back to Profile, got %v", a.screen)
	}

	// 6. Test Escape shortcut from Player page returns to screenProfile
	a.openPlayer(12)
	if a.screen != screenPlayer {
		t.Fatalf("expected screenPlayer, got %v", a.screen)
	}
	tt = ui.NewTester(a.view, 1024, 768)
	tt.Key(0, ui.KeyEscape)
	if a.screen != screenProfile {
		t.Fatalf("expected screenProfile after Escape on player screen, got %v", a.screen)
	}
}

func TestPlayerFullScreenMode(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.initDB()

	vids := []video{
		{id: 20, title: "Fullscreen Flick", status: "done", path: ""},
	}
	a.applyProfile(profile{id: 2, name: "cinema"}, vids)
	a.openPlayer(20)

	// Fullscreen video mode: the viewport shows alone, no page chrome.
	a.playerFullScreen = true
	tt := ui.NewTester(a.view, 1024, 768)
	if a.screen != screenPlayer {
		t.Fatalf("expected screenPlayer in fullscreen mode, got %v", a.screen)
	}
	if tt.HasText("Up Next") || tt.HasText("More From Profile (Random Order)") || tt.HasText("← Back to Profile") {
		t.Fatalf("fullscreen video mode shows page chrome: %v", tt.Texts())
	}
	if !tt.HasText("Fullscreen Flick") {
		t.Fatalf("fullscreen video mode missing viewport content: %v", tt.Texts())
	}

	// Escape leaves fullscreen video mode but stays on the player.
	tt.Key(0, ui.KeyEscape)
	if a.playerFullScreen {
		t.Fatalf("expected playerFullScreen cleared after Escape from fullscreen")
	}
	if a.screen != screenPlayer {
		t.Fatalf("expected to stay on screenPlayer after Escape from fullscreen, got %v", a.screen)
	}

	// Second Escape returns to the profile screen.
	tt.Key(0, ui.KeyEscape)
	if a.screen != screenProfile {
		t.Fatalf("expected screenProfile after second Escape, got %v", a.screen)
	}
}

func TestPlayerExternalPlayback(t *testing.T) {
	a := &app{cfg: defaultConfig()}
	a.initDB()

	tmpFile := filepath.Join(t.TempDir(), "test_play.mp4")
	// Non-existent file toast
	a.playExternal(tmpFile)
	if len(a.toasts) == 0 || a.toasts[len(a.toasts)-1].toastType != "error" {
		t.Errorf("expected error toast for missing file")
	}

	// Empty path toast
	a.playExternal("")
	if len(a.toasts) == 0 || a.toasts[len(a.toasts)-1].toastType != "error" {
		t.Errorf("expected error toast for empty path")
	}
}
