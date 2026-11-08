package ui

import (
	mygoui "github.com/egoist/mygo/ui"
)

// ProfileCardProps defines inputs and callbacks for a single profile card.
type ProfileCardProps struct {
	ID           int64
	Name         string
	URL          string
	OnViewVideos func()
	OnOpenFolder func()
}

// RenderProfileCard renders a reusable shadcn-styled card for an archived profile.
func RenderProfileCard(c *mygoui.Context, props ProfileCardProps) {
	t := c.Theme()
	mygoui.Row(c).FillWidth().Padding(14).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).AlignItems(mygoui.Center).Children(func() {
		// Icon Container
		mygoui.Column(c).Size(36, 36).Center().Radius(6).Background(t.SurfaceHover).Children(func() {
			mygoui.Icon(c, IconFolder).Size(18, 18).TextColor(t.Text)
		})

		// Profile Info
		mygoui.Column(c).Fill().Gap(2).Children(func() {
			mygoui.Text(c, props.Name).Bold().FontSize(14)
			mygoui.Text(c, props.URL).TextColor(t.TextMuted).FontSize(12)
		})

		// Actions
		mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
			if props.OnOpenFolder != nil {
				if mygoui.Button(c.Key("open-folder"), "Open folder").Radius(6).Clicked() {
					props.OnOpenFolder()
				}
			}
			if props.OnViewVideos != nil {
				if mygoui.PrimaryButton(c.Key("view-vids"), "View Videos").Radius(6).Clicked() {
					props.OnViewVideos()
				}
			}
		})
	})
}
