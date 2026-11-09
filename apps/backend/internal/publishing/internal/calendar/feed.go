package calendar

import (
	"slices"
	"time"
)

// The window a platform's cap is counted over: Zernio counts TikTok's as a trailing 24 hours, not a calendar day.
const day = 24 * time.Hour

// A post on the service, whoever made it, on the platforms it goes out on: what counts against a platform's daily
// budget, and what a new post keeps its distance from.
type Posting struct {
	// The post's id on the service; one post may go out at a time of its own on each platform.
	ID        string
	At        time.Time
	Platforms []Platform
}

// Every post of the last day and every one still to come, by hand or by a job. A post still to come is a reservation:
// a new post goes around it, before it when there is room, and never crowds it.
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
