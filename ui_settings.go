package main

import (
	"github.com/egoist/mygo/ui"
)

func (a *app) viewSettings(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Gap(16).Padding(24).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				if ui.Button(c, "← Back").Radius(6).Clicked() {
					a.screen = screenMain
				}
				ui.Text(c, "Settings").FontSize(20).Bold()
			})

			// Download & Engine Settings Card
			ui.Column(c).FillWidth().Padding(16).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Text(c, "Download & Engine Settings").Bold().FontSize(14)
				ui.Text(c, "Configure default download location, concurrency workers, and timeouts.").TextColor(t.TextMuted).FontSize(12)

				ui.TextInput(c, &a.outDir).Label("Default Download Location").FillWidth()
				ui.TextInput(c, &a.delayStr).Label("Delay between downloads (e.g. 1s)").FillWidth()
				ui.TextInput(c, &a.timeoutStr).Label("Per-video timeout (e.g. 5m)").FillWidth()
			})

			// Authentication Card
			ui.Column(c).FillWidth().Padding(16).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Text(c, "TikTok Authentication").Bold().FontSize(14)
				ui.Text(c, "Used when archiving private or age-restricted TikTok accounts.").TextColor(t.TextMuted).FontSize(12)

				ui.TextInput(c, &a.cfg.Cookie).Label("Cookies file path").FillWidth()
				ui.TextInput(c, &a.cfg.SessionID).Label("Session ID").FillWidth()
			})

			// Theme & Appearance Card
			ui.Column(c).FillWidth().Padding(16).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Text(c, "Theme & Appearance").Bold().FontSize(14)
				ui.Text(c, "Accent color, spacing, and corner radius are applied on next build.").TextColor(t.TextMuted).FontSize(12)

				ui.TextInput(c, &a.cfg.Theme.AccentHex).Label("Accent hex").Width(200)
				ui.Text(c, "Spacing: "+a.spacingSel).FontSize(13)
				ui.Text(c, "Corner radius (DIP)").FontSize(13)
				ui.Text(c, "Dark mode").FontSize(13)
			})
		})
	})
}
