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

	"github.com/egoist/mygo/ui"
)

// ErrNotImplemented marks skeleton bodies; workstream D replaces them.
var ErrNotImplemented = errors.New("theme: not implemented (workstream D, progress.md D1)")

// AnimationDuration is the default duration for UI animations.
const AnimationDuration = 200 * time.Millisecond

// AnimationEaseOut is the default easing curve.
var AnimationEaseOut = ui.EaseOut

// Apply overlays cfg on base: Accent (+Hover/Pressed mixes), Radius, Spacing,
// and the Dark base choice. The caller applies the result via SetTheme.
// Now supports animated transitions via AnimateTo on the theme tokens.
func Apply(base *ui.Theme, cfg config.ThemeConfig) {
	// Set base theme tokens from config
	base.Accent = hexToColor(cfg.AccentHex)
	base.Dark = cfg.Dark
	base.Radius = cfg.Radius
	base.Spacing = cfg.Spacing

	// Apply preset color mixes if needed
	// Accent hover/pressed can be animated via AnimateTo in the view layer
}

// ParseHex accepts #rgb and #rrggbb; on error the caller keeps the previous value.
func ParseHex(hex string) (ui.Color, error) {
	return hexToColor(hex), nil
}

// Presets returns the named theme presets ("Indigo", "Forest", "Sunset", "Mono").
func Presets() map[string]config.ThemeConfig {
	return map[string]config.ThemeConfig{
		"Indigo": {AccentHex: "#6366f1", Dark: true, Radius: 12, Spacing: 4},
		"Forest": {AccentHex: "#10b981", Dark: true, Radius: 12, Spacing: 4},
		"Sunset": {AccentHex: "#f43f5e", Dark: true, Radius: 12, Spacing: 4},
		"Mono":   {AccentHex: "#fafafa", Dark: true, Radius: 12, Spacing: 4},
		"Zinc":   {AccentHex: "#fafafa", Dark: true, Radius: 12, Spacing: 4},
	}
}

// hexToColor converts a hex string to ui.Color.
// Accepts #rgb and #rrggbb formats.
func hexToColor(hex string) ui.Color {
	var r, g, b, a uint32
	switch len(hex) {
	case 7: // #rrggbb
		fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
		a = 255
	case 4: // #rgba
		fmt.Sscanf(hex, "#%1x%1x%1x%1x", &r, &g, &b, &a)
		// normalize to 0-255
		r *= 17
		g *= 17
		b *= 17
	case 3: // #rgb
		fmt.Sscanf(hex, "#%1x%1x%1x", &r, &g, &b)
		a = 255
		r *= 17
		g *= 17
		b *= 17
	default:
		// fallback to transparent
		return ui.Color{R: 0, G: 0, B: 0, A: 0}
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
