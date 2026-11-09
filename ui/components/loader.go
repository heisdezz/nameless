package components

import (
	mygoui "github.com/egoist/mygo/ui"
)

// LoaderProps defines inputs for the Loader component.
type LoaderProps struct {
	// Size of the loader spinner (optional)
	Size int
	// Color of the loader (optional)
	Color mygoui.Color
	// Text shown below the loader (optional)
	Text string
	// Whether to show a subtle background overlay
	ShowOverlay bool
}

// RenderLoader renders a reusable loading spinner component.
// Can be used for page loading, section loading, or content loading states.
func RenderLoader(c *mygoui.Context, props LoaderProps) {
	t := c.Theme()

	// Default props values
	size := 32
	if props.Size > 0 {
		size = props.Size
	}

	color := t.Accent
	if props.Color != (mygoui.Color{}) {
		color = props.Color
	}

	mygoui.Column(c.Key("loader-root")).
		Center().
		Children(func() {
			if props.ShowOverlay {
				mygoui.Box(c.Key("loader-overlay")).
					Fill().
					Background(mygoui.Hex("#0000004d")).
					Children(func() {
						renderLoaderContent(c, size, color, props.Text)
					})
			} else {
				renderLoaderContent(c, size, color, props.Text)
			}
		})
}

// renderLoaderContent renders the actual loader spinner and optional text.
func renderLoaderContent(c *mygoui.Context, size int, color mygoui.Color, text string) {
	t := c.Theme()

	mygoui.Column(c.Key("loader-content")).
		Center().Gap(12).Children(func() {
		// Self-animating spinner (paint-only motion: it keeps
		// turning without rebuilding the view every frame).
		mygoui.Spinner(c.Key("loader-icon")).Size(float32(size), float32(size)).TextColor(color).Label(text)

		// Loading text
		if text != "" {
			mygoui.Text(c.Key("loader-text"), text).FontSize(14).TextColor(t.TextMuted)
		}
	})
}
