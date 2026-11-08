package main

import (
	"github.com/egoist/mygo/ui"
	"nameless/ui/pages"
)

func (a *app) viewSettings(c *ui.Context) {
	var cookie, sessionID, accentHex *string
	if a.cfg != nil {
		cookie = &a.cfg.Cookie
		sessionID = &a.cfg.SessionID
		accentHex = &a.cfg.Theme.AccentHex
	}

	pages.RenderSettings(c, pages.SettingsProps{
		OutDir:     &a.outDir,
		DelayStr:   &a.delayStr,
		TimeoutStr: &a.timeoutStr,
		Cookie:     cookie,
		SessionID:  sessionID,
		AccentHex:  accentHex,
		Spacing:    a.spacingSel,
		OnBack: func() {
			a.screen = screenMain
		},
	})
}
