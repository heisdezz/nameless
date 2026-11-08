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

	"nameless/internal/config"

	"github.com/egoist/mygo/ui"
)

// ErrNotImplemented marks skeleton bodies; workstream D replaces them.
var ErrNotImplemented = errors.New("theme: not implemented (workstream D, progress.md D1)")

// Apply overlays cfg on base: Accent (+Hover/Pressed mixes), Radius, Spacing,
// and the Dark base choice. The caller applies the result via SetTheme.
func Apply(base *ui.Theme, cfg config.ThemeConfig) { panic(ErrNotImplemented) }

// ParseHex accepts #rgb and #rrggbb; on error the caller keeps the previous value.
func ParseHex(hex string) (ui.Color, error) { return ui.Color{}, ErrNotImplemented }

// Presets returns the named theme presets ("Indigo", "Forest", "Sunset", "Mono").
func Presets() map[string]config.ThemeConfig { return nil }
