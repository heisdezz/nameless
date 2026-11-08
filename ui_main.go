package main

import (
	"nameless/ui/pages"

	"github.com/egoist/mygo/ui"
)

func (a *app) viewMain(c *ui.Context) {
	profilesCount, downloadedCount, _ := a.overviewStats()
	workers := 3
	if a.cfg != nil && a.cfg.Workers > 0 {
		workers = a.cfg.Workers
	}

	recents := a.recentDownloads(5)
	recentItems := make([]pages.RecentDownloadItem, len(recents))
	for i, v := range recents {
		recentItems[i] = pages.RecentDownloadItem{
			ID:    v.id,
			Title: v.title,
			Path:  v.path,
		}
	}

	pages.RenderHome(c, pages.HomeProps{
		ProfileURL:      &a.profileURL,
		Fetching:        a.fetching,
		FetchErr:        a.fetchErr,
		DownloadedCount: downloadedCount,
		ProfilesCount:   profilesCount,
		Workers:         workers,
		EngineStatus:    a.ytdlpVersion,
		YTDLPMissing:    a.ytdlpMissing,
		RecentDownloads: recentItems,
		OnFetch: func() {
			a.fetchProfile()
		},
		OnDetect: func() {
			a.detectYTDLP()
		},
		OnInstallDocs: func() {
			a.openInstallDocs(c)
		},
		OnOpenFolder: func(path string) {
			a.openFolder(c, path)
		},
	})
}
