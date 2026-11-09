package pages

import (
	"fmt"

	"nameless/ui"

	mygoui "github.com/egoist/mygo/ui"
)

// iconVolumeMuted shows a volume icon with an X while the player is muted.
var iconVolumeMuted = mygoui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"/><line x1="23" y1="9" x2="17" y2="15"/><line x1="17" y1="9" x2="23" y2="15"/></svg>`))

// iconBtn renders a small square control (30×28) with a centered icon
// glyph, as Box + Center: a Button with an empty label reserves a label
// slot next to its children and pushes the icon off-center.
func iconBtn(c *mygoui.Context, key string, icon *mygoui.SVG, iconSize float32, col mygoui.Color) mygoui.Element {
	btn := mygoui.Box(c.Key(key)).Size(30, 28).Radius(6).
		Background(mygoui.Hex("#1c1c24")).
		Center()
	btn.Children(func() {
		mygoui.Icon(c, icon).Size(iconSize, iconSize).TextColor(col)
	})
	return btn
}

// iconFullScreenExit is a "minimize" arrow pair, shown while the video
// fills the window.
var iconFullScreenExit = mygoui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 14 10 14 10 20"/><polyline points="20 10 14 10 14 4"/><line x1="14" y1="10" x2="21" y2="3"/><line x1="3" y1="21" x2="10" y2="14"/></svg>`))

// playerScreen builds the black cinema viewport box: either a fixed
// 360-high strip inside the normal page, or the whole window in
// fullscreen video mode.
func playerScreen(c *mygoui.Context, key string, fullscreen bool) mygoui.Element {
	if fullscreen {
		return mygoui.Box(c.Key(key)).Fill().
			Background(mygoui.Hex("#000000")).
			Clip()
	}
	return mygoui.Box(c.Key(key)).
		FillWidth().
		Height(360).
		Radius(10).
		Background(mygoui.Hex("#050508")).
		Border(1, mygoui.Hex("#27272a")).
		Clip()
}

// playerPicture renders what the screen shows: the live player when one
// exists, else the poster/thumbnail, else a film placeholder.
func playerPicture(c *mygoui.Context, props PlayerProps, fullscreen bool) {
	if props.Player != nil {
		v := mygoui.Video(c, props.Player)
		if fullscreen {
			v.Fill()
		} else {
			v.FillWidth().Height(360)
		}
		v.Fit(mygoui.Contain)
		return
	}
	if props.CurrentVideo.Thumbnail != nil {
		mygoui.Column(c).Fill().Center().Children(func() {
			img := mygoui.Image(c, props.CurrentVideo.Thumbnail)
			if fullscreen {
				img.Fill()
			} else {
				img.Height(360)
			}
			img.Clip()
		})
		return
	}
	mygoui.Column(c).Fill().Center().Background(mygoui.Hex("#0b0b10")).Children(func() {
		mygoui.Icon(c, ui.IconFilm).Size(54, 54).TextColor(mygoui.Hex("#3f3f46"))
	})
}

// playerCenterOverlay is the big translucent play/pause circle centered
// within the viewport.
func playerCenterOverlay(c *mygoui.Context, props PlayerProps, playing bool) {
	mygoui.Column(c).Absolute().Fill().Center().Children(func() {
		btn := mygoui.Box(c.Key("player-center-play")).
			Size(56, 56).
			Radius(28).
			Background(mygoui.Hex("#000000aa")).
			Border(1, mygoui.Hex("#ffffff44")).
			Center()

		btn.Children(func() {
			if playing {
				mygoui.Icon(c, ui.IconPause).Size(24, 24).TextColor(mygoui.Hex("#ffffff"))
			} else {
				mygoui.Icon(c, ui.IconPlay).Size(24, 24).TextColor(mygoui.Hex("#ffffff"))
			}
		})

		if btn.Clicked() && props.OnPlayPause != nil {
			props.OnPlayPause()
		}
	})
}

// playerControls builds the translucent bar overlaid on the bottom of the
// viewport: transport, time readouts, scrubber, mute and fullscreen
// toggles. In fullscreen the last button shows an exit-fullscreen glyph.
func playerControls(c *mygoui.Context, props PlayerProps, currentTime, totalTime string, playing, fullscreen bool) {
	mygoui.Row(c).Absolute().Bottom(0).Left(0).Right(0).
		Padding(10, 14).
		Gap(12).
		AlignItems(mygoui.Center).
		Background(mygoui.Hex("#09090bee")).
		Children(func() {
			// Play/Pause icon button
			ctlIcon := ui.IconPlay
			if playing {
				ctlIcon = ui.IconPause
			}
			playBtn := iconBtn(c, "ctrl-play-pause", ctlIcon, 13, mygoui.Hex("#ffffff"))
			if playBtn.Clicked() && props.OnPlayPause != nil {
				props.OnPlayPause()
			}

			// Current elapsed time
			mygoui.Text(c, currentTime).FontSize(11).Bold().TextColor(mygoui.Hex("#a1a1aa"))

			// Scrubber: seekable when a real player exists
			pct := props.ProgressPct
			if pct < 0 {
				pct = 0
			}
			if pct > 1 {
				pct = 1
			}
			if props.Player != nil && props.Seek != nil && props.OnSeek != nil {
				if total := props.Player.Duration(); total > 0 {
					seek := mygoui.Slider(c.Key("player-seek"), props.Seek, 0, total.Seconds()).
						FillWidth().Height(12)
					if seek.Changed() && props.OnSeek != nil {
						props.OnSeek(*props.Seek)
					}
				} else {
					mygoui.Progress(c, 0).Fill().Height(6).Radius(3)
				}
			} else {
				mygoui.Progress(c, pct).Fill().Height(6).Radius(3)
			}

			// Total duration
			mygoui.Text(c, totalTime).FontSize(11).Bold().TextColor(mygoui.Hex("#a1a1aa"))

			// Volume: mute toggle when a real player exists
			if props.Player != nil {
				muted := props.Player.Muted()
				icon := ui.IconVolume
				col := mygoui.Hex("#71717a")
				if muted {
					icon = iconVolumeMuted
					col = mygoui.Hex("#f4f4f5")
				}
				muteBtn := iconBtn(c, "ctrl-mute", icon, 15, col)
				if muteBtn.Clicked() {
					props.Player.SetMuted(!muted)
				}
			} else {
				mygoui.Icon(c, ui.IconVolume).Size(15, 15).TextColor(mygoui.Hex("#71717a"))
			}

			// Enter/exit fullscreen video
			fullIcon, fullKey := ui.IconMaximize, "ctrl-fullscreen"
			if fullscreen {
				fullIcon, fullKey = iconFullScreenExit, "ctrl-exit-fullscreen"
			}
			fullBtn := iconBtn(c, fullKey, fullIcon, 13, mygoui.Hex("#ffffff"))
			if fullBtn.Clicked() && props.OnFullScreen != nil {
				props.OnFullScreen()
			}
		})
}

// PlayerProps defines inputs and callbacks for the inner video player page.
type PlayerProps struct {
	ProfileName  string
	CurrentVideo VideoItem
	NextVideos   []VideoItem
	RandomVideos []VideoItem
	// Player is the in-app system video player. When non-nil the viewport
	// renders it (real decoded frames with sound) and playback controls
	// drive it directly; when nil the viewport falls back to the poster.
	Player *mygoui.VideoPlayer
	// FullScreen collapses the page into a viewport that fills the
	// window (fullscreen video mode); the window itself goes fullscreen.
	FullScreen bool
	// Seek is the slider binding (seconds) for the playhead, and OnSeek
	// reports a user scrub; both nil when there is no in-app player.
	Seek          *float64
	OnSeek        func(seconds float64)
	IsPlaying     bool
	ProgressPct   float64
	CurrentTime   string
	TotalTime     string
	OnBack        func()
	OnPlayPause   func()
	OnFullScreen  func()
	OnOpenPlayer  func()
	OnOpenFolder  func(path string)
	OnSelectVideo func(v VideoItem)
	OnShuffle     func()
}

// RenderPlayer renders the inner video player screen with the updated layout:
// - Top Navigation Bar
// - Player viewport (fixed height) + Next 2-3 videos (same fixed height) beside
// - Video details below player
// - More From Profile (Random Order) at bottom
func RenderPlayer(c *mygoui.Context, props PlayerProps) {
	t := c.Theme()

	title := props.CurrentVideo.Title
	if title == "" {
		title = fmt.Sprintf("Video #%d", props.CurrentVideo.ID)
	}

	currentTime := props.CurrentTime
	if currentTime == "" {
		currentTime = "00:00"
	}
	totalTime := props.TotalTime
	if totalTime == "" {
		if props.CurrentVideo.Duration != "" {
			totalTime = props.CurrentVideo.Duration
		} else {
			totalTime = "--:--"
		}
	}

	// Fullscreen video mode: only the viewport, filling the window, with
	// the transport bar overlaid — no page chrome at all.
	if props.FullScreen {
		playing := props.IsPlaying
		if props.Player != nil {
			playing = props.Player.Playing()
		}
		screen := playerScreen(c, "player-viewport-fs", true)
		screen.Children(func() {
			playerPicture(c, props, true)
			//	Title overlay, top-left
			mygoui.Row(c).Absolute().Top(0).Left(0).Right(0).Padding(10, 14).Children(func() {
				mygoui.Text(c, title).FontSize(13).Bold().TextColor(mygoui.Hex("#ffffff"))
			})
			playerCenterOverlay(c, props, playing)
			playerControls(c, props, currentTime, totalTime, playing, true)
		})
		return
	}

	mygoui.Scroll(c).Fill().Children(func() {
		mygoui.Column(c).Fill().Gap(18).Padding(24).Children(func() {
			// Top Navigation Bar
			mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
				mygoui.Row(c).Gap(10).AlignItems(mygoui.Center).Children(func() {
					if mygoui.Button(c, "← Back to Profile").Radius(6).Clicked() {
						if props.OnBack != nil {
							props.OnBack()
						}
					}
					mygoui.Icon(c, ui.IconFolder).Size(16, 16).TextColor(t.TextMuted)
					mygoui.Text(c, props.ProfileName).Bold().FontSize(15).TextColor(t.Text)
					mygoui.Text(c, "•").TextColor(t.TextMuted)
					mygoui.Text(c, "Player").Bold().FontSize(14).TextColor(t.Accent)
				})

				mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
					openExtBtn := mygoui.PrimaryButton(c.Key("player-open-ext"), "").
						Label("Open in System Player").
						Radius(6)
					openExtBtn.Children(func() {
						mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
							mygoui.Icon(c, ui.IconExternalLink).Size(14, 14)
							mygoui.Text(c, "Play in System Player").FontSize(12).Bold()
						})
					})
					if openExtBtn.Clicked() && props.OnOpenPlayer != nil {
						props.OnOpenPlayer()
					}

					if props.CurrentVideo.Path != "" {
						openFoldBtn := mygoui.Button(c.Key("player-open-folder"), "").
							Label("Open Folder").
							Radius(6)
						openFoldBtn.Children(func() {
							mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
								mygoui.Icon(c, ui.IconFolder).Size(14, 14)
								mygoui.Text(c, "Open Folder").FontSize(12)
							})
						})
						if openFoldBtn.Clicked() && props.OnOpenFolder != nil {
							props.OnOpenFolder(props.CurrentVideo.Path)
						}
					}
				})
			})

			// Player viewport with fixed height, beside Next 2-3 videos with same height
			mygoui.Row(c).FillWidth().Gap(16).Children(func() {
				// LEFT: Main Video Player Box with fixed height
				mygoui.Column(c).Fill().Gap(14).Radius(12).Background(t.Surface).Border(1, t.Border).Padding(16).Clip().Children(func() {
					// Cinema Black Viewport - fixed height 360
					screenBox := playerScreen(c, "player-viewport", false)

					// Playing state; with a real player always reflect it.
					playing := props.IsPlaying
					if props.Player != nil {
						playing = props.Player.Playing()
					}

					screenBox.Children(func() {
						// Live playback / poster / placeholder
						playerPicture(c, props, false)
						// Center Big Play/Pause Button
						playerCenterOverlay(c, props, playing)
						// Bottom Player Controls Bar
						playerControls(c, props, currentTime, totalTime, playing, false)
					})
				})

				// RIGHT: Next 2-3 Videos in Profile - same fixed height as player
				mygoui.Column(c).Width(320).Height(360).Gap(12).Radius(12).Background(t.Surface).Border(1, t.Border).Padding(16).Children(func() {
					mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
						mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Children(func() {
							mygoui.Icon(c, ui.IconFilm).Size(15, 15).TextColor(t.Accent)
							mygoui.Text(c, "Up Next").Bold().FontSize(15)
						})
						mygoui.Text(c, "Next 2-3 in profile").FontSize(11).TextColor(t.TextMuted)
					})

					if len(props.NextVideos) == 0 {
						mygoui.Column(c).FillWidth().Padding(24).Center().Children(func() {
							mygoui.Text(c, "No more videos in profile").TextColor(t.TextMuted).FontSize(12)
						})
					} else {
						for i, item := range props.NextVideos {
							nextV := item
							idx := i
							itemBox := mygoui.Row(c.Key(fmt.Sprintf("next-vid-%d", nextV.ID))).
								FillWidth().
								Padding(8).
								Gap(10).
								Radius(8).
								Background(t.Background).
								Border(1, t.Border).
								AlignItems(mygoui.Center).Grow(1)

							itemBox.Children(func() {
								// Mini Thumbnail
								thumbBox := mygoui.Box(c).Size(56, 56).Radius(6).Background(mygoui.Hex("#18181f")).Clip().AlignItems(mygoui.Center).JustifyItems(mygoui.Center)
								thumbBox.Children(func() {
									if nextV.Thumbnail != nil {
										mygoui.Image(c, nextV.Thumbnail).Size(56, 56).Clip()
									} else {
										mygoui.Column(c).Fill().Center().Children(func() {
											mygoui.Icon(c, ui.IconFilm).Size(18, 18).TextColor(mygoui.Hex("#52525b"))
										})
									}
									mygoui.Row(c).Absolute().Left(3).Top(3).Padding(1, 4).Radius(4).
										Background(mygoui.Hex("#09090bcc")).Children(func() {
										mygoui.Text(c, fmt.Sprintf("%d", idx+1)).FontSize(9).Bold().TextColor(mygoui.Hex("#f4f4f5"))
									})
								})

								// Info
								mygoui.Column(c).Fill().Gap(2).Children(func() {
									nextTitle := nextV.Title
									if nextTitle == "" {
										nextTitle = fmt.Sprintf("Video #%d", nextV.ID)
									}
									mygoui.Text(c, nextTitle).Bold().FontSize(12).TextColor(t.Text).MaxLines(2)
									mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Justify(mygoui.Center).Children(func() {
										if nextV.Duration != "" {
											mygoui.Text(c, nextV.Duration).FontSize(10).TextColor(t.TextMuted)
										}
										mygoui.Text(c, nextV.Status).FontSize(10).TextColor(t.TextMuted)
									})
								})

								// Play Icon button
								// mygoui.Box(c).Size(28, 28).Radius(14).Background(mygoui.Hex("#27272a")).Center().Children(func() {
								// 	mygoui.Icon(c, ui.IconPlay).Size(11, 11).TextColor(mygoui.Hex("#ffffff"))
								// })
							})

							if itemBox.Clicked() && props.OnSelectVideo != nil {
								props.OnSelectVideo(nextV)
							}
						}
					}
				})
			}) // end row: player viewport + up-next

			// Video Details below player (moved outside the player column)
			mygoui.Column(c).FillWidth().Gap(14).Radius(12).Background(t.Surface).Border(1, t.Border).Padding(18).Children(func() {
				mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
					mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
						mygoui.Icon(c, ui.IconFilm).Size(16, 16).TextColor(t.Accent)
						mygoui.Text(c, "Details").Bold().FontSize(15)
					})
				})

				mygoui.Column(c).Gap(4).Children(func() {
					mygoui.Text(c, title).Bold().FontSize(16).TextColor(t.Text)
					mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
						switch props.CurrentVideo.Status {
						case "done":
							mygoui.Icon(c, ui.IconCheck).Size(12, 12).TextColor(t.Success)
							mygoui.Text(c, "Downloaded").FontSize(12).TextColor(t.Success)
						case "downloading":
							mygoui.Icon(c, ui.IconWave).Size(12, 12).TextColor(t.Accent)
							mygoui.Text(c, "Downloading...").FontSize(12).TextColor(t.Accent)
						case "failed":
							mygoui.Icon(c, ui.IconAlert).Size(12, 12).TextColor(t.Danger)
							mygoui.Text(c, "Failed").FontSize(12).TextColor(t.Danger)
						default:
							mygoui.Text(c, "Pending").FontSize(12).TextColor(t.TextMuted)
						}

						if totalTime != "--:--" {
							mygoui.Text(c, "•").TextColor(t.TextMuted)
							mygoui.Text(c, totalTime).FontSize(12).TextColor(t.TextMuted)
						}
					})

					// Playback engine errors surface here (bad file,
					// missing decoder, broken stream).
					if props.Player != nil {
						if err := props.Player.Err(); err != nil {
							mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(6, 8).
								Radius(6).Background(mygoui.Hex("#fbbf2422")).Children(func() {
								mygoui.Icon(c, ui.IconAlert).Size(12, 12).TextColor(t.Danger)
								mygoui.Text(c, "Playback error: "+err.Error()).FontSize(11).TextColor(t.Danger)
							})
						}
					}
				})
			})

			// Bottom Section: List of Random Order of Videos in Profile
			mygoui.Column(c).FillWidth().Gap(14).Radius(12).Background(t.Surface).Border(1, t.Border).Padding(18).Children(func() {
				mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
					mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
						mygoui.Icon(c, ui.IconShuffle).Size(16, 16).TextColor(t.Accent)
						mygoui.Text(c, "More From Profile (Random Order)").Bold().FontSize(15)
					})

					if props.OnShuffle != nil {
						shufBtn := mygoui.Button(c.Key("player-shuffle-btn"), "").
							Label("Shuffle").
							Radius(6)
						shufBtn.Children(func() {
							mygoui.Row(c).Gap(4).AlignItems(mygoui.Center).Padding(2).Children(func() {
								mygoui.Icon(c, ui.IconShuffle).Size(12, 12)
								mygoui.Text(c, "Shuffle").FontSize(11)
							})
						})
						if shufBtn.Clicked() {
							props.OnShuffle()
						}
					}
				})

				if len(props.RandomVideos) == 0 {
					mygoui.Column(c).FillWidth().Padding(20).Center().Children(func() {
						mygoui.Text(c, "No other videos in profile").TextColor(t.TextMuted).FontSize(12)
					})
				} else {
					// Grid of random cards (min 3, max 6 columns)
					randCols := 4
					if len(props.RandomVideos) < 4 {
						randCols = len(props.RandomVideos)
					}
					mygoui.Grid(c).Columns(randCols).Gap(12).FillWidth().Children(func() {
						for i, item := range props.RandomVideos {
							rv := item
							idx := i
							c.Key(fmt.Sprintf("rand-vid-%d", rv.ID))
							ui.RenderVideoCard(c, ui.VideoCardProps{
								ID:        rv.ID,
								Index:     idx,
								Title:     rv.Title,
								URL:       rv.URL,
								Status:    rv.Status,
								Path:      rv.Path,
								Duration:  rv.Duration,
								Pct:       rv.Pct,
								Thumbnail: rv.Thumbnail,
								OnWatch: func() {
									if props.OnSelectVideo != nil {
										props.OnSelectVideo(rv)
									}
								},
								OnOpenFolder: func() {
									if props.OnOpenFolder != nil {
										props.OnOpenFolder(rv.Path)
									}
								},
							})
						}
					})
				}
			})
		})
	})
}
