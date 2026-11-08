package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

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
	DateStr      string
	OnViewVideos func()
	OnOpenFolder func()
	OnDelete     func()
}

// formatSubtitle formats a clean path/handle for the profile without showing the full URL.
func formatSubtitle(url, name string) string {
	if name != "" {
		return "media/profiles/" + name
	}
	if strings.Contains(url, "@") {
		idx := strings.Index(url, "@")
		sub := url[idx:]
		if s := strings.Index(sub, "/"); s > 0 {
			sub = sub[:s]
		}
		return sub
	}
	return "media/profiles"
}

// RenderProfileCard renders a sleek media card for an archived profile in the grid.
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

	dateStr := props.DateStr
	if dateStr == "" {
		dateStr = time.Now().Format("02/01/2006")
	}

	subtitle := formatSubtitle(props.URL, props.Name)

	mygoui.Column(c).FillWidth().Radius(12).Background(t.Surface).Border(1, t.Border).Clip().Children(func() {
		// 1. Top Cover Thumbnail / Banner
		coverBox := mygoui.Box(c.Key(fmt.Sprintf("card-cover-%d", props.ID))).FillWidth().Height(110).Background(mygoui.Hex("#1c1c20")).Clip()
		coverBox.Children(func() {
			if bmp != nil {
				mygoui.Image(c, bmp).FillWidth().Height(110).Clip()
			} else {
				// Fallback cover placeholder
				mygoui.Column(c).Fill().Center().Background(mygoui.Hex("#18181c")).Children(func() {
					mygoui.Avatar(c, props.Name, nil).Size(46, 46)
				})
			}

			// Pill badge overlaid at bottom-right of the cover (Green stack icon + item count)
			mygoui.Row(c).Absolute().Right(8).Bottom(8).Padding(3, 8).Gap(5).AlignItems(mygoui.Center).
				Radius(6).Background(mygoui.Hex("#09090bcc")).Border(1, mygoui.Hex("#27272a")).Children(func() {
				mygoui.Icon(c, IconLayers).Size(12, 12).TextColor(mygoui.Hex("#22c55e"))
				mygoui.Text(c, fmt.Sprintf("%d items", props.VideosCount)).FontSize(11).Bold().TextColor(mygoui.Hex("#ffffff"))
			})
		})
		if coverBox.Clicked() && props.OnViewVideos != nil {
			props.OnViewVideos()
		}

		// 2. Card Content Area
		mygoui.Column(c).FillWidth().Padding(12).Gap(8).Children(func() {
			// Title & Subtitle
			mygoui.Column(c).FillWidth().Gap(2).Children(func() {
				titleEl := mygoui.Text(c, props.Name).Bold().FontSize(14).TextColor(t.Text).SingleLine()
				if titleEl.Clicked() && props.OnViewVideos != nil {
					props.OnViewVideos()
				}
				mygoui.Text(c, subtitle).TextColor(t.TextMuted).FontSize(11).SingleLine()
			})

			// Date and Delete row
			mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
				mygoui.Row(c).AlignItems(mygoui.Center).Gap(4).Children(func() {
					mygoui.Icon(c, IconCalendar).Size(11, 11).TextColor(t.TextMuted)
					mygoui.Text(c, dateStr).FontSize(11).TextColor(t.TextMuted)
				})
				if props.OnDelete != nil {
					delBtn := mygoui.Button(c.Key(fmt.Sprintf("card-del-%d", props.ID)), "").
						Label("Delete").
						Radius(4).
						Background(t.Surface)
					delBtn.Children(func() {
						mygoui.Row(c).Fill().Center().Children(func() {
							mygoui.Icon(c, IconTrash).Size(12, 12).TextColor(mygoui.Hex("#ef4444"))
						})
					})
					if delBtn.Clicked() {
						props.OnDelete()
					}
				}
			})

			// Action Buttons Stacked Vertically to prevent horizontal overflow in 3-7 column grids
			mygoui.Column(c).FillWidth().Gap(6).Children(func() {
				if props.OnViewVideos != nil {
					if mygoui.PrimaryButton(c.Key(fmt.Sprintf("card-view-%d", props.ID)), "View Videos").FillWidth().Height(28).Radius(6).Clicked() {
						props.OnViewVideos()
					}
				}
				if props.OnOpenFolder != nil {
					if mygoui.Button(c.Key(fmt.Sprintf("card-folder-%d", props.ID)), "Open folder").FillWidth().Height(26).Radius(6).Clicked() {
						props.OnOpenFolder()
					}
				}
			})
		})
	})
}
