package theme

import (
	"testing"

	"nameless/internal/config"
	appui "nameless/ui"
)

func TestApplyOverlaysConfig(t *testing.T) {
	base := appui.ShadcnTheme()
	Apply(&base, config.ThemeConfig{AccentHex: "#f43f5e", Dark: false, Radius: 16, Spacing: 6})
	if base.Radius != 16 || base.Spacing != 6 || base.Dark != false {
		t.Fatalf("radius/spacing/dark not applied: %+v", base)
	}
	// #f43f5e
	if base.Accent.R != 244 || base.Accent.G != 63 || base.Accent.B != 94 {
		t.Fatalf("accent not applied: %+v", base.Accent)
	}
	if base.AccentHover == base.Accent || base.AccentPressed == base.Accent {
		t.Fatalf("hover/pressed should be derived mixes, got hover=%+v pressed=%+v accent=%+v", base.AccentHover, base.AccentPressed, base.Accent)
	}
}

func TestApplyInvalidHexKeepsBaseAccent(t *testing.T) {
	base := appui.ShadcnTheme()
	want := base.Accent
	Apply(&base, config.ThemeConfig{AccentHex: "not-a-color", Dark: true, Radius: 8, Spacing: 4})
	if base.Accent != want {
		t.Fatalf("invalid hex should keep base accent, got %+v want %+v", base.Accent, want)
	}
}

func TestParseHex(t *testing.T) {
	c, err := ParseHex("#10b981")
	if err != nil || c.R != 16 || c.G != 185 || c.B != 129 {
		t.Fatalf("ParseHex #10b981: %v %+v", err, c)
	}
	if _, err := ParseHex("nope"); err == nil {
		t.Fatal("expected error for invalid hex")
	}
	c, err = ParseHex("#f00")
	if err != nil || c.R != 255 || c.G != 0 || c.B != 0 {
		t.Fatalf("ParseHex #f00: %v %+v", err, c)
	}
}

func TestLightShadcnTheme(t *testing.T) {
	lt := appui.LightShadcnTheme()
	if lt.Dark {
		t.Fatal("light theme must not be dark")
	}
	if lt.Background.R != 255 || lt.Text.R != 9 {
		t.Fatalf("light zinc tokens wrong: bg=%+v text=%+v", lt.Background, lt.Text)
	}
	dt := appui.ShadcnTheme()
	if !dt.Dark || dt.Background.R != 9 {
		t.Fatalf("dark zinc tokens wrong: %+v", dt)
	}
}

func TestApplyDarkPresetRecolorsWindow(t *testing.T) {
	base := appui.ShadcnTheme()
	cfg := config.ThemeConfig{Preset: "Nord", AccentHex: "#88c0d0", Dark: true, Radius: 8, Spacing: 4}
	want, _ := appui.ThemePresetByName("Nord")
	Apply(&base, cfg)
	wantBG, _ := ParseHex(want.Background)
	wantText, _ := ParseHex(want.Text)
	wantDanger, _ := ParseHex(want.Danger)
	if base.Background != wantBG {
		t.Fatalf("dark preset background not applied: got %+v want %+v", base.Background, wantBG)
	}
	if base.Text != wantText {
		t.Fatalf("dark preset text not applied: got %+v want %+v", base.Text, wantText)
	}
	if base.Danger != wantDanger {
		t.Fatalf("dark preset danger not applied: got %+v want %+v", base.Danger, wantDanger)
	}
}

func TestApplyLightPresetKeepsBaseSurfaces(t *testing.T) {
	base := appui.LightShadcnTheme()
	baseBackground := base.Background
	baseSurface := base.Surface
	Apply(&base, config.ThemeConfig{Preset: "Tokyo Night", AccentHex: "#7aa2f7", Dark: false, Radius: 8, Spacing: 4})
	if base.Background != baseBackground || base.Surface != baseSurface {
		t.Fatalf("light mode must keep zinc surfaces: bg=%+v surface=%+v", base.Background, base.Surface)
	}
	wantAccent, _ := ParseHex("#7aa2f7")
	if base.Accent != wantAccent {
		t.Fatalf("light mode should keep configured accent: got %+v want %+v", base.Accent, wantAccent)
	}
}

func TestApplyUnknownPresetFallsBack(t *testing.T) {
	base := appui.ShadcnTheme()
	Apply(&base, config.ThemeConfig{Preset: "Nope", AccentHex: "#f43f5e", Dark: true, Radius: 8, Spacing: 4})
	if base.Background.R != 9 || base.Background.G != 9 || base.Background.B != 11 {
		t.Fatalf("unknown preset should keep shadcn base: %+v", base.Background)
	}
}

func TestPresetPickerCoversPopularThemes(t *testing.T) {
	var names []string
	for _, p := range appui.ThemePresets() {
		names = append(names, p.Name)
	}
	want := []string{"Shadcn", "Catppuccin", "Atom One", "Dracula", "Nord", "Tokyo Night"}
	if len(names) != len(want) {
		t.Fatalf("expected %d presets, got %v", len(want), names)
	}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("preset %d = %q, want %q (all: %v)", i, names[i], name, names)
		}
		if _, ok := appui.ThemePresetByName(name); !ok {
			t.Fatalf("ThemePresetByName(%q) should find preset", name)
		}
	}
}

func TestPresetsMatchSettingsUI(t *testing.T) {
	for _, name := range []string{"Zinc", "Emerald", "Violet", "Indigo", "Rose"} {
		if _, ok := Presets()[name]; !ok {
			t.Fatalf("missing preset %q", name)
		}
	}
}
