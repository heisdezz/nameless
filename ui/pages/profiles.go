package pages

import (
	"fmt"

	mygoui "github.com/egoist/mygo/ui"
	"nameless/ui"
)

// ProfileItem represents a profile in the directory.
type ProfileItem struct {
	ID   int64
	Name string
	URL  string
}

// ProfilesProps defines inputs and callbacks for the Profiles overview page.
type ProfilesProps struct {
	Profiles     []ProfileItem
	OnNew        func()
	OnViewVideos func(p ProfileItem)
	OnOpenFolder func(p ProfileItem)
}

// RenderProfiles renders the dedicated Profiles directory page.
func RenderProfiles(c *mygoui.Context, props ProfilesProps) {
	t := c.Theme()
	mygoui.Scroll(c).Fill().Children(func() {
		mygoui.Column(c).Fill().Gap(16).Padding(24).Children(func() {
			// Page Header
			mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
				mygoui.Column(c).Gap(2).Children(func() {
					mygoui.Text(c, "Archived Profiles").FontSize(20).Bold()
					mygoui.Text(c, fmt.Sprintf("%d creator profiles saved", len(props.Profiles))).TextColor(t.TextMuted).FontSize(13)
				})
				if mygoui.PrimaryButton(c.Key("new-profile-btn"), "+ New Download").Radius(6).Clicked() {
					if props.OnNew != nil {
						props.OnNew()
					}
				}
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

			// Profiles Cards Grid/List
			for _, item := range props.Profiles {
				p := item
				c.Key(fmt.Sprintf("prof-card-%d", p.ID))
				ui.RenderProfileCard(c, ui.ProfileCardProps{
					ID:   p.ID,
					Name: p.Name,
					URL:  p.URL,
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
				})
			}
		})
	})
}
