package ui

import mygoui "github.com/egoist/mygo/ui"

// ShadcnTheme returns the dark shadcn / Next.js zinc theme tokens.
func ShadcnTheme() mygoui.Theme {
	t := *mygoui.DarkTheme()
	t.Dark = true
	// True shadcn / Next.js zinc palette
	t.Background = mygoui.Color{R: 9, G: 9, B: 11, A: 255}       // #09090b (zinc-950)
	t.Surface = mygoui.Color{R: 24, G: 24, B: 27, A: 255}        // #18181b (zinc-900)
	t.SurfaceHover = mygoui.Color{R: 39, G: 39, B: 42, A: 255}   // #27272a (zinc-800)
	t.SurfacePressed = mygoui.Color{R: 32, G: 32, B: 35, A: 255} // #202023
	t.Border = mygoui.Color{R: 39, G: 39, B: 42, A: 255}         // #27272a (zinc-800 hairline)
	t.Text = mygoui.Color{R: 244, G: 244, B: 245, A: 255}        // #f4f4f5 (zinc-100)
	t.TextMuted = mygoui.Color{R: 161, G: 161, B: 170, A: 255}   // #a1a1aa (zinc-400)

	// Primary button in shadcn dark is crisp white background with dark text
	t.Accent = mygoui.Color{R: 250, G: 250, B: 250, A: 255}        // #fafafa
	t.AccentHover = mygoui.Color{R: 228, G: 228, B: 231, A: 255}   // #e4e4e7
	t.AccentPressed = mygoui.Color{R: 212, G: 212, B: 216, A: 255} // #d4d4d8
	t.AccentText = mygoui.Color{R: 9, G: 9, B: 11, A: 255}         // #09090b (dark text on white)

	t.Danger = mygoui.Color{R: 239, G: 68, B: 68, A: 255}  // #ef4444
	t.Success = mygoui.Color{R: 34, G: 197, B: 94, A: 255} // #22c55e

	t.Radius = 6
	t.Spacing = 4
	return t
}

// ErrColor returns the standard danger color for errors.
func ErrColor() mygoui.Color { return mygoui.Color{R: 239, G: 68, B: 68, A: 255} }

// DefaultThemeName is the preset applied when config has none yet.
const DefaultThemeName = "Shadcn"

// ThemePreset is a named, full-surface palette for the app chrome. Every
// color is a #rrggbb hex string. Dark presets recolor the whole window;
// light mode keeps the zinc light chrome and only takes the accent and
// semantic colors from the preset.
type ThemePreset struct {
	Name           string
	Background     string
	Surface        string
	SurfaceHover   string
	SurfacePressed string
	Border         string
	Text           string
	TextMuted      string
	Accent         string
	Danger         string
	Success        string
}

// ThemePresets lists the built-in popular themes: the shadcn zinc
// default plus five well-known editor palettes.
func ThemePresets() []ThemePreset {
	return []ThemePreset{
		{
			Name: "Shadcn", Background: "#09090b", Surface: "#18181b",
			SurfaceHover: "#27272a", SurfacePressed: "#202023", Border: "#27272a",
			Text: "#f4f4f5", TextMuted: "#a1a1aa",
			Accent: "#fafafa", Danger: "#ef4444", Success: "#22c55e",
		},
		{
			Name: "Catppuccin", Background: "#1e1e2e", Surface: "#181825",
			SurfaceHover: "#313244", SurfacePressed: "#292a3d", Border: "#313244",
			Text: "#cdd6f4", TextMuted: "#a6adc8",
			Accent: "#cba6f7", Danger: "#f38ba8", Success: "#a6e3a1",
		},
		{
			Name: "Atom One", Background: "#282c34", Surface: "#21252b",
			SurfaceHover: "#3a3f4b", SurfacePressed: "#2f333d", Border: "#3e4451",
			Text: "#d7dae0", TextMuted: "#5c6370",
			Accent: "#61afef", Danger: "#e06c75", Success: "#98c379",
		},
		{
			Name: "Dracula", Background: "#282a36", Surface: "#21222c",
			SurfaceHover: "#44475a", SurfacePressed: "#363948", Border: "#44475a",
			Text: "#f8f8f2", TextMuted: "#6272a4",
			Accent: "#bd93f9", Danger: "#ff5555", Success: "#50fa7b",
		},
		{
			Name: "Nord", Background: "#2e3440", Surface: "#3b4252",
			SurfaceHover: "#434c5e", SurfacePressed: "#3f4760", Border: "#434c5e",
			Text: "#eceff4", TextMuted: "#7b88a1",
			Accent: "#88c0d0", Danger: "#bf616a", Success: "#a3be8c",
		},
		{
			Name: "Tokyo Night", Background: "#1a1b26", Surface: "#16161e",
			SurfaceHover: "#292e42", SurfacePressed: "#24283b", Border: "#292e42",
			Text: "#c0caf5", TextMuted: "#565f89",
			Accent: "#7aa2f7", Danger: "#f7768e", Success: "#9ece6a",
		},
	}
}

// ThemePresetByName looks a preset up by its display name; unknown names
// fall back to the shadcn default.
func ThemePresetByName(name string) (ThemePreset, bool) {
	for _, p := range ThemePresets() {
		if p.Name == name {
			return p, true
		}
	}
	return ThemePresets()[0], false
}

// LightShadcnTheme returns the light shadcn zinc theme tokens, mirroring
// ShadcnTheme for the non-dark appearance.
func LightShadcnTheme() mygoui.Theme {
	t := *mygoui.LightTheme()
	t.Dark = false
	t.Background = mygoui.Color{R: 255, G: 255, B: 255, A: 255}   // #ffffff
	t.Surface = mygoui.Color{R: 244, G: 244, B: 245, A: 255}      // #f4f4f5 (zinc-100)
	t.SurfaceHover = mygoui.Color{R: 233, G: 233, B: 236, A: 255} // #e9e9ec
	t.SurfacePressed = mygoui.Color{R: 221, G: 221, B: 225, A: 255}
	t.Border = mygoui.Color{R: 228, G: 228, B: 231, A: 255}    // #e4e4e7 (zinc-200)
	t.Text = mygoui.Color{R: 9, G: 9, B: 11, A: 255}           // #09090b (zinc-950)
	t.TextMuted = mygoui.Color{R: 113, G: 113, B: 122, A: 255} // #71717a (zinc-500)
	t.Accent = mygoui.Color{R: 9, G: 9, B: 11, A: 255}
	t.AccentHover = mygoui.Color{R: 39, G: 39, B: 42, A: 255}
	t.AccentPressed = mygoui.Color{R: 63, G: 63, B: 70, A: 255}
	t.AccentText = mygoui.Color{R: 250, G: 250, B: 250, A: 255}
	t.Danger = mygoui.Color{R: 220, G: 38, B: 38, A: 255}
	t.Success = mygoui.Color{R: 22, G: 163, B: 74, A: 255}
	t.Radius = 6
	t.Spacing = 4
	return t
}
