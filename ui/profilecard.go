package ui

import (
	"fmt"
	"os"

	mygoui "github.com/egoist/mygo/ui"
)

// ProfileCardProps defines inputs and callbacks for a single profile card.
type ProfileCardProps struct {
	ID           int64
	Name         string
	URL          string
	AvatarPath   string
	AvatarURL    string
	VideosCount  int
	OnViewVideos func()
	OnOpenFolder func()
}

// RenderProfileCard renders a reusable shadcn-styled card for an archived profile in a grid.
func RenderProfileCard(c *mygoui.Context, props ProfileCardProps) {
	t := c.Theme()

	var bmp *mygoui.Bitmap
	if props.AvatarPath != "" {
		if data, err := os.ReadFile(props.AvatarPath); err == nil {
			if b, err := mygoui.DecodeBitmap(data); err == nil {
				bmp = b
			}
		}
	}

	mygoui.Column(c).FillWidth().Padding(12).Gap(10).Radius(10).Background(t.Surface).Border(1, t.Border).Children(func() {
		// Profile Image (profileImg) centered
		mygoui.Row(c).FillWidth().Center().Children(func() {
			mygoui.Avatar(c, props.Name, bmp).Size(52, 52)
		})

		// Profile Name & URL centered
		mygoui.Column(c).FillWidth().Gap(2).AlignItems(mygoui.Center).Children(func() {
			mygoui.Text(c, props.Name).Bold().FontSize(14).TextColor(t.Text)
			mygoui.Text(c, props.URL).TextColor(t.TextMuted).FontSize(11)
		})

		// Action Buttons Stacked Vertically to prevent horizontal overflow in 3-7 column grids
		mygoui.Column(c).FillWidth().Gap(6).Children(func() {
			if props.OnViewVideos != nil {
				if mygoui.PrimaryButton(c.Key(fmt.Sprintf("view-vids-%d", props.ID)), "View Videos").FillWidth().Height(28).Radius(6).Clicked() {
					props.OnViewVideos()
				}
			}
			if props.OnOpenFolder != nil {
				if mygoui.Button(c.Key(fmt.Sprintf("open-folder-%d", props.ID)), "Open folder").FillWidth().Height(26).Radius(6).Clicked() {
					props.OnOpenFolder()
				}
			}
		})
	})
}
