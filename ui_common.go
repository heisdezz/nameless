package main

import "github.com/egoist/mygo/ui"

func (a *app) view(c *ui.Context) {
	a.drainToasts(c)
	switch a.screen {
	case screenMain:
		a.viewMain(c)
	case screenProfile:
		a.viewProfile(c)
	case screenSettings:
		a.viewSettings(c)
	}
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
func errColor() ui.Color { return ui.Color{R: 220, G: 60, B: 60} }
