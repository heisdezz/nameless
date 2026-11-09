package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"nameless/ui/pages"

	"github.com/egoist/mygo/ui"
	uivideo "github.com/egoist/mygo/ui/video"
)

// openPlayer switches to the player screen for the specified video ID.
func (a *app) openPlayer(videoID int64) {
	a.currentVideoID = videoID
	a.syncPlayer()
	a.shuffleRandomVideos()
	a.screen = screenPlayer
}

// syncPlayer ensures the in-app player matches the current video: it closes
// a stale player and opens a fresh one over the downloaded file. The player
// plays as soon as the Video widget shows it. UI thread only.
func (a *app) syncPlayer() {
	var cur video
	found := false
	for i := range a.videos {
		if a.videos[i].id == a.currentVideoID {
			cur = a.videos[i]
			found = true
			break
		}
	}
	// Keep a player that already matches both the video and its file.
	if a.player != nil && found && a.playerVideoID == a.currentVideoID && a.playerPath == cur.path {
		return
	}
	if a.player != nil {
		a.player.Close()
		a.player = nil
	}
	a.playerVideoID = a.currentVideoID
	a.playerPath = ""

	if !found || cur.path == "" {
		return // not downloaded yet: poster fallback
	}
	if _, err := os.Stat(cur.path); err != nil {
		return // file missing
	}
	p, err := uivideo.Open(cur.path)
	if err != nil {
		// System without a media framework (GStreamer missing on Linux),
		// or a broken file: fall back to the poster + external player.
		a.showToast("Cannot play in-app", err.Error(), "error")
		return
	}
	a.player = p
	a.playerPath = cur.path
}

// stopPlayer pauses playback (keeping position so returning to the same
// video resumes) and leaves fullscreen. It MUST run on every route out
// of the player screen — the media backend keeps playing audio even
// when the Video widget is no longer rendered.
func (a *app) stopPlayer() {
	if a.player != nil {
		a.player.Pause()
	}
	a.exitPlayerFullScreen()
}

// shuffleRandomVideos selects up to 8 random video IDs from the current profile excluding currentVideoID.
func (a *app) shuffleRandomVideos() {
	if len(a.videos) <= 1 {
		a.randomVideoIDs = nil
		return
	}
	var others []int64
	for _, v := range a.videos {
		if v.id != a.currentVideoID {
			others = append(others, v.id)
		}
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(others), func(i, j int) {
		others[i], others[j] = others[j], others[i]
	})
	if len(others) > 8 {
		others = others[:8]
	}
	a.randomVideoIDs = others
}

// viewPlayer renders the inner video player screen.
func (a *app) viewPlayer(c *ui.Context) {
	if a.currentProfile == nil || len(a.videos) == 0 {
		a.stopPlayer()
		a.screen = screenProfilesList
		return
	}

	// Space toggles play/pause
	c.OnWindowShortcut(0, ui.KeySpace, func() {
		a.togglePlayer()
	})

	// Find current video index
	var currentVideo video
	currIdx := 0
	found := false
	for i, v := range a.videos {
		if v.id == a.currentVideoID {
			currentVideo = v
			currIdx = i
			found = true
			break
		}
	}
	if !found && len(a.videos) > 0 {
		currentVideo = a.videos[0]
		a.currentVideoID = currentVideo.id
		currIdx = 0
	}

	// Next 2-3 videos in sequence from profile
	var nextVideos []pages.VideoItem
	total := len(a.videos)
	if total > 1 {
		count := 3
		if total-1 < count {
			count = total - 1
		}
		for k := 1; k <= count; k++ {
			nextIdx := (currIdx + k) % total
			nv := a.videos[nextIdx]
			var thumb *ui.Bitmap
			a.mu.Lock()
			if a.thumbCache != nil {
				thumb = a.thumbCache[nv.id]
			}
			dur := ""
			if a.videoDurations != nil {
				dur = a.videoDurations[nv.id]
			}
			a.mu.Unlock()
			nextVideos = append(nextVideos, pages.VideoItem{
				ID:        nv.id,
				URL:       nv.url,
				Title:     nv.title,
				Status:    nv.status,
				Path:      nv.path,
				Duration:  dur,
				Thumbnail: thumb,
			})
		}
	}

	// Random order of videos from profile
	var randomVideos []pages.VideoItem
	for _, rid := range a.randomVideoIDs {
		for _, v := range a.videos {
			if v.id == rid {
				var thumb *ui.Bitmap
				a.mu.Lock()
				if a.thumbCache != nil {
					thumb = a.thumbCache[v.id]
				}
				dur := ""
				if a.videoDurations != nil {
					dur = a.videoDurations[v.id]
				}
				a.mu.Unlock()
				randomVideos = append(randomVideos, pages.VideoItem{
					ID:        v.id,
					URL:       v.url,
					Title:     v.title,
					Status:    v.status,
					Path:      v.path,
					Duration:  dur,
					Thumbnail: thumb,
				})
				break
			}
		}
	}

	// Playback state derived straight from the real player, and a fresh
	// player whenever the current video has none yet (e.g. after the
	// download finished in the background since the screen opened).
	a.syncPlayer()
	a.mu.Lock()
	var currentThumb *ui.Bitmap
	if a.thumbCache != nil {
		currentThumb = a.thumbCache[currentVideo.id]
	}
	a.mu.Unlock()

	currTimeStr := "00:00"
	totTimeStr := "--:--"
	var progressPct float64
	playing := false
	if a.player != nil {
		playing = a.player.Playing()
		if d := a.player.Duration(); d > 0 {
			totTimeStr = formatClock(d)
		}
		pos := a.player.Position()
		currTimeStr = formatClock(pos)
		if d := a.player.Duration(); d > 0 {
			progressPct = float64(pos) / float64(d)
			a.playerSeek = pos.Seconds()
		}
	} else {
		a.mu.Lock()
		if a.videoDurations != nil && a.videoDurations[currentVideo.id] != "" {
			totTimeStr = a.videoDurations[currentVideo.id]
		}
		a.mu.Unlock()
	}

	pages.RenderPlayer(c, pages.PlayerProps{
		ProfileName: a.currentProfile.name,
		CurrentVideo: pages.VideoItem{
			ID:        currentVideo.id,
			URL:       currentVideo.url,
			Title:     currentVideo.title,
			Status:    currentVideo.status,
			Path:      currentVideo.path,
			Duration:  totTimeStr,
			Thumbnail: currentThumb,
		},
		NextVideos:   nextVideos,
		RandomVideos: randomVideos,
		Player:       a.player,
		Seek:         &a.playerSeek,
		OnSeek: func(seconds float64) {
			if a.player != nil {
				a.player.Seek(time.Duration(seconds * float64(time.Second)))
			}
		},
		IsPlaying:   playing,
		ProgressPct: progressPct,
		CurrentTime: currTimeStr,
		TotalTime:   totTimeStr,
		FullScreen:  a.playerFullScreen,
		OnBack: func() {
			a.stopPlayer()
			a.screen = screenProfile
		},
		OnPlayPause: func() {
			a.togglePlayer()
		},
		OnFullScreen: func() {
			a.togglePlayerFullScreen()
		},
		OnOpenPlayer: func() {
			a.playExternal(currentVideo.path)
		},
		OnOpenFolder: func(path string) {
			a.openFolder(c, path)
		},
		OnSelectVideo: func(v pages.VideoItem) {
			a.openPlayer(v.ID)
		},
		OnShuffle: func() {
			a.shuffleRandomVideos()
		},
	})
}

// formatClock renders a duration as M:SS.
func formatClock(d time.Duration) string {
	d = d.Truncate(time.Second)
	return fmt.Sprintf("%d:%02d", int(d/time.Minute), int(d/time.Second)%60)
}

// togglePlayer flips in-app playback, keeping the controls in sync with the
// next frame the player draws.
func (a *app) togglePlayer() {
	if a.player == nil {
		return
	}
	if a.player.Playing() {
		a.player.Pause()
	} else {
		a.player.Play()
	}
	if a.win != nil {
		a.win.Update(func() {})
	}
}

// togglePlayerFullScreen enters or leaves fullscreen video mode: the
// window goes fullscreen and the player view collapses to just the
// viewport with overlaid controls.
func (a *app) togglePlayerFullScreen() {
	if a.screen != screenPlayer {
		return
	}
	a.playerFullScreen = !a.playerFullScreen
	if a.win != nil {
		a.win.SetFullScreen(a.playerFullScreen)
		a.win.Update(func() {})
	}
}

// exitPlayerFullScreen leaves fullscreen video mode, restoring the
// window and the player page layout.
func (a *app) exitPlayerFullScreen() {
	if !a.playerFullScreen {
		return
	}
	a.playerFullScreen = false
	if a.win != nil {
		a.win.SetFullScreen(false)
		a.win.Update(func() {})
	}
}
