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

// The post of the clip, the place-th of its run's ofRun, on every platform of the plan that takes it in its hours, a
// Spacing from every other post there, before it or after it, and never sooner than the review: false when none does.
// A post still to come, made by hand or by another run, is a reservation: the post goes before it when the gap and the
// days around it leave room. It rolls its words too, a teaser and a call picked at random, so their variants can be
// compared.
func (p Plan) Next(clip Clip, place, ofRun int, feed Feed, now time.Time, dice Dice) (Post, bool) {
	platforms := p.Budget.platformsOf(p.Platforms, place)
	at := p.at(platforms, feed, now)
	// A platform left out no longer holds the post back, so the time is found again for the others.
	for open := p.open(platforms, feed, at); len(open) < len(platforms); open = p.open(platforms, feed, at) {
		platforms = open
		at = p.at(platforms, feed, now)
	}
	if len(platforms) == 0 {
		return Post{}, false
	}

	return Post{
		clip: clip, at: at, platforms: platforms, wording: wordingOf(clip, dice), position: place, ofRun: ofRun,
	}, true
}

// The first whole minute from the review on that is a Spacing from every post on the platforms, before it or after it.
func (p Plan) at(platforms []Platform, feed Feed, now time.Time) time.Time {
	at := wholeMinuteFrom(now.Add(max(p.Review, leastLead)))
	// Each clash moves the time past a post for good, so this ends within as many turns as the feed has posts.
	for clash, crowded := feed.clash(platforms, at, p.Spacing); crowded; clash, crowded = feed.clash(platforms, at, p.Spacing) {
		at = wholeMinuteFrom(clash.Add(p.Spacing))
	}
	return at
}

// The platforms that take a post at at: in their hours, with room left in every trailing day that holds it.
func (p Plan) open(platforms []Platform, feed Feed, at time.Time) []Platform {
	var open []Platform
	for _, platform := range platforms {
		if p.Budget.open(platform, feed, at) {
			open = append(open, platform)
		}
	}
	return open
}

func wholeMinuteFrom(at time.Time) time.Time {
	whole := at.Truncate(time.Minute)
	if whole.Before(at) {
		whole = whole.Add(time.Minute)
	}
	return whole
}
