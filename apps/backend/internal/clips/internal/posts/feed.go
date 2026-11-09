package posts

import (
	"slices"
	"time"
)

// The window a platform's cap is counted over: Zernio counts TikTok's as a trailing 24 hours, not a calendar day.
const day = 24 * time.Hour

// A post on the service, whoever made it, on the platforms it goes out on: what counts against a platform's daily
// budget, and what a new post keeps its distance from.
type Posting struct {
	At        time.Time
	Platforms []Platform
}

// Every post of the last day and every one still to come, by hand or by the job. A post still to come is a
// reservation: a new post goes around it, before it when there is room, and never crowds it.
type Feed []Posting

// Inclusive: the posts at this time or after are what the budget of a post from now on counts.
func PostedSince(now time.Time) time.Time {
	return now.Add(-day)
}

// The most posts on the platform in any trailing day that holds at: its own, and the one of each post on the platform
// in the day after it, which a post at at would count in too.
func (f Feed) fullestDay(platform Platform, at time.Time) int {
	fullest := f.dayBefore(platform, at)
	for _, posting := range f {
		if posting.At.After(at) && !posting.At.After(at.Add(day)) && slices.Contains(posting.Platforms, platform) {
			fullest = max(fullest, f.dayBefore(platform, posting.At))
		}
	}
	return fullest
}

// The posts on the platform in the trailing day that ends at end, both ends included.
func (f Feed) dayBefore(platform Platform, end time.Time) int {
	count := 0
	for _, posting := range f {
		if !posting.At.Before(end.Add(-day)) && !posting.At.After(end) && slices.Contains(posting.Platforms, platform) {
			count++
		}
	}
	return count
}

// The latest post on any of the platforms less than spacing from at, before it or after it: false when none is.
func (f Feed) clash(platforms []Platform, at time.Time, spacing time.Duration) (time.Time, bool) {
	var latest time.Time
	for _, posting := range f {
		if posting.At.Sub(at).Abs() < spacing && posting.At.After(latest) && sharesOne(posting.Platforms, platforms) {
			latest = posting.At
		}
	}
	return latest, !latest.IsZero()
}

func sharesOne(a, b []Platform) bool {
	return slices.ContainsFunc(a, func(platform Platform) bool { return slices.Contains(b, platform) })
}

// With the post the service took, on each platform that took it.
func (f Feed) With(post Post, scheduled Scheduled) Feed {
	if scheduled.ID == "" {
		return f
	}
	var took []Platform
	for _, platform := range post.platforms {
		if scheduled.Refused[platform] == nil {
			took = append(took, platform)
		}
	}
	return append(slices.Clone(f), Posting{At: post.at, Platforms: took})
}

// What each platform takes. A platform it does not name takes none.
type Budget map[Platform]Allowance

type Allowance struct {
	PerDay   int
	Hours    Hours
	BestOnly bool
}

// UTC, From included, To not. Zero is every hour.
type Hours struct {
	From int
	To   int
}

func (h Hours) hold(at time.Time) bool {
	if h == (Hours{}) {
		return true
	}
	hour := at.UTC().Hour()
	return hour >= h.From && hour < h.To
}

// How many more posts at at the platform takes: none once any trailing day that holds at is full.
func (b Budget) left(platform Platform, feed Feed, at time.Time) int {
	return max(0, b[platform].PerDay-feed.fullestDay(platform, at))
}

// The platforms the place-th post of a run may go to: one that takes the best only takes the first.
func (b Budget) platformsOf(platforms []Platform, place int) []Platform {
	var of []Platform
	for _, platform := range platforms {
		if !b[platform].BestOnly || place == 1 {
			of = append(of, platform)
		}
	}
	return of
}

// Whether a post at at goes out on the platform: in its hours, with room left in every trailing day that holds it.
func (b Budget) open(platform Platform, feed Feed, at time.Time) bool {
	return b[platform].Hours.hold(at) && b.left(platform, feed, at) > 0
}

// How many clips a run makes: perRun at most, and no more than the platform with the most room left now takes, since a
// clip goes to every platform that still has room. Not the hours: a clip's time is known once it is rendered.
func (b Budget) Clips(perRun int, platforms []Platform, feed Feed, now time.Time) int {
	room := 0
	for _, platform := range platforms {
		left := b.left(platform, feed, now)
		if b[platform].BestOnly {
			left = min(left, 1)
		}
		room = max(room, left)
	}
	return min(perRun, room)
}
