package pages

import (
	"fmt"

	"nameless/ui"

	mygoui "github.com/egoist/mygo/ui"
)

// SettingsProps defines inputs and bindings for the Settings page.
type SettingsProps struct {
	OutDir         *string
	NamingTemplate *string
	// WorkersF64 binds concurrent downloads (videos in flight at once,
	// default 3, max 5). ConnectionsF64 binds connections per single
	// download (default 5) — a different limit, not to be confused.
	WorkersF64           *float64
	ConnectionsF64       *float64
	DelayStr             *string
	TimeoutStr           *string
	VideoQuality         *string
	WriteThumbnail       *bool
	WriteInfoJSON        *bool
	WriteSubtitles       *bool
	Cookie               *string
	SessionID            *string
	CookiesFromBrowser   *string
	YTDLPPath            *string
	TikAPI               *string
	TikCookies           *string
	TikFormat            *string
	AccentHex            *string
	Dark                 *bool
	RadiusF64            *float64
	Spacing              string
	EngineStatus         string
	OnBack               func()
	OnDetect             func()
	OnSave               func()
	OnResetDefaults      func()
	OnSelectAccentPreset func(hex string)
	OnSelectSpacing      func(spacing string)
}

// RenderSettings renders the comprehensive, modern shadcn-styled settings page.
func RenderSettings(c *mygoui.Context, props SettingsProps) {
	t := c.Theme()
	mygoui.Scroll(c).Fill().Children(func() {
		mygoui.Column(c).Fill().Gap(18).Padding(24).Children(func() {
			// Top Navigation Bar
			mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
				mygoui.Row(c).Gap(10).AlignItems(mygoui.Center).Children(func() {
					if mygoui.Button(c.Key("settings-back-btn"), "← Back").Radius(6).Clicked() {
						if props.OnBack != nil {
							props.OnBack()
						}
					}
					mygoui.Column(c).Gap(1).Children(func() {
						mygoui.Text(c, "Settings").FontSize(22).Bold()
						mygoui.Text(c, "Configure download destinations, yt-dlp concurrency, authentication, and themes.").TextColor(t.TextMuted).FontSize(12)
					})
				})

				mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
					if props.OnResetDefaults != nil {
						if mygoui.Button(c.Key("reset-defaults-btn"), "Reset Defaults").Radius(6).Clicked() {
							props.OnResetDefaults()
						}
					}
					if props.OnSave != nil {
						saveBtn := mygoui.PrimaryButton(c.Key("save-settings-btn"), "").Label("Save Settings").Radius(6)
						saveBtn.Children(func() {
							mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
								mygoui.Icon(c, ui.IconCheck).Size(14, 14)
								mygoui.Text(c, "Save Settings").FontSize(13).Bold()
							})
						})
						if saveBtn.Clicked() {
							props.OnSave()
						}
					}
				})
			})

			// 1. Download & Storage Settings Card
			mygoui.Column(c).FillWidth().Padding(18).Gap(14).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				// Section header with accent bar
				mygoui.Row(c).Gap(8).Children(func() {
					mygoui.Box(c).Height(2).Background(t.Accent).FillWidth()
				})
				mygoui.Column(c).Gap(2).Children(func() {
					mygoui.Row(c).Gap(4).AlignItems(mygoui.Start).Children(func() {
						mygoui.Icon(c, ui.IconFolder).Size(16, 16).TextColor(t.Text)
						mygoui.Text(c, "Download & Engine Settings").Bold().FontSize(15)
					})
					mygoui.Text(c, "Storage & Naming").FontSize(11).TextColor(t.TextMuted)
				})
				mygoui.Text(c, "Configure where TikTok creator archives are saved, video naming formats, and additional assets to preserve.").TextColor(t.TextMuted).FontSize(12)

				// Default Download Location
				if props.OutDir != nil {
					mygoui.Column(c).Gap(6).FillWidth().Children(func() {
						mygoui.Text(c, "Default Download Location").FontSize(12).Bold()
						mygoui.TextInput(c, props.OutDir).Placeholder("./downloads").FillWidth()
						mygoui.Text(c, "Folder where profiles and videos will be saved on your local machine.").TextColor(t.TextMuted).FontSize(11)
					})
				}

				// Naming Template
				if props.NamingTemplate != nil {
					mygoui.Column(c).Gap(6).FillWidth().Children(func() {
						mygoui.Text(c, "Video Filename Template").FontSize(12).Bold()
						mygoui.TextInput(c, props.NamingTemplate).Placeholder("%(upload_date)s_%(id)s_%(title)s.%(ext)s").FillWidth()
						mygoui.Text(c, "Pattern for output filenames. Tags: %(id)s, %(title)s, %(upload_date)s, %(uploader)s, %(ext)s").TextColor(t.TextMuted).FontSize(11)
					})
				}

				// Video Quality
				if props.VideoQuality != nil {
					mygoui.Column(c).Gap(6).FillWidth().Children(func() {
						mygoui.Text(c, "Video Quality Preference").FontSize(12).Bold()
						mygoui.TextInput(c, props.VideoQuality).Placeholder("best").FillWidth()
						mygoui.Text(c, "Quality format selector (e.g. 'best', '1080p', '720p', or 'audio_only'). When TikDownloader Format is left at the default (mp4), this selects the download format.").TextColor(t.TextMuted).FontSize(11)
					})
				}

				mygoui.Box(c).Height(1).FillWidth().Background(t.Border)

				// Media Asset Toggles (Switches)
				mygoui.Column(c).Gap(12).FillWidth().Children(func() {
					mygoui.Text(c, "Additional Assets").FontSize(12).Bold()

					// Thumbnail Switch
					if props.WriteThumbnail != nil {
						mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
							mygoui.Column(c).Gap(2).Children(func() {
								mygoui.Text(c, "Save Video Thumbnails").FontSize(13)
								mygoui.Text(c, "Downloads cover thumbnails alongside each video file.").TextColor(t.TextMuted).FontSize(11)
							})
							mygoui.Switch(c, props.WriteThumbnail)
						})
					}

					// Metadata Info JSON Switch
					if props.WriteInfoJSON != nil {
						mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
							mygoui.Column(c).Gap(2).Children(func() {
								mygoui.Text(c, "Save Metadata Info JSON").FontSize(13)
								mygoui.Text(c, "Writes complete metadata (tags, sound, view count) to a companion .json file.").TextColor(t.TextMuted).FontSize(11)
							})
							mygoui.Switch(c, props.WriteInfoJSON)
						})
					}

					// Subtitles Switch
					if props.WriteSubtitles != nil {
						mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
							mygoui.Column(c).Gap(2).Children(func() {
								mygoui.Text(c, "Save Subtitles & Captions").FontSize(13)
								mygoui.Text(c, "Downloads creator captions and subtitles if available.").TextColor(t.TextMuted).FontSize(11)
							})
							mygoui.Switch(c, props.WriteSubtitles)
						})
					}
				})
			})

			// 2. Concurrency & Performance Settings Card
			mygoui.Column(c).FillWidth().Padding(18).Gap(14).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				// Section header with accent bar
				mygoui.Row(c).Gap(8).Children(func() {
					mygoui.Box(c).Height(2).Background(t.Accent).FillWidth()
				})
				mygoui.Column(c).Gap(2).Children(func() {
					mygoui.Row(c).Gap(4).AlignItems(mygoui.Start).Children(func() {
						mygoui.Icon(c, ui.IconWave).Size(16, 16).TextColor(t.Text)
						mygoui.Text(c, "Engine & Concurrency").Bold().FontSize(15)
					})
					mygoui.Text(c, "Engine & Concurrency").FontSize(11).TextColor(t.TextMuted)
				})
				mygoui.Text(c, "Tune worker parallelism and delay throttling to maximize throughput while avoiding rate limits.").TextColor(t.TextMuted).FontSize(12)

				// Concurrent Downloads Slider (videos in flight at once)
				if props.WorkersF64 != nil {
					wVal := int(*props.WorkersF64)
					if wVal < 1 {
						wVal = 1
					}
					if wVal > 5 {
						wVal = 5
					}
					mygoui.Column(c).Gap(8).FillWidth().Children(func() {
						mygoui.Row(c).FillWidth().Justify(mygoui.SpaceBetween).Children(func() {
							mygoui.Text(c, "Concurrent Downloads (Parallel Workers)").FontSize(12).Bold()
							mygoui.Text(c, fmt.Sprintf("%d simultaneous downloads", wVal)).FontSize(12).TextColor(t.Text)
						})
						mygoui.StepSlider(c, props.WorkersF64, 1, 5, 1).FillWidth()
						mygoui.Text(c, "How many videos download at once. Default 3, max 5. This is not connections per download.").TextColor(t.TextMuted).FontSize(11)
					})
				}

				// Connections Per Download Slider (per single file)
				if props.ConnectionsF64 != nil {
					cVal := int(*props.ConnectionsF64)
					if cVal < 1 {
						cVal = 1
					}
					if cVal > 16 {
						cVal = 16
					}
					mygoui.Column(c).Gap(8).FillWidth().Children(func() {
						mygoui.Row(c).FillWidth().Justify(mygoui.SpaceBetween).Children(func() {
							mygoui.Text(c, "Connections Per Download").FontSize(12).Bold()
							mygoui.Text(c, fmt.Sprintf("%d connections", cVal)).FontSize(12).TextColor(t.Text)
						})
						mygoui.StepSlider(c, props.ConnectionsF64, 1, 16, 1).FillWidth()
						mygoui.Text(c, "Parallel connections for one video file. Default 5. Separate from concurrent downloads above.").TextColor(t.TextMuted).FontSize(11)
					})
				}

				// Delay and Timeout Rows
				mygoui.Row(c).FillWidth().Gap(14).Children(func() {
					if props.DelayStr != nil {
						mygoui.Column(c).Gap(6).FillWidth().Children(func() {
							mygoui.Text(c, "Delay Between Downloads").FontSize(12).Bold()
							mygoui.TextInput(c, props.DelayStr).Placeholder("1s").FillWidth()
							mygoui.Text(c, "Cooldown between requests (e.g. '1s', '500ms').").TextColor(t.TextMuted).FontSize(11)
						})
					}
					if props.TimeoutStr != nil {
						mygoui.Column(c).Gap(6).FillWidth().Children(func() {
							mygoui.Text(c, "Per-Video Timeout").FontSize(12).Bold()
							mygoui.TextInput(c, props.TimeoutStr).Placeholder("5m").FillWidth()
							mygoui.Text(c, "Kill individual worker after timeout (e.g. '5m').").TextColor(t.TextMuted).FontSize(11)
						})
					}
				})

				// Custom yt-dlp binary path
				if props.YTDLPPath != nil {
					mygoui.Column(c).Gap(6).FillWidth().Children(func() {
						mygoui.Text(c, "Custom yt-dlp Executable Path").FontSize(12).Bold()
						mygoui.TextInput(c, props.YTDLPPath).Placeholder("/usr/bin/yt-dlp").FillWidth()
						mygoui.Text(c, "Leave blank to auto-detect yt-dlp from system PATH.").TextColor(t.TextMuted).FontSize(11)
					})
				}

				// Engine status + Detect (previously passed but never rendered)
				if props.EngineStatus != "" || props.OnDetect != nil {
					status := props.EngineStatus
					if status == "" {
						status = "yt-dlp status unknown — press Detect"
					}
					mygoui.Row(c).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
						mygoui.Text(c, status).FontSize(12).TextColor(t.TextMuted)
						if props.OnDetect != nil {
							if mygoui.Button(c.Key("detect-ytdlp-btn"), "Detect yt-dlp").Radius(6).Clicked() {
								props.OnDetect()
							}
						}
					})
				}

				mygoui.Box(c).Height(1).FillWidth().Background(t.Border)

				// TikDownloader engine (video files come from tikdownloader.io;
				// yt-dlp is only used for profile URLs/metadata + sidecars)
				mygoui.Column(c).Gap(6).FillWidth().Children(func() {
					mygoui.Text(c, "TikDownloader Engine").FontSize(12).Bold()
					mygoui.Text(c, "API used for the actual video file downloads.").TextColor(t.TextMuted).FontSize(11)
				})
				if props.TikAPI != nil {
					mygoui.Column(c).Gap(6).FillWidth().Children(func() {
						mygoui.Text(c, "TikDownloader API Endpoint").FontSize(12).Bold()
						mygoui.TextInput(c, props.TikAPI).Placeholder("https://tikdownloader.io").FillWidth()
					})
				}
				if props.TikCookies != nil {
					mygoui.Column(c).Gap(6).FillWidth().Children(func() {
						mygoui.Text(c, "TikDownloader API Cookies").FontSize(12).Bold()
						mygoui.TextInput(c, props.TikCookies).Placeholder("cf_clearance=...").FillWidth()
						mygoui.Text(c, "Optional cookies for the tikdownloader.io API (separate from TikTok cookies above).").TextColor(t.TextMuted).FontSize(11)
					})
				}
				if props.TikFormat != nil {
					mygoui.Column(c).Gap(6).FillWidth().Children(func() {
						mygoui.Text(c, "TikDownloader Format").FontSize(12).Bold()
						mygoui.TextInput(c, props.TikFormat).Placeholder("mp4").FillWidth()
						mygoui.Text(c, "Format selector for tikdownloader.io (mp4, hd, 720p, 1080p, best). Non-default values win over Video Quality above.").TextColor(t.TextMuted).FontSize(11)
					})
				}
			})

			// 3. Authentication & Cookies Card
			mygoui.Column(c).FillWidth().Padding(18).Gap(14).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				// Section header with accent bar
				mygoui.Row(c).Gap(8).Children(func() {
					mygoui.Box(c).Height(2).Background(t.Accent).FillWidth()
				})
				mygoui.Column(c).Gap(2).Children(func() {
					mygoui.Row(c).Gap(4).AlignItems(mygoui.Start).Children(func() {
						mygoui.Icon(c, ui.IconSettings).Size(16, 16).TextColor(t.Text)
						mygoui.Text(c, "TikTok Authentication & Cookies").Bold().FontSize(15)
					})
					mygoui.Text(c, "TikTok Authentication & Cookies").FontSize(11).TextColor(t.TextMuted)
				})
				mygoui.Text(c, "Authenticate requests to download private profiles, age-restricted videos, or solve CAPTCHA limits.").TextColor(t.TextMuted).FontSize(12)

				// Browser Cookie Source
				if props.CookiesFromBrowser != nil {
					mygoui.Column(c).Gap(8).FillWidth().Children(func() {
						mygoui.Text(c, "Extract Cookies from Browser").FontSize(12).Bold()
						mygoui.TextInput(c, props.CookiesFromBrowser).Placeholder("chrome, firefox, brave, edge").FillWidth()
						mygoui.Text(c, "Automatically read cookies from your web browser (e.g. 'chrome', 'firefox', 'brave', 'edge').").TextColor(t.TextMuted).FontSize(11)
					})
				}

				// Cookie File and Session ID
				mygoui.Row(c).Gap(14).Children(func() {
					if props.Cookie != nil {
						mygoui.Column(c).Gap(6).FillWidth().Children(func() {
							mygoui.Text(c, "Netscape Cookies File").FontSize(12).Bold()
							mygoui.TextInput(c, props.Cookie).Placeholder("path/to/cookies.txt").FillWidth()
							mygoui.Text(c, "Path to exported cookies.txt file.").TextColor(t.TextMuted).FontSize(11)
						})
					}
					if props.SessionID != nil {
						mygoui.Column(c).Gap(6).FillWidth().Children(func() {
							mygoui.Text(c, "Session ID Cookie").FontSize(12).Bold()
							mygoui.TextInput(c, props.SessionID).Placeholder("sessionid value...").FillWidth()
							mygoui.Text(c, "Direct value of the sessionid cookie.").TextColor(t.TextMuted).FontSize(11)
						})
					}
				})
			})

			// 4. Appearance & Theme Card
			mygoui.Column(c).FillWidth().Padding(18).Gap(14).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				// Section header with accent bar
				mygoui.Row(c).Gap(8).Children(func() {
					mygoui.Box(c).Height(2).Background(t.Accent).FillWidth()
				})
				mygoui.Column(c).Gap(2).Children(func() {
					mygoui.Row(c).Gap(4).AlignItems(mygoui.Start).Children(func() {
						mygoui.Icon(c, ui.IconSidebar).Size(16, 16).TextColor(t.Text)
						mygoui.Text(c, "Theme & Appearance").Bold().FontSize(15)
					})
					mygoui.Text(c, "Theme & Appearance").FontSize(11).TextColor(t.TextMuted)
				})
				mygoui.Text(c, "Customize application color accents, spacing density, and rounded corners.").TextColor(t.TextMuted).FontSize(12)

				// Accent Color & Presets
				if props.AccentHex != nil {
					mygoui.Column(c).Gap(8).FillWidth().Children(func() {
						mygoui.Text(c, "Accent Color").FontSize(12).Bold()
						mygoui.Row(c).Gap(12).AlignItems(mygoui.Center).Children(func() {
							mygoui.TextInput(c, props.AccentHex).Placeholder("#fafafa").Width(140)

							// Color Preset Pills
							presets := []struct {
								name string
								hex  string
							}{
								{"Zinc", "#fafafa"},
								{"Emerald", "#10b981"},
								{"Violet", "#8b5cf6"},
								{"Indigo", "#6366f1"},
								{"Rose", "#f43f5e"},
							}
							for _, p := range presets {
								hex := p.hex
								btn := mygoui.Button(c.Key("accent-"+p.name), p.name).Radius(6)
								if *props.AccentHex == hex {
									btn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
								} else {
									btn.Background(t.Background).TextColor(t.TextMuted)
								}
								if btn.Clicked() {
									*props.AccentHex = hex
									if props.OnSelectAccentPreset != nil {
										props.OnSelectAccentPreset(hex)
									}
								}
							}
						})
					})
				}

				// Corner Radius Slider
				if props.RadiusF64 != nil {
					rVal := int(*props.RadiusF64)
					mygoui.Column(c).Gap(8).FillWidth().Children(func() {
						mygoui.Row(c).FillWidth().Justify(mygoui.SpaceBetween).Children(func() {
							mygoui.Text(c, "Corner Radius").FontSize(12).Bold()
							mygoui.Text(c, fmt.Sprintf("%d px", rVal)).FontSize(12).TextColor(t.Text)
						})
						mygoui.StepSlider(c, props.RadiusF64, 4, 16, 2).FillWidth()
					})
				}

				// Spacing Density Pills
				mygoui.Column(c).Gap(8).FillWidth().Children(func() {
					mygoui.Text(c, "Interface Spacing Density").FontSize(12).Bold()
					mygoui.Row(c).Gap(10).AlignItems(mygoui.Center).Children(func() {
						for _, s := range []string{"Compact", "Default", "Roomy"} {
							spacingVal := s
							btn := mygoui.Button(c.Key("spacing-"+s), s).Radius(6)
							if props.Spacing == s {
								btn.Background(t.SurfaceHover).TextColor(t.Text).Bold()
							} else {
								btn.Background(t.Background).TextColor(t.TextMuted)
							}
							if btn.Clicked() && props.OnSelectSpacing != nil {
								props.OnSelectSpacing(spacingVal)
							}
						}
					})
				})

				// Dark Mode Toggle
				if props.Dark != nil {
					mygoui.Row(c).Gap(12).FillWidth().AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
						mygoui.Column(c).Gap(2).Children(func() {
							mygoui.Text(c, "Dark Mode").FontSize(13)
							mygoui.Text(c, "Enabled by default with authentic shadcn dark zinc palette.").TextColor(t.TextMuted).FontSize(11)
						})
						mygoui.Switch(c, props.Dark)
					})
				}
			})

			// Action Footer
			mygoui.Row(c).FillWidth().Justify(mygoui.SpaceBetween).AlignItems(mygoui.Center).Padding(10).Children(func() {
				if props.OnBack != nil {
					if mygoui.Button(c.Key("footer-back-btn"), "← Back").Radius(6).Clicked() {
						props.OnBack()
					}
				}
				mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
					if props.OnResetDefaults != nil {
						if mygoui.Button(c.Key("footer-reset-btn"), "Reset Defaults").Radius(6).Clicked() {
							props.OnResetDefaults()
						}
					}
					if props.OnSave != nil {
						if mygoui.PrimaryButton(c.Key("footer-save-btn"), "Save Settings").Radius(6).Clicked() {
							props.OnSave()
						}
					}
				})
			})
		})
	})
}
