package posts

import (
	"context"
	"fmt"
	"time"
)

type PostID string

// Under this, nobody has the time to cancel a post, and a service publishes a time already past at once.
const leastReview = time.Hour

type Post struct {
	clip Clip
	at   time.Time
}

func (p Post) Reviewable(now time.Time) bool {
	return p.at.Sub(now) >= leastReview
}

func (p Post) TakeSnapshot() PostSnapshot {
	return PostSnapshot{Title: p.clip.title, Caption: p.clip.caption, Key: p.clip.key, Video: p.clip.video, At: p.at}
}

type PostSnapshot struct {
	Title   string
	Caption string
	// Kept with the post, so a later run knows the story is told.
	Key   StoryKey
	Video VideoFile
	At    time.Time
}

type Service interface {
	// For the post's time, never sooner. A post scheduled twice is one post there.
	Schedule(ctx context.Context, post Post) (PostID, error)
}

type BusyError struct {
	RetryAfter time.Duration
	Reason     string
}

func (e *BusyError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("the posting service is busy for %s: %s", e.RetryAfter, e.Reason)
	}
	return "the posting service is busy: " + e.Reason
}
