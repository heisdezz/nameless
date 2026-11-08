package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *app) view(c *ui.Context) {
	a.drainToasts(c)
	c.OnShortcut(0, ui.KeyEscape, func() {
		a.screen = screenMain
	})
	ui.Row(c).Fill().Children(func() {
		a.viewSidebar(c)
		ui.Column(c).Fill().Children(func() {
			switch a.screen {
			case screenMain:
				a.viewMain(c)
			case screenProfile:
				a.viewProfile(c)
			case screenSettings:
				a.viewSettings(c)
			}
		})
	})
}

// viewSidebar renders the persistent left navigation rail.
func (a *app) viewSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(170).FillHeight().Padding(12).Gap(10).Background(t.Surface).Children(func() {
		// App Header / Branding
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, "nameless").Bold().FontSize(18)
			ui.Text(c, "TikTok Downloader").TextColor(t.TextMuted).FontSize(11)
		})

		ui.Box(c).Height(1).FillWidth().Background(t.Border)

		// Navigation links
		ui.Column(c).Gap(6).Children(func() {
			newBtn := ui.Button(c.Key("nav-new"), "+ New Download").FillWidth()
			if a.screen == screenMain {
				newBtn.Bold()
			}
			if newBtn.Clicked() {
				a.screen = screenMain
			}

			settingsBtn := ui.Button(c.Key("nav-settings"), "Settings").FillWidth()
			if a.screen == screenSettings {
				settingsBtn.Bold()
			}
			if settingsBtn.Clicked() {
				a.enterSettings()
			}
		})

		ui.Box(c).Height(1).FillWidth().Background(t.Border)

		// Recent Profiles Section Header
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "PROFILES").TextColor(t.TextMuted).Bold().FontSize(11)
		})

		// Scrollable Profile List
		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).Gap(4).FillWidth().Children(func() {
				if len(a.profiles) == 0 {
					ui.Text(c, "No profiles yet").TextColor(t.TextMuted).FontSize(12)
				}
				for i := range a.profiles {
					p := a.profiles[i]
					btn := ui.Button(c.Key(fmt.Sprintf("nav-profile-%d", p.id)), p.name).FillWidth()
					if a.screen == screenProfile && a.currentProfile != nil && a.currentProfile.id == p.id {
						btn.Bold()
					}
					if btn.Clicked() {
						a.openProfile(p)
					}
				}
			})
		})

		// Footer Status
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
			version := a.ytdlpVersion
			if version == "" {
				version = "yt-dlp ready"
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
