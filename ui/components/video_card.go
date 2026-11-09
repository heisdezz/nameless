package components

import (
	"fmt"

	"nameless/ui"

	mygoui "github.com/egoist/mygo/ui"
)

// VideoCardProps defines inputs and callbacks for a single video card in the profile detail grid.
type VideoCardProps struct {
	ID           int64
	Index        int
	Title        string
	URL          string
	Status       string
	Path         string
	Duration     string
	Pct          float64
	Thumbnail    *mygoui.Bitmap
	OnWatch      func()
	OnOpenFolder func()
	OnDownload   func()
}

// RenderVideoCard renders a poster-style media card for a video in the profile grid,
// styled with rounded corners, dark cinema theme, top badges, and watch actions.
func RenderVideoCard(c *mygoui.Context, props VideoCardProps) {
	t := c.Theme()

	title := props.Title
	if title == "" {
		title = fmt.Sprintf("Video #%d", props.Index+1)
	}

	mygoui.Column(c.Key(fmt.Sprintf("vcard-%d", props.ID))).
		Fill().
		Radius(12).
		Background(t.Surface).
		Border(1, t.Border).
		Clip().AspectRatio(1).
		Children(func() {
			// 1. Cover / Poster Thumbnail
			coverBox := mygoui.Box(c.Key(fmt.Sprintf("vcard-cover-%d", props.ID))).
				FillWidth().
				Height(180).
				Background(mygoui.Hex("#131318")).
				Clip()

			coverBox.Children(func() {
				if props.Thumbnail != nil {
					mygoui.Image(c, props.Thumbnail).FillWidth().Height(180).Clip()
				} else {
					// Sleek fallback cover placeholder
					mygoui.Column(c).Fill().Center().Background(mygoui.Hex("#18181f")).Children(func() {
						mygoui.Icon(c, ui.IconFilm).Size(36, 36).TextColor(mygoui.Hex("#3f3f46"))
					})
				}

				// Top-Left Badge: Index #
				mygoui.Row(c).Absolute().Left(8).Top(8).Padding(2, 6).Gap(4).AlignItems(mygoui.Center).
					Radius(6).Background(mygoui.Hex("#09090bcc")).Border(1, mygoui.Hex("#27272a")).Children(func() {
					mygoui.Icon(c, ui.IconFilm).Size(10, 10).TextColor(mygoui.Hex("#a1a1aa"))
					mygoui.Text(c, fmt.Sprintf("#%d", props.Index+1)).FontSize(10).Bold().TextColor(mygoui.Hex("#f4f4f5"))
				})

				// Top-Right Badge: Duration or Status (matching reference card pill style)
				mygoui.Row(c).Absolute().Right(8).Top(8).Padding(2, 7).Gap(4).AlignItems(mygoui.Center).
					Radius(6).Background(mygoui.Hex("#09090bcc")).Border(1, mygoui.Hex("#27272a")).Children(func() {
					if props.Duration != "" {
						mygoui.Text(c, props.Duration).FontSize(10).Bold().TextColor(mygoui.Hex("#ffffff"))
					} else {
						switch props.Status {
						case "done":
							mygoui.Icon(c, ui.IconCheck).Size(10, 10).TextColor(t.Success)
							mygoui.Text(c, "Done").FontSize(10).Bold().TextColor(t.Success)
						case "downloading":
							mygoui.Icon(c, ui.IconWave).Size(10, 10).TextColor(t.Accent)
							mygoui.Text(c, "Downloading").FontSize(10).Bold().TextColor(t.Accent)
						case "failed":
							mygoui.Icon(c, ui.IconAlert).Size(10, 10).TextColor(t.Danger)
							mygoui.Text(c, "Failed").FontSize(10).Bold().TextColor(t.Danger)
						default:
							mygoui.Text(c, "Queued").FontSize(10).TextColor(t.TextMuted)
						}
					}
				})

				// Center Play Icon Overlay (invites watching)
				mygoui.Column(c).Absolute().Fill().Center().Children(func() {
					mygoui.Box(c).Size(40, 40).Radius(20).Background(mygoui.Hex("#00000088")).
						Border(1, mygoui.Hex("#ffffff33")).Center().Children(func() {
						mygoui.Icon(c, ui.IconPlay).Size(16, 16).TextColor(mygoui.Hex("#ffffff"))
					})
				})

				// Bottom Progress Bar if downloading
				if props.Status == "downloading" {
					pct := props.Pct
					if pct <= 0 {
						pct = 0.05
					}
					mygoui.Box(c).Absolute().Bottom(0).Left(0).Right(0).Height(4).Background(mygoui.Hex("#27272a")).Children(func() {
						mygoui.Progress(c, pct).FillWidth().Height(4)
					})
				}
			})

			if coverBox.Clicked() && props.OnWatch != nil {
				props.OnWatch()
			}

			// 2. Metadata / Information Area
			mygoui.Column(c).Absolute().Bottom(0).Background(t.Background).FillWidth().Padding(10, 12).Gap(6).Children(func() {
				// Title (single line with ellipsis)
				titleEl := mygoui.Text(c, title).Bold().FontSize(13).TextColor(t.Text).SingleLine()
				if titleEl.Clicked() && props.OnWatch != nil {
					props.OnWatch()
				}

				// Subtitle: Status & Details
				mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
					mygoui.Row(c).AlignItems(mygoui.Center).Gap(5).Children(func() {
						switch props.Status {
						case "done":
							mygoui.Icon(c, ui.IconCheck).Size(11, 11).TextColor(t.Success)
							mygoui.Text(c, "Downloaded").FontSize(11).TextColor(t.Success)
						case "downloading":
							mygoui.Icon(c, ui.IconWave).Size(11, 11).TextColor(t.Accent)
							mygoui.Text(c, "Downloading...").FontSize(11).TextColor(t.Accent)
						case "failed":
							mygoui.Icon(c, ui.IconAlert).Size(11, 11).TextColor(t.Danger)
							mygoui.Text(c, "Failed").FontSize(11).TextColor(t.Danger)
						default:
							mygoui.Icon(c, ui.IconDownload).Size(11, 11).TextColor(t.TextMuted)
							mygoui.Text(c, "Pending").FontSize(11).TextColor(t.TextMuted)
						}
					})

					if props.Duration != "" {
						mygoui.Text(c, props.Duration).FontSize(11).TextColor(t.TextMuted)
					}
				})

				// 3. Actions Row: Open folder when downloaded,
				// otherwise Download (Retry when failed).
				mygoui.Row(c).FillWidth().Gap(6).AlignItems(mygoui.Center).Children(func() {
					if props.Status != "done" && props.OnDownload != nil {
						label := "Download"
						switch props.Status {
						case "failed":
							label = "Retry"
						case "downloading":
							label = "Downloading..."
						}
						dlBtn := mygoui.PrimaryButton(c.Key(fmt.Sprintf("vcard-download-%d", props.ID)), "").
							Label(label).
							Height(28).
							Radius(6).Grow(1)
						dlBtn.Children(func() {
							mygoui.Row(c).Fill().Center().Gap(4).Padding(2).Children(func() {
								mygoui.Icon(c, ui.IconDownload).Size(12, 12)
								mygoui.Text(c, label).FontSize(11)
							})
						})
						if props.Status == "downloading" {
							dlBtn.Disabled(true)
						}
						if dlBtn.Clicked() && props.Status != "downloading" {
							props.OnDownload()
						}
						return
					}
					if props.Status == "done" && props.OnOpenFolder != nil {
						folderBtn := mygoui.Button(c.Key(fmt.Sprintf("vcard-folder-%d", props.ID)), "").
							Label("Open folder").
							Height(28).
							Radius(6).Grow(1)

						folderBtn.Children(func() {
							mygoui.Row(c).Fill().Center().Gap(4).Padding(2).Children(func() {
								mygoui.Icon(c, ui.IconFolder).Size(12, 12).TextColor(t.Text)
								mygoui.Text(c, "Open folder").FontSize(11)
							})
						})

						if folderBtn.Clicked() {
							props.OnOpenFolder()
						}
					}
				})
			})
		})
}
