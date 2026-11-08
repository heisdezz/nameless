package main

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

func (a *app) viewMain(c *ui.Context) {
	t := c.Theme()
	profilesCount, downloadedCount, _ := a.overviewStats()

	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Fill().Gap(16).Padding(24).Children(func() {
			// Top Search / Profile URL Input Bar with Fetch action button
			ui.Row(c).FillWidth().Gap(10).AlignItems(ui.Center).Radius(8).Padding(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Icon(c, iconSearch).Size(16, 16).TextColor(t.TextMuted)
				ui.TextInput(c, &a.profileURL).Placeholder("https://www.tiktok.com/@username").FillWidth().OnSubmit(func() {
					a.fetchProfile()
				})
				fetchBtn := ui.PrimaryButton(c, "").Label("Fetch profile").Radius(6)
				fetchBtn.Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(2).Children(func() {
						ui.Icon(c, iconDownload).Size(14, 14)
						ui.Text(c, "Fetch profile").FontSize(13).Bold()
					})
				})
				if a.fetching {
					fetchBtn.Disabled(true)
				}
				if fetchBtn.Clicked() {
					a.fetchProfile()
				}
			})

			if a.fetchErr != "" {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, iconAlert).Size(14, 14).TextColor(errColor())
					ui.Text(c, a.fetchErr).TextColor(errColor()).FontSize(12)
				})
			}

			// Overview Stats Section Header
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "OVERVIEW STATS").TextColor(t.TextMuted).Bold().FontSize(11)
			})

			// 4 Stat Metric Cards (shadcn zinc cards with hairline border)
			ui.Row(c).FillWidth().Gap(10).Children(func() {
				// Stat 1: Downloaded Videos
				ui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Icon(c, iconCheck).Size(14, 14).TextColor(t.TextMuted)
						ui.Text(c, "Total Downloaded").TextColor(t.TextMuted).FontSize(11)
					})
					ui.Text(c, fmt.Sprintf("%d Videos", downloadedCount)).Bold().FontSize(18).TextColor(t.Text)
				})

				// Stat 2: Profiles Count
				ui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Icon(c, iconFolder).Size(14, 14).TextColor(t.TextMuted)
						ui.Text(c, "Profiles Archived").TextColor(t.TextMuted).FontSize(11)
					})
					ui.Text(c, fmt.Sprintf("%d Profiles", profilesCount)).Bold().FontSize(18).TextColor(t.Text)
				})

				// Stat 3: Workers / Concurrency
				workers := 3
				if a.cfg != nil && a.cfg.Workers > 0 {
					workers = a.cfg.Workers
				}
				ui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Icon(c, iconWave).Size(14, 14).TextColor(t.TextMuted)
						ui.Text(c, "Concurrency").TextColor(t.TextMuted).FontSize(11)
					})
					ui.Text(c, fmt.Sprintf("%d Workers", workers)).Bold().FontSize(18).TextColor(t.Text)
				})

				// Stat 4: Engine Status
				version := a.ytdlpVersion
				if version == "" {
					version = "yt-dlp Ready"
				}
				ui.Column(c).Fill().Padding(14).Gap(4).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						ui.Icon(c, iconSettings).Size(14, 14).TextColor(t.TextMuted)
						ui.Text(c, "Engine").TextColor(t.TextMuted).FontSize(11)
					})
					ui.Text(c, version).Bold().FontSize(14).TextColor(t.Text)
				})
			})

			// Quick-Start Hero Card
			ui.Column(c).FillWidth().Padding(18).Gap(8).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				ui.Row(c).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "● BULK PROFILE ARCHIVE").TextColor(t.TextMuted).Bold().FontSize(11)
				})
				ui.Text(c, "Archive TikTok Profiles Offline").FontSize(17).Bold()
				ui.Text(c, "Paste a creator URL above and click Fetch Profile. Download destination, workers, and cookies are configured in Settings.").TextColor(t.TextMuted).FontSize(13)
			})
		})
	})
}
