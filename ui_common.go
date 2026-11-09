package main

import (
	"fmt"

	"nameless/internal/theme"
	appui "nameless/ui"

	"github.com/egoist/mygo/ui"
)

// shadcnTheme returns the dark shadcn / Next.js zinc theme tokens.
func shadcnTheme() ui.Theme {
	return appui.ShadcnTheme()
}

// effectiveTheme builds the frame theme from config: light or dark
// shadcn base, with accent/radius/spacing overlaid from Settings.
func (a *app) effectiveTheme() ui.Theme {
	if a.cfg == nil {
		return shadcnTheme()
	}
	var t ui.Theme
	if a.cfg.Theme.Dark {
		t = appui.ShadcnTheme()
	} else {
		t = appui.LightShadcnTheme()
	}
	theme.Apply(&t, a.cfg.Theme)
	return t
}

func (a *app) view(c *ui.Context) {
	theme := a.effectiveTheme()
	c.SetTheme(&theme)
	a.drainToasts(c)
	t := c.Theme()

	// Advance video progress animations one frame forward.
	// Called on each view loop iteration so downloading videos' progress bars
	// animate smoothly from 0% to their target over ~200ms.
	a.advanceVideoProgressInView()

	// Advance theme transition animation one frame forward.
	// When theme settings change (accent, radius, spacing), interpolates
	// between old and new values over ~200ms (20 frames at ~60fps).
	a.advanceThemeAnimation()

	// Apply animated theme tokens if a transition is in progress.
	if a.themeDirty {
		// Cubic ease-out interpolation: 1 - (1 - t)^3
		// where t = themeAnim (0 to 1 float32)
		tVal := a.themeAnim
		inv := 1.0 - float64(tVal)
		e := 1.0 - inv*inv*inv
		// Interpolate radius (float32)
		animRadius := a.themeOld.Radius + (a.themeNew.Radius-a.themeOld.Radius)*float32(e)
		// Interpolate spacing (float32)
		animSpacing := a.themeOld.Spacing + (a.themeNew.Spacing-a.themeOld.Spacing)*float32(e)
		// Interpolate accent color (linear RGB interpolation with cubic ease)
		animAccent := hexToColorMixFloat(a.themeOld.AccentHex, a.themeNew.AccentHex, float64(tVal))
		// Apply the animated values
		t.Radius = animRadius
		t.Spacing = animSpacing
		t.Accent = animAccent
	} else {
		// No transition active — apply fresh theme from config
		t.Radius = theme.Radius
		t.Spacing = theme.Spacing
		t.Accent = theme.Accent
	}

	// Keyboard shortcut: Escape leaves fullscreen video mode, then the
	// player screen; from profile/settings it returns to main.
	c.OnWindowShortcut(0, ui.KeyEscape, func() {
		switch {
		case a.screen == screenPlayer:
			if a.playerFullScreen {
				a.exitPlayerFullScreen()
				return
			}
			if a.player != nil {
				a.player.Pause()
			}
			a.screen = screenProfile
		default:
			a.screen = screenMain
		}
	})

	// Fullscreen video mode: the viewport alone covers the window, with
	// no sidebar or page chrome.
	if a.playerFullScreen && a.screen == screenPlayer {
		a.viewPlayer(c)
		return
	}

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
			case screenPlayer:
				a.viewPlayer(c)
			case screenSettings:
				a.viewSettings(c)
			}
		})
	})
}

// advanceVideoProgressInView animates video progress bars one frame forward.
// Called on each view loop iteration so downloading videos' progress bars
// animate smoothly from 0% to their target over ~200ms.
func (a *app) advanceVideoProgressInView() {
	if a.win == nil {
		return
	}
	for _, v := range a.videos {
		if v.status == "downloading" && v.pct < 1.0 {
			a.animateVideoProgressFrame(v.id)
		}
	}
}

// animateVideoProgressFrame advances the progress animation for a single video by one frame.
// Uses a simple frame counter to ramp from 0 to target progress over ~200ms
// (20 frames at ~60fps), matching the AnimationDuration theme constant.
func (a *app) animateVideoProgressFrame(id int64) {
	if a.animTargetProgress == nil {
		return
	}
	target, hasTarget := a.animTargetProgress[id]
	if !hasTarget {
		return
	}

	const framesForFullAnimation = 20
	a.mu.Lock()
	a.animFrame++

	progress := float64(a.animFrame) / float64(framesForFullAnimation)

	// Clamp to [0, 1]
	if progress > 1.0 {
		progress = 1.0
		a.animFrame = 0
		a.animTargetProgress[id] = 0 // clear target
		a.mu.Unlock()
		return
	}

	// Compute intermediate percentage: progress * target
	pct := progress * target
	a.mu.Unlock()

	a.win.Update(func() {
		a.setVideoProgress(id, pct, int64(progress*100))
	})
}

// advanceThemeAnimation one frame forward.
// When themeDirty is true, interpolates between themeOld and themeNew over frames.
// AnimationDuration is 200ms; at ~60fps this takes ~20 frames to complete.
func (a *app) advanceThemeAnimation() {
	if !a.themeDirty {
		return
	}
	// Increment animation progress
	a.themeAnim += 1.0 / 20.0 // 20 frames = ~200ms at 60fps
	if a.themeAnim >= 1.0 {
		a.themeAnim = 1.0
		a.themeDirty = false
	}
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
		isActive := (a.screen == screenProfile || a.screen == screenPlayer) && a.currentProfile != nil && a.currentProfile.id == p.id
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

// pow returns x raised to the power n.
func pow(x, n float64) float64 {
	if n == 0 {
		return 1
	}
	if n == 1 {
		return x
	}
	result := 1.0
	for i := 0; i < int(n); i++ {
		result *= x
	}
	return result
}

// hexToColorMixFloat linearly interpolates between two hex colors based on progress (0 to 1).
// Both hex strings should be in #rrggbb format. Returns the interpolated ui.Color.
func hexToColorMixFloat(oldHex, newHex string, progress float64) ui.Color {
	// Parse #rrggbb format manually
	getByte := func(hexStr string) uint8 {
		if len(hexStr) != 6 {
			return 0
		}
		var v uint8
		_, err := fmt.Sscanf(hexStr, "%02x", &v)
		if err != nil {
			return 0
		}
		return v
	}
	oldR, oldG, oldB := getByte(oldHex[0:6]), getByte(oldHex[2:4]), getByte(oldHex[4:6])
	newR, newG, newB := getByte(newHex[0:6]), getByte(newHex[2:4]), getByte(newHex[4:6])

	// Alpha is always 255 (opaque)
	lerp := func(x, y uint8) uint8 {
		return uint8(float32(x) + (float32(y)-float32(x))*float32(progress))
	}
	return ui.Color{R: lerp(oldR, newR), G: lerp(oldG, newG), B: lerp(oldB, newB), A: 255}
}
