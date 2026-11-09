package components

import (
	"testing"
	"time"

	"nameless/ui"

	mygoui "github.com/egoist/mygo/ui"
)

func TestRenderProfileCard(t *testing.T) {
	rendered := false
	view := func(c *mygoui.Context) {
		RenderProfileCard(c, ProfileCardProps{
			ID:          1,
			Name:        "Test Creator",
			URL:         "https://tiktok.com/@testcreator",
			VideosCount: 12,
			Date:        time.Now().Add(-2 * time.Hour),
			OnViewVideos: func() {
				rendered = true
			},
		})
	}

	tt := mygoui.NewTester(view, 400, 300)
	if !tt.HasText("Test Creator") {
		t.Errorf("expected 'Test Creator' on profile card, got %v", tt.Texts())
	}
	if !tt.HasText("12 items") {
		t.Errorf("expected '12 items' on profile card, got %v", tt.Texts())
	}
	if !tt.HasText("2 hours ago") {
		t.Errorf("expected '2 hours ago' on profile card, got %v", tt.Texts())
	}

	_ = tt.Click("Test Creator")
	if !rendered {
		t.Errorf("expected OnViewVideos to be invoked on title click")
	}
}

func TestRenderVideoCard(t *testing.T) {
	watched := false
	folderOpened := false

	view := func(c *mygoui.Context) {
		RenderVideoCard(c, VideoCardProps{
			ID:       42,
			Index:    0,
			Title:    "Epic Beat",
			Status:   "done",
			Duration: "1:45",
			Path:     "/tmp/video_42.mp4",
			OnWatch: func() {
				watched = true
			},
			OnOpenFolder: func() {
				folderOpened = true
			},
		})
	}

	tt := mygoui.NewTester(view, 400, 300)
	if !tt.HasText("Epic Beat") {
		t.Errorf("expected 'Epic Beat' on video card, got %v", tt.Texts())
	}
	if !tt.HasText("#1") {
		t.Errorf("expected index '#1' on video card, got %v", tt.Texts())
	}
	if !tt.HasText("1:45") {
		t.Errorf("expected duration '1:45' on video card, got %v", tt.Texts())
	}
	if !tt.HasText("Downloaded") {
		t.Errorf("expected status 'Downloaded' on video card, got %v", tt.Texts())
	}

	_ = tt.Click("Epic Beat")
	if !watched {
		t.Errorf("expected OnWatch to be invoked on title click")
	}

	_ = tt.Click("Open folder")
	if !folderOpened {
		t.Errorf("expected OnOpenFolder to be invoked")
	}
}

func TestRenderStatCard(t *testing.T) {
	view := func(c *mygoui.Context) {
		RenderStatCard(c, StatCardProps{
			Title: "Downloaded",
			Value: "42 Videos",
			Icon:  ui.IconCheck,
		})
	}

	tt := mygoui.NewTester(view, 200, 150)
	if !tt.HasText("Downloaded") || !tt.HasText("42 Videos") {
		t.Errorf("expected stat card texts, got %v", tt.Texts())
	}
}

func TestRenderVideoCardDownloadButton(t *testing.T) {
	downloaded := false
	view := func(c *mygoui.Context) {
		RenderVideoCard(c, VideoCardProps{
			ID:      7,
			Index:   0,
			Title:   "Pending Vid",
			Status:  "pending",
			OnDownload: func() {
				downloaded = true
			},
		})
	}
	tt := mygoui.NewTester(view, 400, 300)
	if !tt.HasText("Download") {
		t.Errorf("expected 'Download' button for pending video, got %v", tt.Texts())
	}
	if tt.HasText("Open folder") {
		t.Errorf("pending video should not show 'Open folder', got %v", tt.Texts())
	}
	_ = tt.Click("Download")
	if !downloaded {
		t.Errorf("expected OnDownload to be invoked")
	}
}
