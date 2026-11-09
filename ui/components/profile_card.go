package components

import (
	"fmt"
	"os"
	"strings"
	"time"

	"nameless/ui"

	mygoui "github.com/egoist/mygo/ui"
)

// ProfileCardProps defines inputs and callbacks for a single profile card.
type ProfileCardProps struct {
	ID         int64
	Name       string
	URL        string
	AvatarPath string
	AvatarURL  string
	// Avatar is a pre-decoded bitmap (cached by the caller). When set it
	// is used instead of reading+decoding AvatarPath every frame.
	Avatar       *mygoui.Bitmap
	VideosCount  int
	DateStr      string
	Date         time.Time // use time.Time for relative formatting
	OnViewVideos func()
	OnOpenFolder func()
	OnDelete     func()
}

// formatSubtitle formats a clean path/handle for the profile without showing the full URL.
func formatSubtitle(url, name string, preferHandle bool) string {
	if name != "" {
		return "media/profiles/" + name
	}
	if preferHandle && strings.Contains(url, "@") {
		idx := strings.Index(url, "@")
		sub := url[idx:]
		if s := strings.Index(sub, "/"); s > 0 {
			sub = sub[:s]
		}
		return sub
	}
	return "media/profiles"
}

// dateAgo returns a human-readable relative date string like "2 weeks ago"
// or a formatted date if the time is more than 1 year in the past.
func dateAgo(t time.Time) string {
	if t.IsZero() {
		return time.Now().Format("01/02/2006")
	}
	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		return t.Format("01/02/2006")
	}

	years := int(diff.Hours() / 8760)
	if years > 0 {
		if years == 1 {
			return "1 year ago"
		}
		return fmt.Sprintf("%d years ago", years)
	}

	months := int(diff.Hours() / 730)
	if months > 0 {
		if months == 1 {
			return "1 month ago"
		}
		return fmt.Sprintf("%d months ago", months)
	}

	weeks := int(diff.Hours() / 168)
	if weeks > 0 {
		if weeks == 1 {
			return "1 week ago"
		}
		return fmt.Sprintf("%d weeks ago", weeks)
	}

	days := int(diff.Hours() / 24)
	if days > 0 {
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}

	hours := int(diff.Hours())
	if hours > 0 {
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	}

	minutes := int(diff.Minutes())
	if minutes > 0 {
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	}

	return "just now"
}

// RenderProfileCard renders a sleek media card for an archived profile in the grid,
// matching the card reference: portrait cover, item-count badge bottom-right,
// title/path subtitle, and a bottom meta row with date + pencil/trash icons.
func RenderProfileCard(c *mygoui.Context, props ProfileCardProps) {
	t := c.Theme()

	bmp := props.Avatar
	if bmp == nil && props.AvatarPath != "" {
		if data, err := os.ReadFile(props.AvatarPath); err == nil {
			if b, err := mygoui.DecodeBitmap(data); err == nil {
				bmp = b
			}
		}
	}

	dateStr := props.DateStr
	if dateStr == "" {
		if !props.Date.IsZero() {
			dateStr = dateAgo(props.Date)
		} else {
			dateStr = time.Now().Format("02/01/2006")
		}
	}
	subtitle := formatSubtitle(props.URL, props.Name, true)

	mygoui.Column(c).FillWidth().Radius(12).Background(t.Surface).Border(1, t.Border).Clip().Children(func() {
		// 1. Top Cover Thumbnail / Banner (portrait, matches ref/card_ref.png)
		coverBox := mygoui.Box(c.Key(fmt.Sprintf("card-cover-%d", props.ID))).FillWidth().Height(160).Background(mygoui.Hex("#1c1c20")).Clip()
		coverBox.Children(func() {
			if bmp != nil {
				mygoui.Image(c, bmp).FillWidth().Height(160).Clip()
			} else {
				// Fallback cover placeholder
				mygoui.Column(c).Fill().Center().Background(mygoui.Hex("#18181c")).Children(func() {
					mygoui.Avatar(c, props.Name, nil).Size(52, 52)
				})
			}

			// Pill badge overlaid at bottom-right of the cover (Green stack icon + item count)
			mygoui.Row(c).Absolute().Right(8).Bottom(8).Padding(3, 8).Gap(5).AlignItems(mygoui.Center).
				Radius(6).Background(mygoui.Hex("#09090bcc")).Border(1, mygoui.Hex("#27272a")).Children(func() {
				mygoui.Icon(c, ui.IconLayers).Size(12, 12).TextColor(mygoui.Hex("#22c55e"))
				mygoui.Text(c, fmt.Sprintf("%d items", props.VideosCount)).FontSize(11).Bold().TextColor(mygoui.Hex("#ffffff"))
			})
		})
		if coverBox.Clicked() && props.OnViewVideos != nil {
			props.OnViewVideos()
		}

		// 2. Card Content Area — title, path, and bottom meta row (no bulky action buttons)
		mygoui.Column(c).FillWidth().Padding(10, 12).Gap(2).Children(func() {
			// Title & Subtitle
			mygoui.Column(c).FillWidth().Gap(2).Children(func() {
				titleEl := mygoui.Text(c, props.Name).Bold().FontSize(13).TextColor(t.Text).SingleLine()
				if titleEl.Clicked() && props.OnViewVideos != nil {
					props.OnViewVideos()
				}
				mygoui.Text(c, subtitle).TextColor(t.TextMuted).FontSize(10).SingleLine()
			})

			// Spacer
			mygoui.Column(c).Fill().Children(func() {})

			// Bottom row: date on left, pencil + trash icons on right
			mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
				mygoui.Row(c).AlignItems(mygoui.Center).Gap(4).Children(func() {
					mygoui.Icon(c, ui.IconCalendar).Size(9, 9).TextColor(t.TextMuted)
					mygoui.Text(c, dateStr).FontSize(9).TextColor(t.TextMuted)
				})

				mygoui.Row(c).Gap(4).AlignItems(mygoui.Center).Children(func() {
					if props.OnOpenFolder != nil {
						folderBtn := mygoui.Button(c.Key(fmt.Sprintf("card-folder-%d", props.ID)), "").
							Radius(6).Background(t.Surface)
						folderBtn.Children(func() {
							mygoui.Row(c).Fill().Center().Children(func() {
								mygoui.Icon(c, ui.IconPencil).Size(11, 11).TextColor(t.TextMuted)
							})
						})
						if folderBtn.Clicked() {
							props.OnOpenFolder()
						}
					}
					if props.OnDelete != nil {
						delBtn := mygoui.Button(c.Key(fmt.Sprintf("card-del-%d", props.ID)), "").
							Radius(6).Background(t.Surface)
						delBtn.Children(func() {
							mygoui.Row(c).Fill().Center().Children(func() {
								mygoui.Icon(c, ui.IconTrash).Size(11, 11).TextColor(mygoui.Hex("#ef4444"))
							})
						})
						if delBtn.Clicked() {
							props.OnDelete()
						}
					}
				})
			})
		})
	})
}
