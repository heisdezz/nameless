package main

import (
	"github.com/egoist/mygo/ui"
)

func (a *app) viewMain(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Gap(14).Padding(24).Children(func() {
			ui.Text(c, "New Download").FontSize(22).Bold()
			ui.Text(c, "Paste a TikTok user profile URL to fetch and bulk-download all videos.").TextColor(t.TextMuted).FontSize(13)

			ui.TextInput(c, &a.profileURL).Placeholder("https://www.tiktok.com/@username").Label("Profile URL").FillWidth().MaxWidth(420)
			ui.TextInput(c, &a.outDir).Placeholder("Output folder").Label("Save to").FillWidth().MaxWidth(420)

			ui.Row(c).Gap(10).Children(func() {
				fetchBtn := ui.PrimaryButton(c, "Fetch profile")
				if a.fetching {
					fetchBtn.Disabled(true)
				}
				if fetchBtn.Clicked() {
					a.fetchProfile()
				}
			})

			if a.fetchErr != "" {
				ui.Text(c, a.fetchErr).TextColor(errColor()).FontSize(12)
			}
		})
	})
}
