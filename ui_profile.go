package main

import (
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo/ui"
	"nameless/ui/pages"
)

// viewProfilesList renders the dedicated Profiles overview page using pages.RenderProfiles.
func (a *app) viewProfilesList(c *ui.Context) {
	items := make([]pages.ProfileItem, len(a.profiles))
	for i, p := range a.profiles {
		items[i] = pages.ProfileItem{
			ID:   p.id,
			Name: p.name,
			URL:  p.url,
		}
	}

	pages.RenderProfiles(c, pages.ProfilesProps{
		Profiles: items,
		OnNew: func() {
			a.screen = screenMain
		},
		OnViewVideos: func(p pages.ProfileItem) {
			for _, prof := range a.profiles {
				if prof.id == p.ID {
					a.openProfile(prof)
					break
				}
			}
		},
		OnOpenFolder: func(p pages.ProfileItem) {
			folder := filepath.Join(a.effectiveOutputDir(), p.Name)
			a.openFolder(c, folder)
		},
	})
}

// viewProfile renders the detailed video list and batch download surface for the active profile.
func (a *app) viewProfile(c *ui.Context) {
	t := c.Theme()

	// Keyboard shortcut: Ctrl+D to start/download all
	if c.Shortcut(ui.Ctrl, ui.KeyD) {
		a.startDownload()
	}

	if a.currentProfile == nil {
		a.screen = screenProfilesList
		return
	}

	summary, complete, hasFailed := videoSummary(a.videos)
	color := t.TextMuted
	if complete {
		color = t.Success
	} else if hasFailed {
		color = t.Danger
	}

	videoItems := make([]pages.VideoItem, len(a.videos))
	for i, v := range a.videos {
		videoItems[i] = pages.VideoItem{
			ID:     v.id,
			URL:    v.url,
			Title:  v.title,
			Status: v.status,
			Path:   v.path,
		}
	}

	pages.RenderProfileDetail(c, pages.ProfileDetailProps{
		ProfileName:    a.currentProfile.name,
		ProfileURL:     a.currentProfile.url,
		Videos:         videoItems,
		SummaryText:    summary,
		SummaryColor:   color,
		ShowOpenOutput: complete && !hasFailed,
		Downloading:    a.downloading,
		Paused:         a.paused,
		OnBack: func() {
			a.screen = screenProfilesList
		},
		OnDownloadAll: func() {
			a.startDownload()
		},
		OnPause: func() {
			a.setPaused(true)
		},
		OnResume: func() {
			a.setPaused(false)
		},
		OnRetryFailed: func() {
			a.retryFailed()
		},
		OnOpenOutput: func() {
			a.openFolder(c, a.effectiveOutputDir())
		},
		OnOpenVideoDir: func(path string) {
			target := path
			if target == "" {
				target = a.effectiveOutputDir()
			}
			a.openFolder(c, target)
		},
	})
}

// videoSummary produces the status summary line (e.g. "42/42 downloaded")
// per brief and plan §4.
func videoSummary(videos []video) (summary string, complete bool, hasFailed bool) {
	total := len(videos)
	if total == 0 {
		return "", false, false
	}
	done := 0
	failed := 0
	for _, v := range videos {
		switch v.status {
		case "done":
			done++
		case "failed":
			failed++
		}
	}
	if done == total {
		return fmt.Sprintf("%d/%d downloaded", done, total), true, false
	}
	if done+failed == total && failed > 0 {
		return fmt.Sprintf("%d/%d downloaded (%d failed)", done, total, failed), true, true
	}
	if done > 0 || failed > 0 {
		return fmt.Sprintf("%d/%d downloaded", done, total), false, false
	}
	return "", false, false
}
