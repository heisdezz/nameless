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
