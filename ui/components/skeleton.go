package components

import (
	"time"

	mygoui "github.com/egoist/mygo/ui"
)

// SkeletonCardProps defines inputs for the skeleton video card placeholder.
type SkeletonCardProps struct {
	// Animation duration for shimmer effect
	ShimmerDuration int // milliseconds
}

// RenderSkeletonVideoCard renders a skeleton placeholder for a video card
// that matches the layout of RenderVideoCard but shows a loading shimmer.
func RenderSkeletonVideoCard(c *mygoui.Context, props SkeletonCardProps) {
	t := c.Theme()

	shimmerDur := 1500
	if props.ShimmerDuration > 0 {
		shimmerDur = props.ShimmerDuration
	}

	cardBox := mygoui.Box(c).
		FillWidth().
		Height(280).
		Radius(12).
		Background(t.Surface).
		Border(1, t.Border).
		Clip()

	// Apply shimmer animation via Transition
	cardBox.Transition(mygoui.ElementTransition{
		Duration: 300 * time.Millisecond,
		Ease:     mygoui.EaseOut,
	})

	cardBox.Children(func() {
		// Cover area with shimmer
		coverBox := mygoui.Box(c.Key("skeleton-cover")).
			FillWidth().
			Height(180).
			Background(mygoui.Hex("#131318")).
			Clip()

		coverBox.Children(func() {
			// Shimmer overlay on cover
			mygoui.Box(c.Key("skeleton-cover-shimmer")).
				Fill().
				Background(mygoui.Hex("#27272a")).
				Transition(mygoui.ElementTransition{
					Duration: time.Duration(shimmerDur) * time.Millisecond,
					Ease:     mygoui.Linear,
				})
		})

		// Metadata area with shimmer lines
		mygoui.Column(c.Key("skeleton-meta")).
			Absolute().Bottom(0).Background(t.Background).FillWidth().Padding(10, 12).Gap(8).Children(func() {
			// Title shimmer line
			mygoui.Box(c.Key("skeleton-title")).
				FillWidth().
				Height(18).
				Radius(4).
				Background(mygoui.Hex("#27272a"))

			// Subtitle shimmer line (shorter)
			mygoui.Box(c.Key("skeleton-subtitle")).
				Width(120).
				Height(14).
				Radius(4).
				Background(mygoui.Hex("#27272a"))

			// Action button shimmer
			mygoui.Box(c.Key("skeleton-action")).
				Width(100).
				Height(28).
				Radius(6).
				Background(mygoui.Hex("#27272a"))
		})
	})
}
