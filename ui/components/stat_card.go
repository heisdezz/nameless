package components

import (
	mygoui "github.com/egoist/mygo/ui"
)

// StatCardProps defines the inputs for a dashboard statistic/metric card.
type StatCardProps struct {
	Title    string
	Value    string
	Icon     *mygoui.SVG
	FontSize float32
}

// RenderStatCard renders a statistic card with an icon, title, and prominent value.
func RenderStatCard(c *mygoui.Context, props StatCardProps) {
	t := c.Theme()
	fontSize := props.FontSize
	if fontSize <= 0 {
		fontSize = 18
	}

	mygoui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
		mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
			if props.Icon != nil {
				mygoui.Icon(c, props.Icon).Size(14, 14).TextColor(t.TextMuted)
			}
			mygoui.Text(c, props.Title).TextColor(t.TextMuted).FontSize(11)
		})
		mygoui.Text(c, props.Value).Bold().FontSize(fontSize).TextColor(t.Text)
	})
}
