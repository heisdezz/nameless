package pages

import (
	mygoui "github.com/egoist/mygo/ui"
)

// SettingsProps defines inputs and bindings for the Settings page.
type SettingsProps struct {
	OutDir     *string
	DelayStr   *string
	TimeoutStr *string
	Cookie     *string
	SessionID  *string
	AccentHex  *string
	Spacing    string
	OnBack     func()
}

// RenderSettings renders the modular settings page.
func RenderSettings(c *mygoui.Context, props SettingsProps) {
	t := c.Theme()
	mygoui.Scroll(c).Fill().Children(func() {
		mygoui.Column(c).Fill().Gap(16).Padding(24).Children(func() {
			mygoui.Row(c).Gap(10).AlignItems(mygoui.Center).Children(func() {
				if mygoui.Button(c, "← Back").Radius(6).Clicked() {
					if props.OnBack != nil {
						props.OnBack()
					}
				}
				mygoui.Text(c, "Settings").FontSize(20).Bold()
			})

			// Download & Engine Settings Card
			mygoui.Column(c).FillWidth().Padding(16).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				mygoui.Text(c, "Download & Engine Settings").Bold().FontSize(14)
				mygoui.Text(c, "Configure default download location, concurrency workers, and timeouts.").TextColor(t.TextMuted).FontSize(12)

				if props.OutDir != nil {
					mygoui.TextInput(c, props.OutDir).Label("Default Download Location").FillWidth()
				}
				if props.DelayStr != nil {
					mygoui.TextInput(c, props.DelayStr).Label("Delay between downloads (e.g. 1s)").FillWidth()
				}
				if props.TimeoutStr != nil {
					mygoui.TextInput(c, props.TimeoutStr).Label("Per-video timeout (e.g. 5m)").FillWidth()
				}
			})

			// Authentication Card
			mygoui.Column(c).FillWidth().Padding(16).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				mygoui.Text(c, "TikTok Authentication").Bold().FontSize(14)
				mygoui.Text(c, "Used when archiving private or age-restricted TikTok accounts.").TextColor(t.TextMuted).FontSize(12)

				if props.Cookie != nil {
					mygoui.TextInput(c, props.Cookie).Label("Cookies file path").FillWidth()
				}
				if props.SessionID != nil {
					mygoui.TextInput(c, props.SessionID).Label("Session ID").FillWidth()
				}
			})

			// Theme & Appearance Card
			mygoui.Column(c).FillWidth().Padding(16).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				mygoui.Text(c, "Theme & Appearance").Bold().FontSize(14)
				mygoui.Text(c, "Accent color, spacing, and corner radius are applied on next build.").TextColor(t.TextMuted).FontSize(12)

				if props.AccentHex != nil {
					mygoui.TextInput(c, props.AccentHex).Label("Accent hex").Width(200)
				}
				mygoui.Text(c, "Spacing: "+props.Spacing).FontSize(13)
				mygoui.Text(c, "Corner radius (DIP)").FontSize(13)
				mygoui.Text(c, "Dark mode").FontSize(13)
			})
		})
	})
}
