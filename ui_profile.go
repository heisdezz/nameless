package main

import (
	"fmt"
	"path/filepath"

	"nameless/ui/pages"

	"github.com/egoist/mygo/ui"
)

// viewProfilesList renders the dedicated Profiles overview page using pages.RenderProfiles.
func (a *app) viewProfilesList(c *ui.Context) {
	if a.gridCols == "" {
		a.gridCols = "5"
	}

	// Counts come from the one-query cache (refreshProfileCounts), not a
	// DB query per card per frame. Avatars come from the decode cache.
	a.mu.Lock()
	counts := a.profileCounts
	avatars := a.avatarCache
	a.mu.Unlock()
	items := make([]pages.ProfileItem, len(a.profiles))
	for i, p := range a.profiles {
		items[i] = pages.ProfileItem{
			ID:          p.id,
			Name:        p.name,
			URL:         p.url,
			AvatarPath:  p.avatarPath,
			AvatarURL:   p.avatarURL,
			Avatar:      avatars[p.id],
			VideosCount: counts[p.id],
		}
	}

	pages.RenderProfiles(c, pages.ProfilesProps{
		Profiles: items,
		GridCols: &a.gridCols,
		Loading:  a.profilesLoading,
		Page:     a.profilesPage,
		PageSize: a.profilesPageSize,
		OnPageChange: func(page int) {
			a.profilesPage = page
		},
		OnPageSizeChange: func(size int) {
			a.profilesPageSize = size
			a.profilesPage = 1
		},
		OnNew: func() {
			a.stopPlayer()
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
		OnDelete: func(p pages.ProfileItem) {
			a.deleteProfile(p.ID)
		},
	})
}

// viewProfile renders the detailed video list and batch download surface for the active profile.
func (a *app) viewProfile(c *ui.Context) {
	t := c.Theme()

	if a.profileGridCols == "" {
		a.profileGridCols = "4"
	}

	// Keyboard shortcut: Ctrl+D to start/download all
	c.OnWindowShortcut(ui.Ctrl, ui.KeyD, func() {
		a.startDownload()
	})

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

	// One lock for the whole snapshot (was one per video per frame);
	// thumbnails are kicked from applyProfile/openProfile, not here —
	// spawning ffmpeg jobs from the per-frame view caused the lag.
	a.mu.Lock()
	thumbCache := a.thumbCache
	videoDurations := a.videoDurations
	a.mu.Unlock()
	videoItems := make([]pages.VideoItem, len(a.videos))
	for i, v := range a.videos {
		videoItems[i] = pages.VideoItem{
			ID:        v.id,
			URL:       v.url,
			Title:     v.title,
			Status:    v.status,
			Path:      v.path,
			Pct:       v.pct,
			Duration:  videoDurations[v.id],
			Thumbnail: thumbCache[v.id],
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
		OverallPct:     float64(len(a.videos)),
		GridCols:       &a.profileGridCols,
		Page:           a.profilePage,
		PageSize:       a.profilePageSize,
		SearchQuery:    &a.profileSearch,
		SortBy:         &a.profileSort,
		OnSortChange: func(sortBy string) {
			a.profileSort = sortBy
			a.profilePage = 1
		},
		EnableAnimations: true,
		Loading:          a.profileLoading,
		OnPageChange: func(page int) {
			a.profilePage = page
		},
		OnPageSizeChange: func(size int) {
			a.profilePageSize = size
			a.profilePage = 1
		},
		OnBack: func() {
			a.showProfilesList()
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
		OnWatchVideo: func(v pages.VideoItem) {
			a.openPlayer(v.ID)
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
	if failed > 0 {
		return fmt.Sprintf("%d/%d downloaded (%d failed)", done, total, failed), false, true
	}
	return fmt.Sprintf("%d/%d downloaded", done, total), false, false
}
