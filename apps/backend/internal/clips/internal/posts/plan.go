package posts

import (
	"slices"
	"time"
)

type Plan struct {
	// From the end of the run to the post; none by default.
	Review time.Duration
	// The least time between two posts on one platform, the job's or not.
	Spacing   time.Duration
	Platforms []Platform
	Budget    Budget
}

// The best first: the order a run's posts go out in.
func Ranked(clips []Clip) []Clip {
	return slices.SortedFunc(slices.Values(clips), func(a, b Clip) int { return int(a.pick - b.pick) })
}

// The post of the clip, the place-th of its run's ofRun, on every platform of the plan that has room left in its day,
// a Spacing after the last post there and never sooner than the review: false when no platform has room. It rolls its
// words too, a teaser and a call picked at random, so their variants can be compared.
func (p Plan) Next(clip Clip, place, ofRun int, feed Feed, now time.Time, dice Dice) (Post, bool) {
	var platforms []Platform
	for _, platform := range p.Platforms {
		if p.Budget.left(platform, feed, now) > 0 {
			platforms = append(platforms, platform)
		}
	}
	if len(platforms) == 0 {
		return Post{}, false
	}

	at := now.Add(max(p.Review, leastLead))
	for _, platform := range platforms {
		if last := feed.last(platform); !last.IsZero() {
			at = later(at, last.Add(p.Spacing))
		}
	}
	return Post{
		clip: clip, at: wholeMinuteFrom(at), platforms: platforms, wording: wordingOf(clip, dice), position: place, ofRun: ofRun,
	}, true
}

func later(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}

func wholeMinuteFrom(at time.Time) time.Time {
	whole := at.Truncate(time.Minute)
	if whole.Before(at) {
		whole = whole.Add(time.Minute)
	}
	return whole
}
