package main

import (
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
