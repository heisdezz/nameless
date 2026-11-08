package main

import (
	"github.com/egoist/mygo/ui"
)

// viewSettings is the M0 placeholder; WS-D replaces this file in D2/D3.
func (a *app) viewSettings(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Gap(12).Padding(20).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			if ui.Button(c, "← Back").Clicked() {
				a.screen = screenMain
			}
			ui.Text(c, "Settings").FontSize(20).Bold()
		})
		ui.Text(c, "Theme (customizable)").Bold()
		ui.Text(c, "Accent color, spacing, and corner radius are applied on next build.").TextColor(t.TextMuted).FontSize(12)
		ui.TextInput(c, &a.cfg.Theme.AccentHex).Label("Accent hex").Width(200)
		ui.Text(c, "Spacing").FontSize(13)
		ui.Text(c, "Corner radius").FontSize(13)
		ui.Text(c, "Dark mode").FontSize(13)
	})
}
