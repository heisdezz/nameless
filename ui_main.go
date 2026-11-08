package main

import (
	"github.com/egoist/mygo/ui"
)

func (a *app) viewMain(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Center().Gap(16).Padding(24).Children(func() {
		ui.Text(c, "nameless").FontSize(26).Bold()
		ui.Text(c, "Bulk TikTok profile downloader").TextColor(t.TextMuted).FontSize(13)
		ui.TextInput(c, &a.profileURL).Placeholder("https://www.tiktok.com/@username").Label("Profile URL").Width(360)
		ui.TextInput(c, &a.outDir).Placeholder("Output folder").Label("Save to").Width(360)
		ui.Row(c).Gap(10).Children(func() {
			if ui.PrimaryButton(c, "Fetch profile").Clicked() {
				a.fetchProfile()
			}
			if ui.Button(c, "Settings").Clicked() {
				a.enterSettings()
			}
		})
		if a.fetchErr != "" {
			ui.Text(c, a.fetchErr).TextColor(errColor())
		}
		if len(a.profiles) > 0 {
			ui.Text(c, "Recent profiles").Bold().FontSize(14)
			for i := range a.profiles {
				p := a.profiles[i]
				if ui.Button(c, p.name).Clicked() {
					a.openProfile(p)
				}
			}
		}
	})
}
