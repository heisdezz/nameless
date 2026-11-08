package main

import (
	"fmt"

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
			ui.Icon(c, iconWave).Size(24, 24).TextColor(t.Accent)
		})

		ui.Box(c).Height(1).Width(36).Background(t.Border)

		// Home / New Download Button
		homeBtn := ui.Button(c.Key("rail-home"), "").Label("New Download").Tooltip("New Download").Size(44, 40).Radius(12)
		homeBtn.Children(func() {
			ui.Icon(c, iconDownload).Size(20, 20)
		})
		if a.screen == screenMain {
			homeBtn.Background(t.Accent).TextColor(t.AccentText)
		}
		if homeBtn.Clicked() {
			a.screen = screenMain
		}

		// Profiles / Library Button
		profBtn := ui.Button(c.Key("rail-profiles"), "").Label("Profiles").Tooltip("Profiles").Size(44, 40).Radius(12)
		profBtn.Children(func() {
			ui.Icon(c, iconFolder).Size(20, 20)
		})
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
		settingsBtn := ui.Button(c.Key("rail-settings"), "").Label("Settings").Tooltip("Settings").Size(44, 40).Radius(12)
		settingsBtn.Children(func() {
			ui.Icon(c, iconSettings).Size(20, 20)
		})
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
		ui.Row(c).Width(180).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Icon(c, iconWave).Size(18, 18).TextColor(t.Accent)
			ui.Column(c).Gap(2).Children(func() {
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
		})

		// Center: Transport / Batch controls & Progress summary
		ui.Row(c).Fill().Center().Gap(10).Children(func() {
			if a.downloading {
				if a.paused {
					resumeBtn := ui.PrimaryButton(c.Key("dock-resume"), "").Label("Resume").Radius(12)
					resumeBtn.Children(func() {
						ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconPlay).Size(14, 14)
							ui.Text(c, "Resume")
						})
					})
					if resumeBtn.Clicked() {
						a.setPaused(false)
					}
				} else {
					pauseBtn := ui.Button(c.Key("dock-pause"), "").Label("Pause").Radius(12)
					pauseBtn.Children(func() {
						ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
							ui.Icon(c, iconPause).Size(14, 14)
							ui.Text(c, "Pause")
						})
					})
					if pauseBtn.Clicked() {
						a.setPaused(true)
					}
				}
			} else if a.currentProfile != nil && len(a.videos) > 0 {
				dlBtn := ui.PrimaryButton(c.Key("dock-download"), "").Label("Download all").Radius(12)
				dlBtn.Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, iconPlay).Size(14, 14)
						ui.Text(c, "Download all")
					})
				})
				if dlBtn.Clicked() {
					a.startDownload()
				}
			}

			if summary, _, _ := videoSummary(a.videos); summary != "" {
				ui.Text(c, summary).TextColor(t.Accent).Bold().FontSize(12)
			}
		})

		// Right: Actions (Open folder, Engine status)
		ui.Row(c).Width(180).Justify(ui.End).AlignItems(ui.Center).Gap(8).Children(func() {
			if summary, complete, hasFailed := videoSummary(a.videos); complete && !hasFailed && summary != "" {
				dockFolderBtn := ui.Button(c.Key("dock-open-folder"), "").Label("Open output folder").Radius(8)
				dockFolderBtn.Children(func() {
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, iconFolder).Size(13, 13)
						ui.Text(c, "Open output folder")
					})
				})
				if dockFolderBtn.Clicked() {
					a.openFolder(c, a.effectiveOutputDir())
				}
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
