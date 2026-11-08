package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *app) viewProfile(c *ui.Context) {
	t := c.Theme()

	// Keyboard shortcut: Ctrl+D to start/download all
	c.OnWindowShortcut(ui.Ctrl, ui.KeyD, func() {
		a.startDownload()
	})

	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Gap(14).Padding(16).Children(func() {
			// Hero Profile Header Card
			if a.currentProfile != nil {
				ui.Column(c).FillWidth().Padding(20).Gap(12).Radius(20).Background(t.Surface).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
						ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconFolder).Size(22, 22).TextColor(t.Accent)
							ui.Text(c, a.currentProfile.name).FontSize(22).Bold()
						})
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
									openOutBtn := ui.Button(c, "").Label("Open output folder").Radius(10)
									openOutBtn.Children(func() {
										ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
											ui.Icon(c, iconFolder).Size(14, 14)
											ui.Text(c, "Open output folder")
										})
									})
									if openOutBtn.Clicked() {
										a.openFolder(c, a.effectiveOutputDir())
									}
								}
							})
						}
					})
					ui.Text(c, a.currentProfile.url).TextColor(t.TextMuted).FontSize(12)

					// Action Toolbar
					ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
						dlBtn := ui.PrimaryButton(c, "").Label("Download all").Radius(12)
						dlBtn.Children(func() {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, iconPlay).Size(14, 14)
								ui.Text(c, "Download all")
							})
						})
						if dlBtn.Clicked() {
							a.startDownload()
						}

						if a.downloading {
							if a.paused {
								resBtn := ui.Button(c, "").Label("Resume").Radius(12)
								resBtn.Children(func() {
									ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
										ui.Icon(c, iconPlay).Size(14, 14)
										ui.Text(c, "Resume")
									})
								})
								if resBtn.Clicked() {
									a.setPaused(false)
								}
							} else {
								pauseBtn := ui.Button(c, "").Label("Pause").Radius(12)
								pauseBtn.Children(func() {
									ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
										ui.Icon(c, iconPause).Size(14, 14)
										ui.Text(c, "Pause")
									})
								})
								if pauseBtn.Clicked() {
									a.setPaused(true)
								}
							}
						}

						retryBtn := ui.Button(c, "").Label("Retry failed").Radius(12)
						retryBtn.Children(func() {
							ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, iconRefresh).Size(14, 14)
								ui.Text(c, "Retry failed")
							})
						})
						if retryBtn.Clicked() {
							a.retryFailed()
						}
					})
				})
			}

			// Videos Section Header
			ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconWave).Size(16, 16).TextColor(t.Accent)
					ui.Text(c, "Videos").Bold().FontSize(16)
				})
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
					// Status icon
					switch v.status {
					case "done":
						ui.Icon(c, iconCheck).Size(16, 16).TextColor(t.Success)
					case "downloading":
						ui.Icon(c, iconWave).Size(16, 16).TextColor(t.Accent)
					case "failed":
						ui.Icon(c, iconAlert).Size(16, 16).TextColor(t.Danger)
					default:
						ui.Icon(c, iconDownload).Size(16, 16).TextColor(t.TextMuted)
					}

					ui.Column(c).Fill().Gap(2).Children(func() {
						ui.Text(c, v.title).FontSize(13).Bold()
						ui.Text(c, v.status).TextColor(t.TextMuted).FontSize(11)
					})

					if v.status == "done" {
						openBtn := ui.Button(c, "").Label("Open folder").Radius(8)
						openBtn.Children(func() {
							ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, iconFolder).Size(13, 13)
								ui.Text(c, "Open folder")
							})
						})
						if openBtn.Clicked() {
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
