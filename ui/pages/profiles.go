package pages

import (
	"fmt"
	"strconv"

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
	VideosCount int
	DateStr     string
}

// ProfilesProps defines inputs and callbacks for the Profiles overview page.
type ProfilesProps struct {
	Profiles     []ProfileItem
	GridCols     *string
	OnNew        func()
	OnViewVideos func(p ProfileItem)
	OnOpenFolder func(p ProfileItem)
	OnDelete     func(p ProfileItem)
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

			// Profiles Cards Grid (default 5 columns, min 3 max 7)
			mygoui.Grid(c).Columns(cols).Gap(14).FillWidth().Children(func() {
				for _, item := range props.Profiles {
					p := item
					c.Key(fmt.Sprintf("prof-card-%d", p.ID))
					components.RenderProfileCard(c, components.ProfileCardProps{
						ID:          p.ID,
						Name:        p.Name,
						URL:         p.URL,
						AvatarPath:  p.AvatarPath,
						AvatarURL:   p.AvatarURL,
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
				}
			})
		})
	})
}
