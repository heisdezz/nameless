package pages

import (
	"fmt"

	"nameless/ui"

	mygoui "github.com/egoist/mygo/ui"
)

// RecentDownloadItem represents a recently completed video.
type RecentDownloadItem struct {
	ID    int64
	Title string
	Path  string
}

// HomeProps defines inputs and callbacks for the Home / Dashboard page.
type HomeProps struct {
	ProfileURL      *string
	Fetching        bool
	FetchErr        string
	DownloadedCount int
	ProfilesCount   int
	Workers         int
	EngineStatus    string
	YTDLPMissing    bool
	RecentDownloads []RecentDownloadItem
	OnFetch         func()
	OnDetect        func()
	OnInstallDocs   func()
	OnOpenFolder    func(path string)
}

// RenderHome renders the main dashboard page.
func RenderHome(c *mygoui.Context, props HomeProps) {
	t := c.Theme()
	mygoui.Scroll(c).Fill().Children(func() {
		mygoui.Column(c).Fill().Gap(16).Padding(24).Children(func() {
			// Missing yt-dlp Banner (plan §4, milestone 3 task E5)
			if props.YTDLPMissing {
				mygoui.Row(c).FillWidth().Padding(12).Gap(12).Radius(8).Background(mygoui.Color{R: 239, G: 68, B: 68, A: 25}).Border(1, mygoui.Color{R: 239, G: 68, B: 68, A: 120}).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
					mygoui.Row(c).Gap(10).AlignItems(mygoui.Center).Children(func() {
						mygoui.Icon(c, ui.IconAlert).Size(18, 18).TextColor(ui.ErrColor())
						mygoui.Column(c).Gap(2).Children(func() {
							mygoui.Text(c, "yt-dlp not found").Bold().FontSize(13).TextColor(ui.ErrColor())
							mygoui.Text(c, "yt-dlp not found — install it, then press Detect").FontSize(12).TextColor(t.TextMuted)
						})
					})
					mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
						if mygoui.Button(c.Key("detect-ytdlp-btn"), "Detect").Radius(6).Clicked() {
							if props.OnDetect != nil {
								props.OnDetect()
							}
						}
						if mygoui.Button(c.Key("install-ytdlp-btn"), "Install instructions").Radius(6).Clicked() {
							if props.OnInstallDocs != nil {
								props.OnInstallDocs()
							}
						}
					})
				})
			}

			// Top Search / Profile URL Bar
			mygoui.Row(c).FillWidth().Gap(10).AlignItems(mygoui.Center).Radius(8).Padding(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				mygoui.Icon(c, ui.IconSearch).Size(16, 16).TextColor(t.TextMuted)
				if props.ProfileURL != nil {
					mygoui.TextInput(c, props.ProfileURL).Placeholder("https://www.tiktok.com/@username").FillWidth().OnSubmit(func() {
						if props.OnFetch != nil {
							props.OnFetch()
						}
					})
				}
				fetchBtn := mygoui.PrimaryButton(c, "").Label("Fetch profile").Radius(6)
				fetchBtn.Children(func() {
					mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
						mygoui.Icon(c, ui.IconDownload).Size(14, 14)
						mygoui.Text(c, "Fetch profile").FontSize(13).Bold()
					})
				})
				if props.Fetching {
					fetchBtn.Disabled(true)
					// Show loading spinner instead of button label
					mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
						mygoui.Icon(c, ui.IconDownload).Size(14, 14)
						mygoui.Text(c, "Fetching...").FontSize(13).Bold()
					})
				} else if props.FetchErr != "" {
					fetchBtn.Disabled(true)
					// Show retry button on error
					fetchBtn = mygoui.Button(c.Key("retry-fetch-btn"), "").Label("Retry").Radius(6)
					fetchBtn.Children(func() {
						mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
							mygoui.Icon(c, ui.IconRefresh).Size(14, 14)
							mygoui.Text(c, "Retry").FontSize(13)
						})
					})
					if fetchBtn.Clicked() && props.OnFetch != nil {
						props.OnFetch()
					}
				} else if fetchBtn.Clicked() && props.OnFetch != nil {
					props.OnFetch()
				}
			})

			if props.FetchErr != "" && !props.Fetching {
				mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Children(func() {
					mygoui.Icon(c, ui.IconAlert).Size(14, 14).TextColor(ui.ErrColor())
					mygoui.Text(c, props.FetchErr).TextColor(ui.ErrColor()).FontSize(12)
				})
			}

			// Overview Stats Header
			mygoui.Row(c).AlignItems(mygoui.Center).Children(func() {
				mygoui.Text(c, "OVERVIEW STATS").TextColor(t.TextMuted).Bold().FontSize(11)
			})

			// 4 Metric Cards
			mygoui.Row(c).FillWidth().Gap(10).Children(func() {
				// Stat 1: Downloaded Videos
				mygoui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
						mygoui.Icon(c, ui.IconCheck).Size(14, 14).TextColor(t.TextMuted)
						mygoui.Text(c, "Total Downloaded").TextColor(t.TextMuted).FontSize(11)
					})
					mygoui.Text(c, fmt.Sprintf("%d Videos", props.DownloadedCount)).Bold().FontSize(18).TextColor(t.Text)
				})

				// Stat 2: Profiles Count
				mygoui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
						mygoui.Icon(c, ui.IconFolder).Size(14, 14).TextColor(t.TextMuted)
						mygoui.Text(c, "Profiles Archived").TextColor(t.TextMuted).FontSize(11)
					})
					mygoui.Text(c, fmt.Sprintf("%d Profiles", props.ProfilesCount)).Bold().FontSize(18).TextColor(t.Text)
				})

				// Stat 3: Workers
				workers := props.Workers
				if workers <= 0 {
					workers = 3
				}
				mygoui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
						mygoui.Icon(c, ui.IconWave).Size(14, 14).TextColor(t.TextMuted)
						mygoui.Text(c, "Concurrency").TextColor(t.TextMuted).FontSize(11)
					})
					mygoui.Text(c, fmt.Sprintf("%d Workers", workers)).Bold().FontSize(18).TextColor(t.Text)
				})

				// Stat 4: Engine Status
				version := props.EngineStatus
				if version == "" {
					version = "yt-dlp Ready"
				}
				mygoui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
						mygoui.Icon(c, ui.IconSettings).Size(14, 14).TextColor(t.TextMuted)
						mygoui.Text(c, "Engine").TextColor(t.TextMuted).FontSize(11)
					})
					mygoui.Text(c, version).Bold().FontSize(14).TextColor(t.Text)
				})
			})

			// Quick-Start Hero Card
			mygoui.Column(c).FillWidth().Padding(18).Gap(8).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				mygoui.Row(c).AlignItems(mygoui.Center).Children(func() {
					mygoui.Text(c, "● BULK PROFILE ARCHIVE").TextColor(t.TextMuted).Bold().FontSize(11)
				})
				mygoui.Text(c, "Archive TikTok Profiles Offline").FontSize(17).Bold()
				mygoui.Text(c, "Paste a creator URL above and click Fetch Profile. Download destination, workers, and cookies are configured in Settings.").TextColor(t.TextMuted).FontSize(13)
			})

			// Recent Downloads Section
			mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
				mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
					mygoui.Icon(c, ui.IconDownload).Size(14, 14).TextColor(t.TextMuted)
					mygoui.Text(c, "RECENT DOWNLOADS").TextColor(t.TextMuted).Bold().FontSize(11)
				})
				if len(props.RecentDownloads) > 0 {
					mygoui.Text(c, fmt.Sprintf("%d completed", len(props.RecentDownloads))).TextColor(t.TextMuted).FontSize(11)
				}
			})

			if len(props.RecentDownloads) == 0 {
				mygoui.Column(c).FillWidth().Padding(20).Gap(6).Radius(8).Background(t.Surface).Border(1, t.Border).Center().Children(func() {
					mygoui.Text(c, "No downloaded videos yet").FontSize(13).Bold().TextColor(t.TextMuted)
					mygoui.Text(c, "Videos finished downloading will appear here with quick access to their output folder.").FontSize(12).TextColor(t.TextMuted)
				})
			} else {
				mygoui.Column(c).FillWidth().Gap(6).Children(func() {
					for _, v := range props.RecentDownloads {
						vid := v
						mygoui.Row(c.Key(fmt.Sprintf("recent-%d", vid.ID))).FillWidth().Padding(10).Gap(10).Radius(6).Background(t.Surface).Border(1, t.Border).AlignItems(mygoui.Center).Children(func() {
							mygoui.Icon(c, ui.IconCheck).Size(15, 15).TextColor(t.Success)
							mygoui.Column(c).Fill().Gap(2).Children(func() {
								mygoui.Text(c, vid.Title).Bold().FontSize(13)
								sub := vid.Path
								if sub == "" {
									sub = "Completed"
								}
								mygoui.Text(c, sub).TextColor(t.TextMuted).FontSize(11)
							})
							if props.OnOpenFolder != nil {
								if mygoui.Button(c.Key(fmt.Sprintf("open-folder-%d", vid.ID)), "Open folder").Radius(6).Clicked() {
									props.OnOpenFolder(vid.Path)
								}
							}
						})
					}
				})
			}
		})
	})
}
