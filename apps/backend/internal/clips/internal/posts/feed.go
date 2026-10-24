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

// Every post of the last day and every one still to come, by hand or by the job.
type Feed []Posting

// Inclusive: the posts at this time or after are what the budget counts.
func PostedSince(now time.Time) time.Time {
	return now.Add(-day)
}

// A post scheduled for later counts already: it will be in the window when the new one goes out.
func (f Feed) spent(platform Platform, now time.Time) int {
	count := 0
	for _, posting := range f {
		if !posting.At.Before(PostedSince(now)) && slices.Contains(posting.Platforms, platform) {
			count++
		}
	}
	return count
}

// Zero when the platform holds no post.
func (f Feed) last(platform Platform) time.Time {
	var last time.Time
	for _, posting := range f {
		if slices.Contains(posting.Platforms, platform) && posting.At.After(last) {
			last = posting.At
		}
	}
	return last
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

// How many posts each platform takes in a trailing day. A platform it does not name takes none.
type Budget map[Platform]int

func (b Budget) left(platform Platform, feed Feed, now time.Time) int {
	return max(0, b[platform]-feed.spent(platform, now))
}

// How many clips a run makes: perRun at most, and no more than the platform with the most room left takes, since a
// clip goes to every platform that still has room.
func (b Budget) Clips(perRun int, platforms []Platform, feed Feed, now time.Time) int {
	room := 0
	for _, platform := range platforms {
		room = max(room, b.left(platform, feed, now))
	}
	return min(perRun, room)
}
