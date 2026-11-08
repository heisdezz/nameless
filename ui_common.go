package main

import (
	"github.com/egoist/mygo/ui"
	appui "nameless/ui"
)

// shadcnTheme returns the dark shadcn / Next.js zinc theme tokens.
func shadcnTheme() ui.Theme {
	return appui.ShadcnTheme()
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

// viewSidebar renders the left sidebar using the modular appui.RenderSidebar component.
func (a *app) viewSidebar(c *ui.Context) {
	activeScreen := "home"
	switch a.screen {
	case screenProfilesList:
		activeScreen = "profiles"
	case screenSettings:
		activeScreen = "settings"
	}

	items := make([]appui.SidebarItem, len(a.profiles))
	for i, p := range a.profiles {
		isActive := a.screen == screenProfile && a.currentProfile != nil && a.currentProfile.id == p.id
		isRunning := a.downloading && a.currentProfile != nil && a.currentProfile.id == p.id
		items[i] = appui.SidebarItem{
			ID:        p.id,
			Name:      p.name,
			IsActive:  isActive,
			IsRunning: isRunning,
		}
	}

	appui.RenderSidebar(c, appui.SidebarProps{
		ActiveScreen: activeScreen,
		Profiles:     items,
		Filter:       &a.sidebarFilter,
		EngineStatus: a.ytdlpVersion,
		OnHome: func() {
			a.screen = screenMain
		},
		OnProfiles: func() {
			a.screen = screenProfilesList
		},
		OnSettings: func() {
			a.enterSettings()
		},
		OnSelectProfile: func(id int64) {
			for _, p := range a.profiles {
				if p.id == id {
					a.openProfile(p)
					break
				}
			}
		},
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
func errColor() ui.Color { return appui.ErrColor() }
