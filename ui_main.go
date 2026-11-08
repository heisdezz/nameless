package main

import (
	"github.com/egoist/mygo/ui"
)

func (a *app) viewMain(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Gap(14).Padding(16).Children(func() {
			// Top Profile URL Input Bar (Search style)
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Radius(16).Padding(12).Background(t.Surface).Children(func() {
				ui.TextInput(c, &a.profileURL).Placeholder("https://www.tiktok.com/@username").FillWidth()
				fetchBtn := ui.PrimaryButton(c, "Fetch profile").Radius(12)
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

			// Save Location Bar
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Radius(16).Padding(12).Background(t.Surface).Children(func() {
				ui.Text(c, "Save to:").TextColor(t.TextMuted).FontSize(12)
				ui.TextInput(c, &a.outDir).Placeholder("Output folder (e.g. ./downloads)").FillWidth()
			})

			// Hero Card (mirroring dash.png hero)
			ui.Column(c).FillWidth().Padding(24).Gap(12).Radius(20).Background(t.Surface).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "● BULK PROFILE DOWNLOADER").TextColor(t.Accent).Bold().FontSize(11)
				})
				ui.Text(c, "Archive Public TikTok Profiles").FontSize(22).Bold()
				ui.Text(c, "Download every video from any creator profile offline in parallel. Supports resuming, cookies, and customizable naming.").TextColor(t.TextMuted).FontSize(13)
			})
		})
	})
}
