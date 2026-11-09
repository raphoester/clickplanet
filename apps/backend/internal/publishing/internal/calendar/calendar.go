package calendar

import (
	"slices"
	"time"
)

// How far ahead a post is scheduled at the soonest: the service publishes a time already past at once, inside the
// request, and a post counts only when the service keeps it as scheduled.
const leastLead = 5 * time.Minute

// When the game posts: the least time between two posts on one platform, and what each platform takes in a day.
type Calendar struct {
	Spacing time.Duration
	Budget  Budget
}

// What each platform takes. A platform it does not name takes none.
type Budget map[Platform]Allowance

type Allowance struct {
	PerDay int
	Hours  Hours
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

// A time and the platforms a post goes out on then.
type Slot struct {
	At        time.Time
	Platforms []Platform
}

// The game's calendar. An hour between two posts on a platform: two posts minutes apart split the first hour of reach
// that TikTok and YouTube both judge a Short on. TikTok: 3 a trailing day, from 14:00 to 20:00 UTC, after school across
// Europe; its views per clip fell from about 600 to about 15 the day a job posted 10 to 13 a day there (2026-10-19).
// YouTube: 12, from 07:00 to 23:00 UTC; it kept about 1,000 views a clip at 5 to 11 a day.
var Default = Calendar{
	Spacing: time.Hour,
	Budget: Budget{
		TikTok:  {PerDay: 3, Hours: Hours{From: 14, To: 20}},
		YouTube: {PerDay: 12, Hours: Hours{From: 7, To: 23}},
	},
}

// How many more posts at at the platform takes: none once any trailing day that holds at is full.
func (c Calendar) Left(platform Platform, feed Feed, at time.Time) int {
	return max(0, c.Budget[platform].PerDay-feed.fullestDay(platform, at))
}

// The slot of a post on the platforms, from notBefore on and never sooner than the lead from now: the first whole minute
// that is a Spacing from every post on them, before it or after it, on each of them in its hours with room left in every
// trailing day that holds it. False when none of them takes it. A platform left out no longer holds the post back, so
// the time is found again for the others.
func (c Calendar) Slot(platforms []Platform, feed Feed, notBefore, now time.Time) (Slot, bool) {
	from := wholeMinuteFrom(later(now.Add(leastLead), notBefore))
	at := c.at(platforms, feed, from)
	for open := c.open(platforms, feed, at); len(open) < len(platforms); open = c.open(platforms, feed, at) {
		platforms = open
		at = c.at(platforms, feed, from)
	}
	if len(platforms) == 0 {
		return Slot{}, false
	}
	return Slot{At: at, Platforms: slices.Clone(platforms)}, true
}

// Whether a post can still go out at the slot: on a whole minute, the lead from now or later, and on each of its
// platforms in its hours, with room left and a Spacing from every other post.
func (c Calendar) Open(slot Slot, feed Feed, now time.Time) bool {
	if len(slot.Platforms) == 0 || !slot.At.Equal(slot.At.Truncate(time.Minute)) || slot.At.Before(now.Add(leastLead)) {
		return false
	}
	if _, crowded := feed.clash(slot.Platforms, slot.At, c.Spacing); crowded {
		return false
	}
	return len(c.open(slot.Platforms, feed, slot.At)) == len(slot.Platforms)
}

// The first whole minute from from on that is a Spacing from every post on the platforms, before it or after it.
func (c Calendar) at(platforms []Platform, feed Feed, from time.Time) time.Time {
	at := from
	// Each clash moves the time past a post for good, so this ends within as many turns as the feed has posts.
	for clash, crowded := feed.clash(platforms, at, c.Spacing); crowded; clash, crowded = feed.clash(platforms, at, c.Spacing) {
		at = wholeMinuteFrom(clash.Add(c.Spacing))
	}
	return at
}

// The platforms that take a post at at: in their hours, with room left in every trailing day that holds it.
func (c Calendar) open(platforms []Platform, feed Feed, at time.Time) []Platform {
	var open []Platform
	for _, platform := range platforms {
		if c.Budget[platform].Hours.hold(at) && c.Left(platform, feed, at) > 0 {
			open = append(open, platform)
		}
	}
	return open
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
