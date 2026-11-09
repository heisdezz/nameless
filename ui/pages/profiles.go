package pages

import (
	"fmt"
	"strconv"
	"time"

	"nameless/ui"
	"nameless/ui/components"

	mygoui "github.com/egoist/mygo/ui"
)

// ProfileItem represents a profile in the directory.
type ProfileItem struct {
	ID          int64
	Name        string
	URL         string
	AvatarPath  string
	AvatarURL   string
	Avatar      *mygoui.Bitmap
	VideosCount int
	DateStr     string
}

// ProfilesProps defines inputs and callbacks for the Profiles overview page.
type ProfilesProps struct {
	Profiles         []ProfileItem
	GridCols         *string
	Loading          bool
	Page             int
	PageSize         int
	OnPageChange     func(page int)
	OnPageSizeChange func(size int)
	OnNew            func()
	OnViewVideos     func(p ProfileItem)
	OnOpenFolder     func(p ProfileItem)
	OnDelete         func(p ProfileItem)
}

// RenderProfiles renders the dedicated Profiles directory page with a grid of profile cards.
func RenderProfiles(c *mygoui.Context, props ProfilesProps) {
	t := c.Theme()

	cols := 5
	if props.GridCols != nil && *props.GridCols != "" {
		if val, err := strconv.Atoi(*props.GridCols); err == nil && val >= 3 && val <= 7 {
			cols = val
		}
	}

	mygoui.Scroll(c).Fill().Children(func() {
		mygoui.Column(c).Fill().Gap(20).Padding(24).Children(func() {
			// Page Header
			mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
				mygoui.Column(c).Gap(2).Children(func() {
					mygoui.Text(c, "Archived Profiles").FontSize(20).Bold()
					mygoui.Text(c, fmt.Sprintf("%d creator profiles saved", len(props.Profiles))).TextColor(t.TextMuted).FontSize(13)
				})

				mygoui.Row(c).AlignItems(mygoui.Center).Gap(10).Children(func() {
					if props.GridCols != nil {
						mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
							mygoui.Text(c, "Columns:").FontSize(12).TextColor(t.TextMuted)
							mygoui.Select(c.Key("grid-cols-select"), props.GridCols, []string{"3", "4", "5", "6", "7"}).Width(60)
						})
					}
					mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
						mygoui.Text(c, "Per page:").FontSize(12).TextColor(t.TextMuted)
						pageSize := props.PageSize
						if pageSize <= 0 {
							pageSize = 20
						}
						pageSizeStr := strconv.Itoa(pageSize)
						sel := mygoui.Select(c.Key("profiles-page-size-select"), &pageSizeStr, []string{"10", "20", "50", "100"}).Width(60)
						if sel.Changed() && props.OnPageSizeChange != nil {
							if n, err := strconv.Atoi(pageSizeStr); err == nil && n > 0 {
								props.OnPageSizeChange(n)
							}
						}
					})
					newBtn := mygoui.PrimaryButton(c.Key("new-profile-btn"), "").Label("+ New Download").Radius(6)
					newBtn.Children(func() {
						mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(2).Children(func() {
							mygoui.Icon(c, ui.IconPlus).Size(14, 14)
							mygoui.Text(c, "+ New Download").FontSize(13).Bold()
						})
					})
					if newBtn.Clicked() && props.OnNew != nil {
						props.OnNew()
					}
				})
			})

			// Loading State (shown while the list refreshes off-thread)
			if props.Loading {
				mygoui.Box(c.Key("profiles-loading")).
					FillWidth().Height(400).
					Children(func() {
						components.RenderLoader(c, components.LoaderProps{
							Size:        48,
							Text:        "Loading profiles...",
							ShowOverlay: true,
							Color:       t.Accent,
						})
					})
				return
			}

			// Empty State
			if len(props.Profiles) == 0 {
				mygoui.Column(c).FillWidth().Padding(32).Gap(12).Radius(8).Background(t.Surface).Border(1, t.Border).Center().Children(func() {
					mygoui.Text(c, "No profiles archived yet").Bold().FontSize(15)
					mygoui.Text(c, "Paste a TikTok creator profile URL on the Home screen to fetch and download videos.").TextColor(t.TextMuted).FontSize(13)
					if mygoui.PrimaryButton(c.Key("empty-fetch-btn"), "Fetch your first profile").Radius(6).Clicked() {
						if props.OnNew != nil {
							props.OnNew()
						}
					}
				})
				return
			}

			// Paginate so only one page of cards is laid out per frame.
			pageSize := props.PageSize
			if pageSize <= 0 {
				pageSize = 20
			}
			page := props.Page
			if page < 1 {
				page = 1
			}
			totalPages := (len(props.Profiles) + pageSize - 1) / pageSize
			if totalPages == 0 {
				totalPages = 1
			}
			if page > totalPages {
				page = totalPages
			}
			start := (page - 1) * pageSize
			end := start + pageSize
			if end > len(props.Profiles) {
				end = len(props.Profiles)
			}
			pageProfiles := props.Profiles[start:end]

			// Profiles Cards Grid (default 5 columns, min 3 max 7) with
			// the same sweep reveal as the video grid: stagger per index.
			mygoui.Grid(c).Columns(cols).Gap(14).FillWidth().Children(func() {
				for idx, item := range pageProfiles {
					p := item
					c.Key(fmt.Sprintf("prof-card-%d", p.ID))
					cardBox := mygoui.Box(c).FillWidth()
					delay := time.Duration(idx) * 25 * time.Millisecond
					if delay > 600*time.Millisecond {
						delay = 600 * time.Millisecond
					}
					cardBox = cardBox.Transition(mygoui.ElementTransition{
						Animation: &mygoui.Animation{
							Duration: 350 * time.Millisecond,
							Ease:     mygoui.EaseOut,
							Delay:    delay,
						},
						Enter: &mygoui.Motion{Opacity: 0, X: -16, Y: 10, Scale: 0.96},
					})
					cardBox.Children(func() {
						components.RenderProfileCard(c, components.ProfileCardProps{
							ID:          p.ID,
							Name:        p.Name,
							URL:         p.URL,
							AvatarPath:  p.AvatarPath,
							AvatarURL:   p.AvatarURL,
							Avatar:      p.Avatar,
							VideosCount: p.VideosCount,
							DateStr:     p.DateStr,
							OnOpenFolder: func() {
								if props.OnOpenFolder != nil {
									props.OnOpenFolder(p)
								}
							},
							OnViewVideos: func() {
								if props.OnViewVideos != nil {
									props.OnViewVideos(p)
								}
							},
							OnDelete: func() {
								if props.OnDelete != nil {
									props.OnDelete(p)
								}
							},
						})
					})
				}
			})

			// Pagination Controls
			if totalPages > 1 {
				mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Justify(mygoui.Center).Padding(12).Children(func() {
					prevBtn := mygoui.Button(c, "").Label("Previous").Radius(6)
					if page <= 1 {
						prevBtn.Disabled(true)
					}
					prevBtn.Children(func() {
						mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(4, 12).Children(func() {
							mygoui.Icon(c, ui.IconArrowLeft).Size(14, 14)
							mygoui.Text(c, "Previous")
						})
					})
					if prevBtn.Clicked() && page > 1 && props.OnPageChange != nil {
						props.OnPageChange(page - 1)
					}
					mygoui.Text(c, fmt.Sprintf("Page %d of %d", page, totalPages)).TextColor(t.TextMuted).FontSize(13)
					nextBtn := mygoui.Button(c, "").Label("Next").Radius(6)
					if page >= totalPages {
						nextBtn.Disabled(true)
					}
					nextBtn.Children(func() {
						mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(4, 12).Children(func() {
							mygoui.Text(c, "Next")
						})
					})
					if nextBtn.Clicked() && page < totalPages && props.OnPageChange != nil {
						props.OnPageChange(page + 1)
					}
				})
			}
		})
	})
}
