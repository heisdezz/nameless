package main

import (
	"github.com/egoist/mygo/ui"
)

// shadcnTheme returns the dark shadcn / Next.js zinc theme tokens.
func shadcnTheme() ui.Theme {
	t := *ui.DarkTheme()
	t.Dark = true
	// True shadcn / Next.js zinc palette
	t.Background = ui.Color{R: 9, G: 9, B: 11, A: 255}         // #09090b (zinc-950)
	t.Surface = ui.Color{R: 24, G: 24, B: 27, A: 255}          // #18181b (zinc-900)
	t.SurfaceHover = ui.Color{R: 39, G: 39, B: 42, A: 255}     // #27272a (zinc-800)
	t.SurfacePressed = ui.Color{R: 32, G: 32, B: 35, A: 255}   // #202023
	t.Border = ui.Color{R: 39, G: 39, B: 42, A: 255}           // #27272a (zinc-800 hairline)
	t.Text = ui.Color{R: 244, G: 244, B: 245, A: 255}          // #f4f4f5 (zinc-100)
	t.TextMuted = ui.Color{R: 161, G: 161, B: 170, A: 255}     // #a1a1aa (zinc-400)

	// Primary button in shadcn dark is crisp white background with dark text
	t.Accent = ui.Color{R: 250, G: 250, B: 250, A: 255}        // #fafafa
	t.AccentHover = ui.Color{R: 228, G: 228, B: 231, A: 255}   // #e4e4e7
	t.AccentPressed = ui.Color{R: 212, G: 212, B: 216, A: 255} // #d4d4d8
	t.AccentText = ui.Color{R: 9, G: 9, B: 11, A: 255}         // #09090b (dark text on white)

	t.Danger = ui.Color{R: 239, G: 68, B: 68, A: 255}   // #ef4444
	t.Success = ui.Color{R: 34, G: 197, B: 94, A: 255}  // #22c55e

	t.Radius = 6
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

	// Zero margin around window - sidebar is flush to the left edge
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		// Left Sidebar Navigation Rail
		a.viewSidebar(c)

		// Hairline border separating sidebar from main content
		ui.Box(c).Width(1).FillHeight().Background(t.Border)

		// Main Content Viewport
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

// viewSidebar renders the left sidebar flush against window edges (no outer margins).
func (a *app) viewSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(200).FillHeight().Padding(12).Gap(10).Background(t.Background).Children(func() {
		// App Header / Branding
		ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
			ui.Icon(c, iconWave).Size(18, 18).TextColor(t.Text)
			ui.Text(c, "nameless").Bold().FontSize(15)
		})

		ui.Box(c).Height(1).FillWidth().Background(t.Border)

		// Navigation Links (shadcn style flat list with subtle hover and active highlight)
		ui.Column(c).Gap(4).Children(func() {
			// Home
			homeBtn := ui.Button(c.Key("nav-home"), "").Label("Home").FillWidth().Height(36).Radius(6)
			homeBtn.Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconDownload).Size(16, 16)
					ui.Text(c, "Home").FontSize(13)
				})
			})
			if a.screen == screenMain {
				homeBtn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
			} else {
				homeBtn.Background(t.Background).TextColor(t.TextMuted)
			}
			if homeBtn.Clicked() {
				a.screen = screenMain
			}

			// Profiles
			profBtn := ui.Button(c.Key("nav-profiles"), "").Label("Profiles").FillWidth().Height(36).Radius(6)
			profBtn.Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconFolder).Size(16, 16)
					ui.Text(c, "Profiles").FontSize(13)
				})
			})
			if a.screen == screenProfilesList || a.screen == screenProfile {
				profBtn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
			} else {
				profBtn.Background(t.Background).TextColor(t.TextMuted)
			}
			if profBtn.Clicked() {
				a.screen = screenProfilesList
			}

			// Settings
			settingsBtn := ui.Button(c.Key("nav-settings"), "").Label("Settings").FillWidth().Height(36).Radius(6)
			settingsBtn.Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconSettings).Size(16, 16)
					ui.Text(c, "Settings").FontSize(13)
				})
			})
			if a.screen == screenSettings {
				settingsBtn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
			} else {
				settingsBtn.Background(t.Background).TextColor(t.TextMuted)
			}
			if settingsBtn.Clicked() {
				a.enterSettings()
			}
		})

		// Spacer
		ui.Column(c).Fill().Children(func() {})

		// Engine Status pill at bottom
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Row(c).AlignItems(ui.Center).Gap(6).Padding(4).Children(func() {
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
func errColor() ui.Color { return ui.Color{R: 239, G: 68, B: 68} }
