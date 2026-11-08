package main

import "github.com/egoist/mygo/ui"

func (a *app) view(c *ui.Context) {
	switch a.screen {
	case screenMain:
		a.viewMain(c)
	case screenProfile:
		a.viewProfile(c)
	case screenSettings:
		a.viewSettings(c)
	}
}

// errColor colors error text (plan §4: fail loudly and recoverably).
func errColor() ui.Color { return ui.Color{R: 220, G: 60, B: 60} }
