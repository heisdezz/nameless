package main

import (
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
		for i := range a.videos {
			v := a.videos[i]
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, v.title).FontSize(13)
				ui.Text(c, v.status).TextColor(t.TextMuted).FontSize(12)
			})
		}
	})
}
