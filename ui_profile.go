package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *app) viewProfile(c *ui.Context) {
	t := c.Theme()

	// Keyboard shortcut: Ctrl+D to start/download all
	if c.Shortcut(ui.Ctrl, ui.KeyD) {
		a.startDownload()
	}

	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Gap(14).Padding(16).Children(func() {
			// Hero Profile Header Card
			if a.currentProfile != nil {
				ui.Column(c).FillWidth().Padding(20).Gap(12).Radius(20).Background(t.Surface).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
						ui.Text(c, a.currentProfile.name).FontSize(22).Bold()
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
									if ui.Button(c, "Open output folder").Radius(10).Clicked() {
										a.openFolder(c, a.effectiveOutputDir())
									}
								}
							})
						}
					})
					ui.Text(c, a.currentProfile.url).TextColor(t.TextMuted).FontSize(12)

					// Action Toolbar
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						if ui.PrimaryButton(c, "Download all").Radius(12).Clicked() {
							a.startDownload()
						}
						if a.downloading {
							if a.paused {
								if ui.Button(c, "Resume").Radius(12).Clicked() {
									a.setPaused(false)
								}
							} else {
								if ui.Button(c, "Pause").Radius(12).Clicked() {
									a.setPaused(true)
								}
							}
						}
						if ui.Button(c, "Retry failed").Radius(12).Clicked() {
							a.retryFailed()
						}
					})
				})
			}

			// Videos Section Header
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Text(c, "Videos").Bold().FontSize(16)
				ui.Text(c, fmt.Sprintf("%d items", len(a.videos))).TextColor(t.TextMuted).FontSize(12)
			})

			if len(a.videos) == 0 {
				ui.Text(c, "No videos found — the account may be private; try cookies in Settings").TextColor(t.TextMuted).FontSize(13)
			}

			// Video Cards
			for i := range a.videos {
				v := a.videos[i]
				c.Key(v.id)
				ui.Row(c).FillWidth().Padding(12).Gap(10).Radius(14).Background(t.Surface).AlignItems(ui.Center).Children(func() {
					ui.Column(c).Fill().Gap(2).Children(func() {
						ui.Text(c, v.title).FontSize(13).Bold()
						ui.Text(c, v.status).TextColor(t.TextMuted).FontSize(11)
					})
					if v.status == "done" {
						if ui.Button(c, "Open folder").Radius(8).Clicked() {
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
