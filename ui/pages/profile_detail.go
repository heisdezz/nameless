package pages

import (
	"fmt"

	"nameless/ui"

	mygoui "github.com/egoist/mygo/ui"
)

// VideoItem represents a video in the profile's video queue.
type VideoItem struct {
	ID     int64
	URL    string
	Title  string
	Status string // done/downloading/failed/pending/paused
	Path   string
	Pct    float64
}

// ProfileDetailProps defines inputs and callbacks for the Profile Detail screen.
type ProfileDetailProps struct {
	ProfileName    string
	ProfileURL     string
	Videos         []VideoItem
	SummaryText    string
	SummaryColor   mygoui.Color
	ShowOpenOutput bool
	Downloading    bool
	Paused         bool
	OverallPct     float64
	OnBack         func()
	OnDownloadAll  func()
	OnPause        func()
	OnResume       func()
	OnRetryFailed  func()
	OnOpenOutput   func()
	OnOpenVideoDir func(path string)
}

// RenderProfileDetail renders the detailed profile video queue and batch actions.
func RenderProfileDetail(c *mygoui.Context, props ProfileDetailProps) {
	t := c.Theme()
	mygoui.Scroll(c).Fill().Children(func() {
		mygoui.Column(c).Fill().Gap(14).Padding(24).Children(func() {
			// Profile Header Card
			mygoui.Column(c).FillWidth().Padding(18).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
					mygoui.Row(c).Gap(10).AlignItems(mygoui.Center).Children(func() {
						if mygoui.Button(c, "← All Profiles").Radius(6).Clicked() {
							if props.OnBack != nil {
								props.OnBack()
							}
						}
						mygoui.Icon(c, ui.IconFolder).Size(20, 20).TextColor(t.Text)
						mygoui.Text(c, props.ProfileName).FontSize(20).Bold()
					})

					if props.SummaryText != "" {
						mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
							mygoui.Text(c, props.SummaryText).TextColor(props.SummaryColor).Bold().FontSize(13)
							if props.ShowOpenOutput {
								openOutBtn := mygoui.Button(c, "").Label("Open output folder").Radius(6)
								openOutBtn.Children(func() {
									mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Children(func() {
										mygoui.Icon(c, ui.IconFolder).Size(14, 14)
										mygoui.Text(c, "Open output folder")
									})
								})
								if openOutBtn.Clicked() && props.OnOpenOutput != nil {
									props.OnOpenOutput()
								}
							}
						})
					}
				})

				// Overall progress bar (plan §4, milestone 3 task E6)
				totalVids := len(props.Videos)
				if totalVids > 0 {
					doneCount := 0
					for _, v := range props.Videos {
						if v.Status == "done" {
							doneCount++
						}
					}
					pct := float64(doneCount) / float64(totalVids)
					if props.OverallPct > 0 {
						pct = props.OverallPct
					}
					mygoui.Column(c).FillWidth().Gap(4).Children(func() {
						mygoui.Progress(c, pct).FillWidth().Height(4).Radius(2)
					})
				}

				// Batch Action Toolbar
				mygoui.Row(c).Gap(10).AlignItems(mygoui.Center).Children(func() {
					dlBtn := mygoui.PrimaryButton(c, "").Label("Download all").Radius(6)
					dlBtn.Children(func() {
						mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
							mygoui.Icon(c, ui.IconPlay).Size(14, 14)
							mygoui.Text(c, "Download all")
						})
					})
					if dlBtn.Clicked() && props.OnDownloadAll != nil {
						props.OnDownloadAll()
					}

					if props.Downloading {
						if props.Paused {
							resBtn := mygoui.Button(c, "").Label("Resume").Radius(6)
							resBtn.Children(func() {
								mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
									mygoui.Icon(c, ui.IconPlay).Size(14, 14)
									mygoui.Text(c, "Resume")
								})
							})
							if resBtn.Clicked() && props.OnResume != nil {
								props.OnResume()
							}
						} else {
							pauseBtn := mygoui.Button(c, "").Label("Pause").Radius(6)
							pauseBtn.Children(func() {
								mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
									mygoui.Icon(c, ui.IconPause).Size(14, 14)
									mygoui.Text(c, "Pause")
								})
							})
							if pauseBtn.Clicked() && props.OnPause != nil {
								props.OnPause()
							}
						}
					}

					retryBtn := mygoui.Button(c, "").Label("Retry failed").Radius(6)
					retryBtn.Children(func() {
						mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
							mygoui.Icon(c, ui.IconRefresh).Size(14, 14)
							mygoui.Text(c, "Retry failed")
						})
					})
					if retryBtn.Clicked() && props.OnRetryFailed != nil {
						props.OnRetryFailed()
					}
				})
			})

			// Empty Profile State (plan §4, milestone 3 task E6/E8)
			if len(props.Videos) == 0 {
				mygoui.Column(c).FillWidth().Padding(36).Gap(10).Radius(8).Background(t.Surface).Border(1, t.Border).Center().Children(func() {
					mygoui.Icon(c, ui.IconAlert).Size(24, 24).TextColor(t.TextMuted)
					mygoui.Text(c, "No videos found").Bold().FontSize(15)
					mygoui.Text(c, "No videos found — the account may be private; try cookies in Settings").TextColor(t.TextMuted).FontSize(13)
				})
			} else {
				// Videos Section Header
				mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
					mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
						mygoui.Icon(c, ui.IconWave).Size(16, 16).TextColor(t.Text)
						mygoui.Text(c, "Videos").Bold().FontSize(16)
					})
					mygoui.Text(c, fmt.Sprintf("%d items", len(props.Videos))).TextColor(t.TextMuted).FontSize(12)
				})

				// Video Items List
				for _, item := range props.Videos {
					v := item
					c.Key(fmt.Sprintf("vid-%d", v.ID))
					mygoui.Row(c).FillWidth().Padding(12).Gap(10).Radius(6).Background(t.Surface).Border(1, t.Border).AlignItems(mygoui.Center).Children(func() {
						switch v.Status {
						case "done":
							mygoui.Icon(c, ui.IconCheck).Size(16, 16).TextColor(t.Success)
						case "downloading":
							mygoui.Icon(c, ui.IconWave).Size(16, 16).TextColor(t.Accent)
						case "failed":
							mygoui.Icon(c, ui.IconAlert).Size(16, 16).TextColor(t.Danger)
						default:
							mygoui.Icon(c, ui.IconDownload).Size(16, 16).TextColor(t.TextMuted)
						}

						mygoui.Column(c).Fill().Gap(2).Children(func() {
							mygoui.Text(c, v.Title).FontSize(13).Bold()
							mygoui.Text(c, v.Status).TextColor(t.TextMuted).FontSize(11)
						})

						if v.Status == "done" {
							openBtn := mygoui.Button(c, "").Label("Open folder").Radius(6)
							openBtn.Children(func() {
								mygoui.Row(c).Gap(4).AlignItems(mygoui.Center).Children(func() {
									mygoui.Icon(c, ui.IconFolder).Size(13, 13)
									mygoui.Text(c, "Open folder")
								})
							})
							if openBtn.Clicked() && props.OnOpenVideoDir != nil {
								props.OnOpenVideoDir(v.Path)
							}
						}
					})
				}
			}
		})
	})
}
