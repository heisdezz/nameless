package main

import (
	appui "nameless/ui"
	"nameless/ui/pages"

	"github.com/egoist/mygo/ui"
)

func (a *app) viewSettings(c *ui.Context) {
	if a.cfg == nil {
		a.cfg = defaultConfig()
	}

	pages.RenderSettings(c, pages.SettingsProps{
		OutDir:             &a.outDir,
		NamingTemplate:     &a.cfg.NamingTemplate,
		WorkersF64:         &a.workersF64,
		ConnectionsF64:     &a.connectionsF64,
		DelayStr:           &a.delayStr,
		TimeoutStr:         &a.timeoutStr,
		VideoQuality:       &a.cfg.VideoQuality,
		WriteThumbnail:     &a.cfg.WriteThumbnail,
		WriteInfoJSON:      &a.cfg.WriteInfoJSON,
		WriteSubtitles:     &a.cfg.WriteSubtitles,
		Cookie:             &a.cfg.Cookie,
		SessionID:          &a.cfg.SessionID,
		CookiesFromBrowser: &a.cfg.CookiesFromBrowser,
		YTDLPPath:          &a.cfg.YTDLPPath,
		TikAPI:             &a.cfg.TikDownloaderAPI,
		TikCookies:         &a.cfg.TikDownloaderCookies,
		TikFormat:          &a.cfg.TikDownloaderFormat,
		AccentHex:          &a.cfg.Theme.AccentHex,
		Dark:               &a.cfg.Theme.Dark,
		ThemePreset:        &a.cfg.Theme.Preset,
		RadiusF64:          &a.radiusF64,
		Spacing:            a.spacingSel,
		EngineStatus:       a.ytdlpVersion,
		OnBack: func() {
			a.screen = screenMain
		},
		OnDetect: func() {
			a.detectYTDLP()
		},
		OnSave: func() {
			a.saveSettings()
		},
		OnResetDefaults: func() {
			a.resetSettingsToDefault()
		},
		OnSelectAccentPreset: func(hex string) {
			a.cfg.Theme.AccentHex = hex
			// Live-preview the transition before Save.
			a.themeNew = a.cfg.Theme
			a.themeDirty = true
		},
		OnSelectThemePreset: func(name string) {
			a.cfg.Theme.Preset = name
			// Hand-picking a scheme also adopts its accent, unless the
			// user then overrides it with an accent preset below.
			if p, ok := appui.ThemePresetByName(name); ok {
				a.cfg.Theme.AccentHex = p.Accent
			}
			a.themeNew = a.cfg.Theme
			a.themeDirty = true
		},
		OnSelectSpacing: func(s string) {
			a.spacingSel = s
			a.cfg.Theme.Spacing = spacingUnit(s)
			a.themeNew = a.cfg.Theme
			a.themeDirty = true
		},
		SaveByUsername: &a.cfg.SaveByUsername,
	})
}
