package ui

import (
	"fmt"
	"strings"

	mygoui "github.com/egoist/mygo/ui"
)

// SidebarItem represents a single profile or session item in the sidebar.
type SidebarItem struct {
	ID        int64
	Name      string
	IsActive  bool
	IsRunning bool
}

// SidebarProps defines inputs and callbacks for RenderSidebar.
type SidebarProps struct {
	ActiveScreen    string // "home", "profiles", "settings"
	Profiles        []SidebarItem
	Filter          *string
	EngineStatus    string
	OnHome          func()
	OnProfiles      func()
	OnSettings      func()
	OnSelectProfile func(id int64)
}

// RenderSidebar renders the authentic shadcn / Next.js sidebar.
func RenderSidebar(c *mygoui.Context, props SidebarProps) {
	t := c.Theme()
	mygoui.Column(c).Width(240).FillHeight().Padding(10).Gap(8).Background(t.Background).Children(func() {
		// 1. App Header / Workspace Selector (Codex / ChatGPT & IDE ref)
		mygoui.Row(c).FillWidth().Gap(8).AlignItems(mygoui.Center).Padding(4).Children(func() {
			mygoui.Column(c).Size(24, 24).Center().Radius(6).Background(t.Surface).Border(1, t.Border).Children(func() {
				mygoui.Icon(c, IconWave).Size(14, 14).TextColor(t.Text)
			})
			mygoui.Row(c).Gap(4).AlignItems(mygoui.Center).Children(func() {
				mygoui.Text(c, "nameless").Bold().FontSize(14)
				mygoui.Icon(c, IconChevronDn).Size(12, 12).TextColor(t.TextMuted)
			})
			mygoui.Column(c).Fill().Children(func() {})
			mygoui.Icon(c, IconSidebar).Size(16, 16).TextColor(t.TextMuted)
		})

		// 2. Search pill input (Ref 1 & Ref 2)
		if props.Filter != nil {
			mygoui.Row(c).FillWidth().Gap(6).AlignItems(mygoui.Center).Padding(6).Radius(6).Background(t.Surface).Border(1, t.Border).Children(func() {
				mygoui.Icon(c, IconSearch).Size(13, 13).TextColor(t.TextMuted)
				mygoui.TextInput(c, props.Filter).Placeholder("Search profiles...").FillWidth()
			})
		}

		// 3. Primary Navigation Group (Home, Profiles, Settings)
		mygoui.Column(c).Gap(2).FillWidth().Children(func() {
			// Home
			homeBtn := mygoui.Button(c.Key("nav-home"), "").Label("Home").FillWidth().Height(34).Radius(6)
			homeBtn.Children(func() {
				mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Padding(4).Children(func() {
					mygoui.Icon(c, IconDownload).Size(15, 15)
					mygoui.Text(c, "Home").FontSize(13)
				})
			})
			if props.ActiveScreen == "home" {
				homeBtn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
			} else {
				homeBtn.Background(t.Background).TextColor(t.TextMuted)
			}
			if homeBtn.Clicked() && props.OnHome != nil {
				props.OnHome()
			}

			// Profiles
			profBtn := mygoui.Button(c.Key("nav-profiles"), "").Label("Profiles").FillWidth().Height(34).Radius(6)
			profBtn.Children(func() {
				mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Padding(4).Children(func() {
					mygoui.Icon(c, IconFolder).Size(15, 15)
					mygoui.Text(c, "Profiles").FontSize(13)
				})
			})
			if props.ActiveScreen == "profiles" {
				profBtn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
			} else {
				profBtn.Background(t.Background).TextColor(t.TextMuted)
			}
			if profBtn.Clicked() && props.OnProfiles != nil {
				props.OnProfiles()
			}

			// Settings
			settingsBtn := mygoui.Button(c.Key("nav-settings"), "").Label("Settings").FillWidth().Height(34).Radius(6)
			settingsBtn.Children(func() {
				mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Padding(4).Children(func() {
					mygoui.Icon(c, IconSettings).Size(15, 15)
					mygoui.Text(c, "Settings").FontSize(13)
				})
			})
			if props.ActiveScreen == "settings" {
				settingsBtn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
			} else {
				settingsBtn.Background(t.Background).TextColor(t.TextMuted)
			}
			if settingsBtn.Clicked() && props.OnSettings != nil {
				props.OnSettings()
			}
		})

		// 4. Section Divider & Header (matching Ref 1, Ref 2, Ref 3)
		mygoui.Box(c).Height(1).FillWidth().Background(t.Border)
		mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Padding(4).Justify(mygoui.SpaceBetween).Children(func() {
			mygoui.Text(c, "ARCHIVED PROFILES").FontSize(11).Bold().TextColor(t.TextMuted)
			mygoui.Text(c, fmt.Sprintf("%d", len(props.Profiles))).FontSize(11).TextColor(t.TextMuted)
		})

		// 5. Scrollable Profiles List (The core feature from all 3 refs!)
		mygoui.Scroll(c).Fill().Children(func() {
			mygoui.Column(c).FillWidth().Gap(2).Children(func() {
				count := 0
				filter := ""
				if props.Filter != nil {
					filter = strings.ToLower(strings.TrimSpace(*props.Filter))
				}
				for _, p := range props.Profiles {
					if filter != "" && !strings.Contains(strings.ToLower(p.Name), filter) {
						continue
					}
					count++
					profID := p.ID
					profItem := mygoui.Button(c.Key(fmt.Sprintf("side-item-%d", p.ID)), "").Label(p.Name).FillWidth().Height(36).Radius(6)
					profItem.Children(func() {
						mygoui.Row(c).FillWidth().Gap(8).AlignItems(mygoui.Center).Padding(4).Children(func() {
							// Avatar badge (Ref 1 [L], [H], [T] style)
							initial := "P"
							runes := []rune(p.Name)
							if len(runes) > 0 {
								initial = strings.ToUpper(string(runes[0]))
							}
							mygoui.Column(c).Size(22, 22).Center().Radius(4).Background(t.Surface).Border(1, t.Border).Children(func() {
								mygoui.Text(c, initial).FontSize(11).Bold().TextColor(t.Text)
							})

							// Profile Title
							mygoui.Text(c, p.Name).FontSize(13)

							// Spacer
							mygoui.Column(c).Fill().Children(func() {})

							// Status Indicator Dot (Ref 1 & Ref 3)
							if p.IsRunning {
								mygoui.Text(c, "●").FontSize(9).TextColor(mygoui.Color{R: 34, G: 197, B: 94, A: 255})
							} else {
								mygoui.Text(c, "●").FontSize(9).TextColor(mygoui.Color{R: 59, G: 130, B: 246, A: 255})
							}
						})
					})
					if p.IsActive {
						profItem.Background(t.SurfaceHover).TextColor(t.Text).Bold()
					} else {
						profItem.Background(t.Background).TextColor(t.TextMuted)
					}
					if profItem.Clicked() && props.OnSelectProfile != nil {
						props.OnSelectProfile(profID)
					}
				}
				if count == 0 {
					mygoui.Row(c).Padding(8).Children(func() {
						mygoui.Text(c, "No profiles found").FontSize(12).TextColor(t.TextMuted)
					})
				}
			})
		})

		// 6. Bottom User Account / Engine Card (Ref 2 RC & Ref 3 YL)
		mygoui.Box(c).Height(1).FillWidth().Background(t.Border)
		mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Gap(8).Padding(6).Radius(6).Background(t.Surface).Border(1, t.Border).Children(func() {
			// Circular Avatar
			mygoui.Column(c).Size(26, 26).Center().Radius(13).Background(t.SurfaceHover).Children(func() {
				mygoui.Text(c, "NL").FontSize(10).Bold().TextColor(t.Text)
			})
			// Title & Subtitle
			mygoui.Column(c).Fill().Gap(1).Children(func() {
				mygoui.Text(c, "nameless").Bold().FontSize(12).TextColor(t.Text)
				version := props.EngineStatus
				if version == "" {
					version = "yt-dlp Ready"
				}
				mygoui.Text(c, version).FontSize(10).TextColor(t.TextMuted)
			})
			// Live dot
			mygoui.Text(c, "●").FontSize(8).TextColor(mygoui.Color{R: 34, G: 197, B: 94, A: 255})
		})
	})
}
