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
