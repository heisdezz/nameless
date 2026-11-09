package pages

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"nameless/ui"
	"nameless/ui/components"

	mygoui "github.com/egoist/mygo/ui"
)

// VideoItem represents a video in the profile's video queue.
type VideoItem struct {
	ID        int64
	URL       string
	Title     string
	Status    string // done/downloading/failed/pending/paused
	Path      string
	Pct       float64
	Duration  string
	Thumbnail *mygoui.Bitmap
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
	GridCols       *string
	// Pagination props
	Page             int
	PageSize         int
	SearchQuery      *string
	SortBy           *string
	OnSortChange     func(sortBy string)
	OnPageChange     func(page int)
	OnPageSizeChange func(size int)
	// Animation props
	EnableAnimations bool
	// Loading state
	Loading bool
	// Callbacks
	OnBack         func()
	OnDownloadAll  func()
	OnPause        func()
	OnResume       func()
	OnRetryFailed  func()
	OnOpenOutput   func()
	OnOpenVideoDir func(path string)
	OnWatchVideo   func(v VideoItem)
}

// sortOptions are the labels offered in the Sort select.
var sortOptions = []string{"Default", "Newest first", "Oldest first", "Title A-Z", "Title Z-A", "Longest first", "Shortest first", "Needs attention"}

// parseDurationSec parses a "M:SS" or "H:MM:SS" duration; unknown -> 0.
func parseDurationSec(d string) int {
	if d == "" || d == "--:--" {
		return 0
	}
	parts := strings.Split(d, ":")
	total := 0
	for _, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return total
}

// sortVideos returns a sorted copy (the input order is never mutated).
// Empty/"Default" preserves the fetched order.
func sortVideos(videos []VideoItem, sortBy string) []VideoItem {
	if sortBy == "" || sortBy == "Default" {
		return videos
	}
	out := make([]VideoItem, len(videos))
	copy(out, videos)
	statusRank := map[string]int{"downloading": 0, "failed": 1, "pending": 2, "paused": 3, "done": 4}
	switch sortBy {
	case "Newest first":
		sort.SliceStable(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	case "Oldest first":
		sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	case "Title A-Z":
		sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	case "Title Z-A":
		sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Title) > strings.ToLower(out[j].Title) })
	case "Longest first":
		sort.SliceStable(out, func(i, j int) bool { return parseDurationSec(out[i].Duration) > parseDurationSec(out[j].Duration) })
	case "Shortest first":
		sort.SliceStable(out, func(i, j int) bool { return parseDurationSec(out[i].Duration) < parseDurationSec(out[j].Duration) })
	case "Needs attention":
		sort.SliceStable(out, func(i, j int) bool {
			ri, rj := statusRank[out[i].Status], statusRank[out[j].Status]
			if ri != rj {
				return ri < rj
			}
			return out[i].ID > out[j].ID
		})
	}
	return out
}

// filterAndPaginate applies search filtering, sorting, and pagination.
func filterAndPaginate(videos []VideoItem, search, sortBy string, page, pageSize int) ([]VideoItem, int) {
	// Apply search filter
	filtered := videos
	if search != "" {
		searchLower := strings.ToLower(search)
		filtered = make([]VideoItem, 0, len(videos))
		for _, v := range videos {
			if strings.Contains(strings.ToLower(v.Title), searchLower) {
				filtered = append(filtered, v)
			}
		}
	}
	filtered = sortVideos(filtered, sortBy)

	// Calculate total pages
	total := len(filtered)
	if pageSize <= 0 {
		pageSize = 20
	}
	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	// Clamp page to valid range
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	// Slice for current page
	start := (page - 1) * pageSize
	end := start + pageSize
	if end > total {
		end = total
	}
	if start >= total {
		return []VideoItem{}, totalPages
	}

	return filtered[start:end], totalPages
}

// RenderProfileDetail renders the detailed profile video queue and batch actions.
func RenderProfileDetail(c *mygoui.Context, props ProfileDetailProps) {
	t := c.Theme()

	cols := 4
	if props.GridCols != nil && *props.GridCols != "" {
		if val, err := strconv.Atoi(*props.GridCols); err == nil && val >= 3 && val <= 7 {
			cols = val
		}
	}

	// Default page size
	pageSize := props.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	search := ""
	if props.SearchQuery != nil {
		search = *props.SearchQuery
	}

	// Filter and paginate videos
	sortBy := ""
	if props.SortBy != nil {
		sortBy = *props.SortBy
	}
	pageVideos, totalPages := filterAndPaginate(props.Videos, search, sortBy, props.Page, pageSize)

	mygoui.Scroll(c).Fill().Children(func() {
		mygoui.Column(c).Fill().Gap(18).Padding(24).Children(func() {
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

			// Filter & Sort bar: search + sort + layout controls grouped
			// in one card instead of scattered across header/sections.
			mygoui.Column(c).FillWidth().Padding(14).Gap(10).Radius(8).Background(t.Surface).Border(1, t.Border).Children(func() {
				// Wrap: on narrow windows the selects flow to a second
				// line instead of overflowing off the right edge.
				mygoui.Row(c).FillWidth().Wrap().Gap(10).AlignItems(mygoui.Center).Children(func() {
					mygoui.Icon(c, ui.IconSearch).Size(16, 16).TextColor(t.TextMuted)
					if props.SearchQuery != nil {
						mygoui.SearchField(c, props.SearchQuery).Placeholder("Search videos...").Grow(1).MinWidth(180)
					}
					// Sort
					if props.SortBy != nil {
						mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
							mygoui.Text(c, "Sort:").FontSize(12).TextColor(t.TextMuted)
							sel := mygoui.Select(c.Key("detail-sort-select"), props.SortBy, sortOptions).Width(140)
							if sel.Changed() && props.OnSortChange != nil {
								props.OnSortChange(*props.SortBy)
							}
						})
					}
					// Column count selector: 3 to 7 columns
					if props.GridCols != nil {
						mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
							mygoui.Text(c, "Columns:").FontSize(12).TextColor(t.TextMuted)
							mygoui.Select(c.Key("detail-grid-cols-select"), props.GridCols, []string{"3", "4", "5", "6", "7"}).Width(60)
						})
					}
					// Page size selector
					mygoui.Row(c).AlignItems(mygoui.Center).Gap(6).Children(func() {
						mygoui.Text(c, "Per page:").FontSize(12).TextColor(t.TextMuted)
						pageSizeStr := strconv.Itoa(pageSize)
						sel := mygoui.Select(c.Key("detail-page-size-select"), &pageSizeStr, []string{"10", "20", "50", "100"}).Width(60)
						if sel.Changed() && props.OnPageSizeChange != nil {
							if n, err := strconv.Atoi(pageSizeStr); err == nil && n > 0 {
								props.OnPageSizeChange(n)
							}
						}
					})
				})
			})

			// Loading State
			if props.Loading {
				// Full grid loader with overlay
				mygoui.Box(c.Key("loading-grid-overlay")).
					FillWidth().
					Height(400).
					Background(mygoui.Hex("#00000000")).
					Children(func() {
						components.RenderLoader(c, components.LoaderProps{
							Size:        48,
							Text:        "Loading videos...",
							ShowOverlay: true,
							Color:       t.Accent,
						})
					})
			} else {
				// Empty Profile State (plan §4, milestone 3 task E6/E8)
				if len(pageVideos) == 0 {
					if search != "" {
						// No results for search
						mygoui.Column(c).FillWidth().Padding(36).Gap(10).Radius(8).Background(t.Surface).Border(1, t.Border).Center().Children(func() {
							mygoui.Icon(c, ui.IconAlert).Size(24, 24).TextColor(t.TextMuted)
							mygoui.Text(c, "No videos match your search").Bold().FontSize(15)
							mygoui.Text(c, fmt.Sprintf("Try a different search term for \"%s\"", search)).TextColor(t.TextMuted).FontSize(13)
						})
					} else {
						// Empty profile
						mygoui.Column(c).FillWidth().Padding(36).Gap(10).Radius(8).Background(t.Surface).Border(1, t.Border).Center().Children(func() {
							mygoui.Icon(c, ui.IconAlert).Size(24, 24).TextColor(t.TextMuted)
							mygoui.Text(c, "No videos found").Bold().FontSize(15)
							mygoui.Text(c, "No videos found — the account may be private; try cookies in Settings").TextColor(t.TextMuted).FontSize(13)
						})
					}
				} else {
					// Videos Section Header with Columns selector and page size selector
					mygoui.Row(c).AlignItems(mygoui.Center).Justify(mygoui.SpaceBetween).Children(func() {
						mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Children(func() {
							mygoui.Icon(c, ui.IconWave).Size(16, 16).TextColor(t.Text)
							mygoui.Text(c, "Videos").Bold().FontSize(16)
							totalFiltered, _ := filterAndPaginate(props.Videos, search, sortBy, 1, len(props.Videos))
							mygoui.Text(c, fmt.Sprintf("%d of %d items", len(pageVideos), len(totalFiltered))).TextColor(t.TextMuted).FontSize(12)
						})

					})

					// Video Cards Grid (min 3 max 7 columns) with a sweep
					// reveal: each card staggers in left-to-right, top to
					// bottom (delay per docs: rows entering one after
					// another). No grid-level transition — that faded the
					// whole grid at once and fought the per-card sweep.
					grid := mygoui.Grid(c).Columns(cols).Gap(14).FillWidth()
					grid.Children(func() {
						for i, item := range pageVideos {
							v := item
							idx := i

							// Build the card box with base styling
							cardBox := mygoui.Box(c).
								FillWidth().
								Height(280).
								Radius(12).
								Background(t.Surface).
								Border(1, t.Border).
								Clip()

							// Sweep reveal: card i enters after i steps,
							// capped so a full page settles in ~0.8s.
							if props.EnableAnimations {
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
							}

							c.Key(fmt.Sprintf("vid-card-item-%d", v.ID))
							cardBox.Children(func() {
								components.RenderVideoCard(c, components.VideoCardProps{
									ID:        v.ID,
									Index:     idx,
									Title:     v.Title,
									URL:       v.URL,
									Status:    v.Status,
									Path:      v.Path,
									Duration:  v.Duration,
									Pct:       v.Pct,
									Thumbnail: v.Thumbnail,
									OnWatch: func() {
										if props.OnWatchVideo != nil {
											props.OnWatchVideo(v)
										}
									},
									OnOpenFolder: func() {
										if props.OnOpenVideoDir != nil {
											props.OnOpenVideoDir(v.Path)
										}
									},
								})
							})
						}
					})

					// Pagination Controls
					if totalPages > 1 {
						mygoui.Row(c).Gap(8).AlignItems(mygoui.Center).Justify(mygoui.Center).Padding(12).Children(func() {
							// Previous button
							prevBtn := mygoui.Button(c, "").Label("Previous").Radius(6)
							if props.Page <= 1 {
								prevBtn.Disabled(true)
							}
							prevBtn.Children(func() {
								mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(4, 12).Children(func() {
									mygoui.Icon(c, ui.IconArrowLeft).Size(14, 14)
									mygoui.Text(c, "Previous")
								})
							})
							if prevBtn.Clicked() && props.Page > 1 && props.OnPageChange != nil {
								props.OnPageChange(props.Page - 1)
							}

							// Page indicator
							mygoui.Text(c, fmt.Sprintf("Page %d of %d", props.Page, totalPages)).TextColor(t.TextMuted).FontSize(13)

							// Next button
							nextBtn := mygoui.Button(c, "").Label("Next").Radius(6)
							if props.Page >= totalPages {
								nextBtn.Disabled(true)
							}
							nextBtn.Children(func() {
								mygoui.Row(c).Gap(6).AlignItems(mygoui.Center).Padding(4, 12).Children(func() {
									mygoui.Text(c, "Next")
									// No right arrow icon available; text only
								})
							})
							if nextBtn.Clicked() && props.Page < totalPages && props.OnPageChange != nil {
								props.OnPageChange(props.Page + 1)
							}
						})
					}
				}
			}
		})
	})
}
