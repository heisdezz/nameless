package main

import (
	"github.com/egoist/mygo/ui"
)

// shadcnTheme returns the dark shadcn-inspired theme tokens with mint teal accent.
func shadcnTheme() ui.Theme {
	t := *ui.DarkTheme()
	t.Dark = true
	// Deep obsidian/zinc background
	t.Background = ui.Color{R: 9, G: 13, B: 17, A: 255}      // #090d11
	t.Surface = ui.Color{R: 17, G: 24, B: 34, A: 255}        // #111822
	t.SurfaceHover = ui.Color{R: 28, G: 40, B: 52, A: 255}   // #1c2834
	t.SurfacePressed = ui.Color{R: 22, G: 31, B: 42, A: 255} // #161f2a
	t.Border = ui.Color{R: 32, G: 45, B: 58, A: 255}         // hairline border
	t.Text = ui.Color{R: 241, G: 245, B: 249, A: 255}        // slate-100
	t.TextMuted = ui.Color{R: 148, G: 163, B: 184, A: 255}   // slate-400

	// Vibrant Mint Teal (matches dash.png accent)
	t.Accent = ui.Color{R: 45, G: 212, B: 191, A: 255}        // #2dd4bf (teal-400)
	t.AccentHover = ui.Color{R: 20, G: 184, B: 166, A: 255}   // #14b8a6 (teal-500)
	t.AccentPressed = ui.Color{R: 13, G: 148, B: 136, A: 255} // #0d9488 (teal-600)
	t.AccentText = ui.Color{R: 4, G: 47, B: 46, A: 255}       // #042f2e (ultra-dark contrast on mint)

	t.Radius = 16
	t.Spacing = 4
	return t
}

func (a *app) view(c *ui.Context) {
	theme := shadcnTheme()
	c.SetTheme(&theme)
	a.drainToasts(c)
	t := c.Theme()

	// Keyboard shortcut: Escape returns to main screen from profile or settings
	c.OnWindowShortcut(0, ui.KeyEscape, func() {
		a.screen = screenMain
	})

	ui.Row(c).Fill().Padding(12).Gap(12).Background(t.Background).Children(func() {
		// Left Sidebar Navigation Rail
		a.viewSidebar(c)

		// Main Content Viewport (full remaining width and height)
		ui.Column(c).Fill().Children(func() {
			switch a.screen {
			case screenMain:
				a.viewMain(c)
			case screenProfilesList:
				a.viewProfilesList(c)
			case screenProfile:
				a.viewProfile(c)
			case screenSettings:
				a.viewSettings(c)
			}
		})
	})
}

// viewSidebar renders the left sidebar with app branding and primary navigation links.
func (a *app) viewSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(180).FillHeight().Padding(14).Gap(14).Radius(18).Background(t.Surface).Children(func() {
		// App Header / Branding
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconWave).Size(22, 22).TextColor(t.Accent)
			ui.Column(c).Gap(2).Children(func() {
				ui.Text(c, "nameless").Bold().FontSize(17)
				ui.Text(c, "TikTok Downloader").TextColor(t.TextMuted).FontSize(11)
			})
		})

		ui.Box(c).Height(1).FillWidth().Background(t.Border)

		// Navigation Links
		ui.Column(c).Gap(6).Children(func() {
			// Home / New Download
			homeBtn := ui.Button(c.Key("nav-home"), "").Label("Home").FillWidth().Height(40).Radius(12)
			homeBtn.Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconDownload).Size(18, 18)
					ui.Text(c, "Home").Bold().FontSize(13)
				})
			})
			if a.screen == screenMain {
				homeBtn.Background(t.Accent).TextColor(t.AccentText)
			}
			if homeBtn.Clicked() {
				a.screen = screenMain
			}

			// Profiles Page
			profBtn := ui.Button(c.Key("nav-profiles"), "").Label("Profiles").FillWidth().Height(40).Radius(12)
			profBtn.Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconFolder).Size(18, 18)
					ui.Text(c, "Profiles").Bold().FontSize(13)
				})
			})
			if a.screen == screenProfilesList || a.screen == screenProfile {
				profBtn.Background(t.Accent).TextColor(t.AccentText)
			}
			if profBtn.Clicked() {
				a.screen = screenProfilesList
			}

			// Settings Page
			settingsBtn := ui.Button(c.Key("nav-settings"), "").Label("Settings").FillWidth().Height(40).Radius(12)
			settingsBtn.Children(func() {
				ui.Row(c).Gap(10).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconSettings).Size(18, 18)
					ui.Text(c, "Settings").Bold().FontSize(13)
				})
			})
			if a.screen == screenSettings {
				settingsBtn.Background(t.Accent).TextColor(t.AccentText)
			}
			if settingsBtn.Clicked() {
				a.enterSettings()
			}
		})

		// Spacer
		ui.Column(c).Fill().Children(func() {})

		// Engine Status at bottom
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			version := a.ytdlpVersion
			if version == "" {
				version = "yt-dlp Ready"
			}
			ui.Text(c, version).TextColor(t.TextMuted).FontSize(11)
		})
	})
}

// drainToasts flushes queued transient notifications into the current frame.
func (a *app) drainToasts(c *ui.Context) {
	a.mu.Lock()
	toasts := a.toasts
	a.toasts = nil
	a.mu.Unlock()
	for _, t := range toasts {
		c.AddToast(ui.Toast{
			Title:       t.title,
			Description: t.description,
			Type:        t.toastType,
		})
	}
}

// errColor colors error text (plan §4: fail loudly and recoverably).
func errColor() ui.Color { return ui.Color{R: 220, G: 60, B: 60} }
