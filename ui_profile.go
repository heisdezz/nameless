package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *app) viewProfile(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Gap(12).Padding(20).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "← Back").Clicked() {
				a.screen = screenMain
			}
			if a.currentProfile != nil {
				ui.Text(c, a.currentProfile.name).FontSize(20).Bold()
			}
		})
		ui.Row(c).Gap(10).Children(func() {
			if ui.PrimaryButton(c, "Download all").Clicked() {
				a.startDownload()
			}
			if a.downloading {
				if a.paused {
					if ui.Button(c, "Resume").Clicked() {
						a.setPaused(false)
					}
				} else {
					if ui.Button(c, "Pause").Clicked() {
						a.setPaused(true)
					}
				}
			}
			if ui.Button(c, "Retry failed").Clicked() {
				a.retryFailed()
			}
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
					if ui.Button(c, "Open output folder").Clicked() {
						a.openFolder(c, a.effectiveOutputDir())
					}
				}
			})
		}
		for i := range a.videos {
			v := a.videos[i]
			c.Key(v.id)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, v.title).FontSize(13)
				ui.Text(c, v.status).TextColor(t.TextMuted).FontSize(12)
				if v.status == "done" {
					if ui.Button(c, "Open folder").Clicked() {
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
