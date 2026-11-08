package main

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// shadcnTheme returns the dark shadcn / Next.js zinc theme tokens.
func shadcnTheme() ui.Theme {
	t := *ui.DarkTheme()
	t.Dark = true
	// True shadcn / Next.js zinc palette
	t.Background = ui.Color{R: 9, G: 9, B: 11, A: 255}       // #09090b (zinc-950)
	t.Surface = ui.Color{R: 24, G: 24, B: 27, A: 255}        // #18181b (zinc-900)
	t.SurfaceHover = ui.Color{R: 39, G: 39, B: 42, A: 255}   // #27272a (zinc-800)
	t.SurfacePressed = ui.Color{R: 32, G: 32, B: 35, A: 255} // #202023
	t.Border = ui.Color{R: 39, G: 39, B: 42, A: 255}         // #27272a (zinc-800 hairline)
	t.Text = ui.Color{R: 244, G: 244, B: 245, A: 255}        // #f4f4f5 (zinc-100)
	t.TextMuted = ui.Color{R: 161, G: 161, B: 170, A: 255}   // #a1a1aa (zinc-400)

	// Primary button in shadcn dark is crisp white background with dark text
	t.Accent = ui.Color{R: 250, G: 250, B: 250, A: 255}        // #fafafa
	t.AccentHover = ui.Color{R: 228, G: 228, B: 231, A: 255}   // #e4e4e7
	t.AccentPressed = ui.Color{R: 212, G: 212, B: 216, A: 255} // #d4d4d8
	t.AccentText = ui.Color{R: 9, G: 9, B: 11, A: 255}         // #09090b (dark text on white)

	t.Danger = ui.Color{R: 239, G: 68, B: 68, A: 255}  // #ef4444
	t.Success = ui.Color{R: 34, G: 197, B: 94, A: 255} // #22c55e

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
		// Left Sidebar Navigation (Shadcn / Next.js replicated)
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

// viewSidebar renders the left sidebar matching shadcn/ui and modern AI desktop references.
func (a *app) viewSidebar(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Width(240).FillHeight().Padding(10).Gap(8).Background(t.Background).Children(func() {
		// 1. App Header / Workspace Selector (Codex / ChatGPT & IDE ref)
		ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
			ui.Column(c).Size(24, 24).Center().Radius(6).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Icon(c, iconWave).Size(14, 14).TextColor(t.Text)
			})
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "nameless").Bold().FontSize(14)
				ui.Icon(c, iconChevronDn).Size(12, 12).TextColor(t.TextMuted)
			})
			ui.Column(c).Fill().Children(func() {})
			ui.Icon(c, iconSidebar).Size(16, 16).TextColor(t.TextMuted)
		})

		// 2. Search pill input (Ref 1 & Ref 2)
		ui.Row(c).FillWidth().Gap(6).AlignItems(ui.Center).Padding(6).Radius(6).Background(t.Surface).Border(1, t.Border).Children(func() {
			ui.Icon(c, iconSearch).Size(13, 13).TextColor(t.TextMuted)
			ui.TextInput(c, &a.sidebarFilter).Placeholder("Search profiles...").FillWidth()
		})

		// 3. Primary Navigation Group (Home, Profiles, Settings)
		ui.Column(c).Gap(2).FillWidth().Children(func() {
			// Home
			homeBtn := ui.Button(c.Key("nav-home"), "").Label("Home").FillWidth().Height(34).Radius(6)
			homeBtn.Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconDownload).Size(15, 15)
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
			profBtn := ui.Button(c.Key("nav-profiles"), "").Label("Profiles").FillWidth().Height(34).Radius(6)
			profBtn.Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconFolder).Size(15, 15)
					ui.Text(c, "Profiles").FontSize(13)
				})
			})
			if a.screen == screenProfilesList {
				profBtn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
			} else {
				profBtn.Background(t.Background).TextColor(t.TextMuted)
			}
			if profBtn.Clicked() {
				a.screen = screenProfilesList
			}

			// Settings
			settingsBtn := ui.Button(c.Key("nav-settings"), "").Label("Settings").FillWidth().Height(34).Radius(6)
			settingsBtn.Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
					ui.Icon(c, iconSettings).Size(15, 15)
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

		// 4. Section Divider & Header (matching Ref 1, Ref 2, Ref 3)
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Padding(4).Justify(ui.SpaceBetween).Children(func() {
			ui.Text(c, "ARCHIVED PROFILES").FontSize(11).Bold().TextColor(t.TextMuted)
			ui.Text(c, fmt.Sprintf("%d", len(a.profiles))).FontSize(11).TextColor(t.TextMuted)
		})

		// 5. Scrollable Profiles List (The core feature from all 3 refs!)
		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).FillWidth().Gap(2).Children(func() {
				count := 0
				filter := strings.ToLower(strings.TrimSpace(a.sidebarFilter))
				for i := range a.profiles {
					p := a.profiles[i]
					if filter != "" && !strings.Contains(strings.ToLower(p.name), filter) {
						continue
					}
					count++
					isActive := a.screen == screenProfile && a.currentProfile != nil && a.currentProfile.id == p.id
					profItem := ui.Button(c.Key(fmt.Sprintf("side-item-%d", p.id)), "").Label(p.name).FillWidth().Height(36).Radius(6)
					profItem.Children(func() {
						ui.Row(c).FillWidth().Gap(8).AlignItems(ui.Center).Padding(4).Children(func() {
							// Avatar badge (Ref 1 [L], [H], [T] style)
							initial := "P"
							runes := []rune(p.name)
							if len(runes) > 0 {
								initial = strings.ToUpper(string(runes[0]))
							}
							ui.Column(c).Size(22, 22).Center().Radius(4).Background(t.Surface).Border(1, t.Border).Children(func() {
								ui.Text(c, initial).FontSize(11).Bold().TextColor(t.Text)
							})

							// Profile Title
							ui.Text(c, p.name).FontSize(13)

							// Spacer
							ui.Column(c).Fill().Children(func() {})

							// Status Indicator Dot (Ref 1 & Ref 3)
							if a.downloading && a.currentProfile != nil && a.currentProfile.id == p.id {
								ui.Text(c, "●").FontSize(9).TextColor(ui.Color{R: 34, G: 197, B: 94, A: 255}) // green active
							} else {
								ui.Text(c, "●").FontSize(9).TextColor(ui.Color{R: 59, G: 130, B: 246, A: 255}) // blue dot like ref
							}
						})
					})
					if isActive {
						profItem.Background(t.SurfaceHover).TextColor(t.Text).Bold()
					} else {
						profItem.Background(t.Background).TextColor(t.TextMuted)
					}
					if profItem.Clicked() {
						a.openProfile(p)
					}
				}
				if count == 0 {
					ui.Row(c).Padding(8).Children(func() {
						ui.Text(c, "No profiles found").FontSize(12).TextColor(t.TextMuted)
					})
				}
			})
		})

		// 6. Bottom User Account / Engine Card (Ref 2 RC & Ref 3 YL)
		ui.Box(c).Height(1).FillWidth().Background(t.Border)
		ui.Row(c).FillWidth().AlignItems(ui.Center).Gap(8).Padding(6).Radius(6).Background(t.Surface).Border(1, t.Border).Children(func() {
			// Circular Avatar
			ui.Column(c).Size(26, 26).Center().Radius(13).Background(t.SurfaceHover).Children(func() {
				ui.Text(c, "NL").FontSize(10).Bold().TextColor(t.Text)
			})
			// Title & Subtitle
			ui.Column(c).Fill().Gap(1).Children(func() {
				ui.Text(c, "nameless").Bold().FontSize(12).TextColor(t.Text)
				version := a.ytdlpVersion
				if version == "" {
					version = "yt-dlp Ready"
				}
				ui.Text(c, version).FontSize(10).TextColor(t.TextMuted)
			})
			// Live dot
			ui.Text(c, "●").FontSize(8).TextColor(ui.Color{R: 34, G: 197, B: 94, A: 255})
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
