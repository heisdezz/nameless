package main

import (
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo/ui"
)

// viewProfilesList renders the dedicated Profiles overview page.
func (a *app) viewProfilesList(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Gap(16).Padding(24).Children(func() {
			// Page Header
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					ui.Text(c, "Archived Profiles").FontSize(20).Bold()
					ui.Text(c, fmt.Sprintf("%d creator profiles saved", len(a.profiles))).TextColor(t.TextMuted).FontSize(13)
				})
				if ui.PrimaryButton(c.Key("new-profile-btn"), "+ New Download").Radius(6).Clicked() {
					a.screen = screenMain
				}
			})

			// Empty State
			if len(a.profiles) == 0 {
				ui.Column(c).FillWidth().Padding(32).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Center().Children(func() {
					ui.Text(c, "No profiles archived yet").Bold().FontSize(15)
					ui.Text(c, "Paste a TikTok creator profile URL on the Home screen to fetch and download videos.").TextColor(t.TextMuted).FontSize(13)
					if ui.PrimaryButton(c.Key("empty-fetch-btn"), "Fetch your first profile").Radius(6).Clicked() {
						a.screen = screenMain
					}
				})
				return
			}

			// Profiles Cards Grid/List
			for i := range a.profiles {
				p := a.profiles[i]
				c.Key(fmt.Sprintf("prof-card-%d", p.id))
				ui.Row(c).FillWidth().Padding(14).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
					// Icon Placeholder
					ui.Column(c).Size(36, 36).Center().Radius(6).Background(t.SurfaceHover).Children(func() {
						ui.Icon(c, iconFolder).Size(18, 18).TextColor(t.Text)
					})

					// Profile Info
					ui.Column(c).Fill().Gap(2).Children(func() {
						ui.Text(c, p.name).Bold().FontSize(14)
						ui.Text(c, p.url).TextColor(t.TextMuted).FontSize(12)
					})

					// Actions
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if ui.Button(c.Key(fmt.Sprintf("open-folder-%d", p.id)), "Open folder").Radius(6).Clicked() {
							folder := filepath.Join(a.effectiveOutputDir(), p.name)
							a.openFolder(c, folder)
						}
						if ui.PrimaryButton(c.Key(fmt.Sprintf("view-vids-%d", p.id)), "View Videos").Radius(6).Clicked() {
							a.openProfile(p)
						}
					})
				})
			}
		})
	})
}

// viewProfile renders the detailed video list and batch download surface for the active profile.
func (a *app) viewProfile(c *ui.Context) {
	t := c.Theme()

	// Keyboard shortcut: Ctrl+D to start/download all
	if c.Shortcut(ui.Ctrl, ui.KeyD) {
		a.startDownload()
	}

	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Gap(14).Padding(24).Children(func() {
			// Navigation Breadcrumb / Back button to Profiles page
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if ui.Button(c.Key("back-to-profiles"), "← All Profiles").Radius(6).Clicked() {
					a.screen = screenProfilesList
				}
			})

			// Profile Header Card
			if a.currentProfile != nil {
				ui.Column(c).FillWidth().Padding(16).Gap(10).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
						ui.Text(c, a.currentProfile.name).FontSize(20).Bold()
						if summary, complete, hasFailed := videoSummary(a.videos); summary != "" {
							color := t.TextMuted
							if complete {
								if hasFailed {
									color = t.Danger
								} else {
									color = t.Success
								}
							}
							ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
								ui.Text(c, summary).TextColor(color).Bold().FontSize(13)
								if complete && !hasFailed {
									if ui.Button(c, "Open output folder").Radius(6).Clicked() {
										a.openFolder(c, a.effectiveOutputDir())
									}
								}
							})
						}
					})
					ui.Text(c, a.currentProfile.url).TextColor(t.TextMuted).FontSize(12)

					// Action Toolbar
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						if ui.PrimaryButton(c, "Download all").Radius(6).Clicked() {
							a.startDownload()
						}
						if a.downloading {
							if a.paused {
								if ui.Button(c, "Resume").Radius(6).Clicked() {
									a.setPaused(false)
								}
							} else {
								if ui.Button(c, "Pause").Radius(6).Clicked() {
									a.setPaused(true)
								}
							}
						}
						if ui.Button(c, "Retry failed").Radius(6).Clicked() {
							a.retryFailed()
						}
					})
				})
			}

			// Videos Section Header
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Text(c, "Videos").Bold().FontSize(15)
				ui.Text(c, fmt.Sprintf("%d items", len(a.videos))).TextColor(t.TextMuted).FontSize(12)
			})

			if len(a.videos) == 0 {
				ui.Text(c, "No videos found — the account may be private; try cookies in Settings").TextColor(t.TextMuted).FontSize(13)
			}

			// Video Cards
			for i := range a.videos {
				v := a.videos[i]
				c.Key(v.id)
				ui.Row(c).FillWidth().Padding(10).Gap(10).Radius(6).Background(t.Surface).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Fill().Gap(2).Children(func() {
						ui.Text(c, v.title).FontSize(13).Bold()
						ui.Text(c, v.status).TextColor(t.TextMuted).FontSize(11)
					})
					if v.status == "done" {
						if ui.Button(c, "Open folder").Radius(6).Clicked() {
							target := v.path
							if target == "" {
								target = a.effectiveOutputDir()
							}
							a.openFolder(c, target)
						}
					}
				})
			}
		})
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
