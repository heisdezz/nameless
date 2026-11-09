// Package theme adapts config.ThemeConfig to ui.Theme tokens.
//
// This is the one internal package allowed to import ui (plan §2 exception,
// contract change log 2026-11-08).
//
// Owned by workstream D (progress.md). Contract: docs/implementation-plan.md §3.
// M0 skeleton: bodies panic with ErrNotImplemented until D1 lands.
package theme

import (
	"errors"
	"fmt"
	"time"

	"nameless/internal/config"
	appui "nameless/ui"

	"github.com/egoist/mygo/ui"
)

// ErrNotImplemented marks skeleton bodies; workstream D replaces them.
var ErrNotImplemented = errors.New("theme: not implemented (workstream D, progress.md D1)")

// AnimationDuration is the default duration for UI animations.
const AnimationDuration = 200 * time.Millisecond

// AnimationEaseOut is the default easing curve.
var AnimationEaseOut = ui.EaseOut

// Apply overlays cfg on base: the named preset palette (dark mode
// recolors the whole window; light keeps the zinc chrome and only takes
// accent + semantic colors), Accent (+Hover/Pressed mixes), Radius,
// Spacing, and the Dark base choice. The caller applies the result via
// SetTheme. Now supports animated transitions via AnimateTo on the theme tokens.
func Apply(base *ui.Theme, cfg config.ThemeConfig) {
	if base == nil {
		return
	}
	// Set base theme tokens from config
	accent, err := ParseHex(cfg.AccentHex)
	if err != nil {
		// Keep the base accent on invalid hex instead of blanking it.
		accent = base.Accent
	}
	base.Accent = accent
	base.AccentHover = mixColor(accent, ui.Color{R: 255, G: 255, B: 255, A: 255}, 0.15)
	base.AccentPressed = mixColor(accent, ui.Color{A: 255}, 0.15)
	base.Dark = cfg.Dark
	base.Radius = cfg.Radius
	base.Spacing = cfg.Spacing

	// Named preset: dark mode recolors the entire window from the preset
	// palette. Light mode keeps the zinc light chrome; preset surfaces are
	// tuned for dark backgrounds and would look wrong on white.
	preset, ok := appui.ThemePresetByName(cfg.Preset)
	if ok && cfg.Dark {
		base.Background = hexToColor(preset.Background)
		base.Surface = hexToColor(preset.Surface)
		base.SurfaceHover = hexToColor(preset.SurfaceHover)
		base.SurfacePressed = hexToColor(preset.SurfacePressed)
		base.Border = hexToColor(preset.Border)
		base.Text = hexToColor(preset.Text)
		base.TextMuted = hexToColor(preset.TextMuted)
		if hexToColor(preset.Accent) != (ui.Color{}) {
			base.Accent = hexToColor(preset.Accent)
			base.AccentHover = mixColor(base.Accent, ui.Color{R: 255, G: 255, B: 255, A: 255}, 0.15)
			base.AccentPressed = mixColor(base.Accent, ui.Color{A: 255}, 0.15)
		}
	}
	// Semantic colors follow the preset in both modes (each palette's
	// danger/success work on white too, so light mode stays coherent).
	if d := hexToColor(preset.Danger); d != (ui.Color{}) {
		base.Danger = d
	}
	if s := hexToColor(preset.Success); s != (ui.Color{}) {
		base.Success = s
	}
}

// mixColor blends c toward target by t (0..1).
func mixColor(c, target ui.Color, t float32) ui.Color {
	lerp := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*t) }
	return ui.Color{R: lerp(c.R, target.R), G: lerp(c.G, target.G), B: lerp(c.B, target.B), A: 255}
}

// ParseHex accepts #rgb, #rgba, #rrggbb and #rrggbbaa; on error the caller
// keeps the previous value.
func ParseHex(hex string) (ui.Color, error) {
	switch len(hex) {
	case 4, 5, 7, 9:
		for _, r := range hex[1:] {
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
				return ui.Color{}, fmt.Errorf("theme: invalid hex color %q", hex)
			}
		}
		if hex[0] != '#' {
			return ui.Color{}, fmt.Errorf("theme: invalid hex color %q", hex)
		}
		return hexToColor(hex), nil
	default:
		return ui.Color{}, fmt.Errorf("theme: invalid hex color %q", hex)
	}
}

// Presets returns the named theme presets ("Indigo", "Forest", "Sunset", "Mono").
func Presets() map[string]config.ThemeConfig {
	return map[string]config.ThemeConfig{
		"Zinc":    {AccentHex: "#fafafa", Dark: true, Radius: 8, Spacing: 4},
		"Emerald": {AccentHex: "#10b981", Dark: true, Radius: 8, Spacing: 4},
		"Violet":  {AccentHex: "#8b5cf6", Dark: true, Radius: 8, Spacing: 4},
		"Indigo":  {AccentHex: "#6366f1", Dark: true, Radius: 8, Spacing: 4},
		"Rose":    {AccentHex: "#f43f5e", Dark: true, Radius: 8, Spacing: 4},
	}
}

// hexToColor converts a hex string to ui.Color.
// Accepts #rgb, #rgba, #rrggbb and #rrggbbaa; invalid input yields
// a transparent color (ParseHex reports the error separately).
func hexToColor(hex string) ui.Color {
	var r, g, b, a uint32
	switch len(hex) {
	case 7: // #rrggbb
		fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
		a = 255
	case 9: // #rrggbbaa
		fmt.Sscanf(hex, "#%02x%02x%02x%02x", &r, &g, &b, &a)
	case 4: // #rgb
		fmt.Sscanf(hex, "#%1x%1x%1x", &r, &g, &b)
		a = 255
		r *= 17
		g *= 17
		b *= 17
	case 5: // #rgba
		fmt.Sscanf(hex, "#%1x%1x%1x%1x", &r, &g, &b, &a)
		r *= 17
		g *= 17
		b *= 17
		a *= 17
	default:
		return ui.Color{}
	}
	return ui.Color{R: uint8(r), G: uint8(g), B: uint8(b), A: uint8(a)}
}

// easeOutCubic is the default easing curve for the theme's generated
// animation values (local twin of ui.EaseOut's shape).
func easeOutCubic(p float32) float32 {
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	inv := 1 - p
	return 1.0 - inv*inv*inv
}

// AnimateThemeRadius animates the radius transition from old to new value.
// Returns the animated value for the frame, from progress (0.0 to 1.0).
func AnimateThemeRadius(old, new float32, progress float32) float32 {
	return old + (new-old)*easeOutCubic(progress)
}

// AnimateThemeSpacing animates the spacing transition from old to new value.
func AnimateThemeSpacing(old, new float32, progress float32) float32 {
	return old + (new-old)*easeOutCubic(progress)
}

// AnimateThemeAccent animates the accent color transition from old to new hex.
func AnimateThemeAccent(oldHex, newHex string, progress float32) ui.Color {
	e := easeOutCubic(progress)
	a, b := hexToColor(oldHex), hexToColor(newHex)
	lerp := func(x, y uint8) uint8 { return uint8(float32(x) + (float32(y)-float32(x))*e) }
	return ui.Color{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: lerp(a.A, b.A)}
}

// ToastAnimation returns the toast opacity for the frame,
// fading in across the toast's lifetime (progress 0.0 to 1.0).
func ToastAnimation(progress float32) float32 {
	return easeOutCubic(progress)
}

// StateBadgeColor returns the color for a video download status badge.
func StateBadgeColor(status string) ui.Color {
	switch status {
	case "done":
		return ui.Color{R: 34, G: 197, B: 94, A: 255} // green
	case "failed":
		return ui.Color{R: 239, G: 68, B: 68, A: 255} // red
	case "downloading":
		return ui.Color{R: 239, G: 166, B: 37, A: 255} // amber/orange
	case "paused":
		return ui.Color{R: 105, G: 106, B: 112, A: 255} // cool gray
	case "pending":
		return ui.Color{R: 161, G: 161, B: 170, A: 255} // muted gray
	default:
		return ui.Color{R: 161, G: 161, B: 170, A: 255} // muted gray
	}
}
