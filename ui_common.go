package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *app) view(c *ui.Context) {
	a.drainToasts(c)
	t := c.Theme()

	// Keyboard shortcut: Escape returns to main screen from profile or settings
	if c.Shortcut(0, ui.KeyEscape) {
		if a.screen != screenMain {
			a.screen = screenMain
		}
	}

	ui.Column(c).Fill().Padding(10).Gap(10).Background(t.Background).Children(func() {
		// Main Row: Left Rail + Center Content + Right Panel
		ui.Row(c).Fill().Gap(10).Children(func() {
			a.viewNavRail(c)

			// Center Content
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

			// Right Panel (Recent Profiles)
			a.viewRightQueue(c)
		})

		// Bottom Persistent Dock (Mirroring audio player dock in dash.png)
		a.viewBottomDock(c)
	})
}

// viewNavRail renders the slim left dock with app logo and navigation buttons.
func (a *app) viewNavRail(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(64).FillHeight().Padding(8).Gap(14).AlignItems(ui.Center).Radius(18).Background(t.Surface).Children(func() {
		// App Mark / Waveform Logo
		ui.Row(c).Center().Padding(4).Children(func() {
			ui.Text(c, "≋").FontSize(22).Bold().TextColor(t.Accent)
		})

		ui.Box(c).Height(1).Width(36).Background(t.Border)

		// Home / New Download Button
		homeBtn := ui.Button(c.Key("rail-home"), "📥").Tooltip("New Download").Size(44, 40).Radius(12)
		if a.screen == screenMain {
			homeBtn.Background(t.Accent).TextColor(t.AccentText)
		}
		if homeBtn.Clicked() {
			a.screen = screenMain
		}

		// Profiles / Library Button
		profBtn := ui.Button(c.Key("rail-profiles"), "📁").Tooltip("Profiles").Size(44, 40).Radius(12)
		if a.screen == screenProfile {
			profBtn.Background(t.Accent).TextColor(t.AccentText)
		}
		if profBtn.Clicked() {
			if a.currentProfile != nil {
				a.screen = screenProfile
			} else if len(a.profiles) > 0 {
				a.openProfile(a.profiles[0])
			} else {
				a.screen = screenMain
			}
		}

		// Spacer
		ui.Column(c).Fill().Children(func() {})

		// Settings Button pinned at bottom
		settingsBtn := ui.Button(c.Key("rail-settings"), "⚙").Tooltip("Settings").Size(44, 40).Radius(12)
		if a.screen == screenSettings {
			settingsBtn.Background(t.Accent).TextColor(t.AccentText)
		}
		if settingsBtn.Clicked() {
			a.enterSettings()
		}
	})
}

// viewRightQueue renders the right-hand panel of recent profiles (mirroring "Up next" in dash.png).
func (a *app) viewRightQueue(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(160).FillHeight().Padding(10).Gap(8).Radius(18).Background(t.Surface).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, "Profiles").Bold().FontSize(12)
			ui.Text(c, fmt.Sprintf("%d", len(a.profiles))).TextColor(t.TextMuted).FontSize(11)
		})

		ui.Box(c).Height(1).FillWidth().Background(t.Border)

		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				if len(a.profiles) == 0 {
					ui.Text(c, "No profiles yet").TextColor(t.TextMuted).FontSize(11)
				}
				for i := range a.profiles {
					p := a.profiles[i]
					btn := ui.Button(c.Key(fmt.Sprintf("right-prof-%d", p.id)), p.name).FillWidth().Radius(8)
					if a.screen == screenProfile && a.currentProfile != nil && a.currentProfile.id == p.id {
						btn.Bold().Background(t.SurfaceHover)
					}
					if btn.Clicked() {
						a.openProfile(p)
					}
				}
			})
		})
	})
}

// viewBottomDock renders the floating dock with batch controls and overall status.
func (a *app) viewBottomDock(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Height(56).FillWidth().Padding(10).AlignItems(ui.Center).Radius(18).Background(t.Surface).Children(func() {
		// Left: Current download or profile title
		ui.Column(c).Width(160).Gap(2).Children(func() {
			title := "nameless"
			sub := "Ready to download"
			if a.currentProfile != nil {
				title = a.currentProfile.name
			}
			if a.downloading {
				sub = "Downloading..."
			} else if summary, _, _ := videoSummary(a.videos); summary != "" {
				sub = summary
			}
			ui.Text(c, title).Bold().FontSize(12)
			ui.Text(c, sub).TextColor(t.TextMuted).FontSize(10)
		})

		// Center: Transport / Batch controls & Progress summary
		ui.Row(c).Fill().Center().Gap(10).Children(func() {
			if a.downloading {
				if a.paused {
					if ui.PrimaryButton(c.Key("dock-resume"), "▶ Resume").Radius(12).Clicked() {
						a.setPaused(false)
					}
				} else {
					if ui.Button(c.Key("dock-pause"), "⏸ Pause").Radius(12).Clicked() {
						a.setPaused(true)
					}
				}
			} else if a.currentProfile != nil && len(a.videos) > 0 {
				if ui.PrimaryButton(c.Key("dock-download"), "▶ Download All").Radius(12).Clicked() {
					a.startDownload()
				}
			}

			if summary, _, _ := videoSummary(a.videos); summary != "" {
				ui.Text(c, summary).TextColor(t.Accent).Bold().FontSize(12)
			}
		})

		// Right: Actions (Open folder, Engine status)
		ui.Row(c).Width(160).Justify(ui.End).AlignItems(ui.Center).Gap(8).Children(func() {
			if ui.Button(c.Key("dock-open-folder"), "Open downloads").Radius(8).Clicked() {
				a.openFolder(c, a.effectiveOutputDir())
			}
			ui.Text(c, "yt-dlp").TextColor(t.TextMuted).FontSize(10)
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
