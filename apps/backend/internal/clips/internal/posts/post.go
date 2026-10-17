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
	clip      Clip
	at        time.Time
	platforms []Platform
	wording   wording
}

func (p Post) Reviewable(now time.Time) bool {
	return p.at.Sub(now) >= leastReview
}

func (p Post) TakeSnapshot() PostSnapshot {
	return PostSnapshot{
		Title: p.clip.title, CallToAction: p.wording.call.ID, Key: p.clip.key, Video: p.clip.video, At: p.at,
		Platforms: p.platforms, TikTok: p.wording.tiktok(p.clip), YouTube: p.wording.youtube(p.clip),
	}
}

type PostSnapshot struct {
	Title        string
	CallToAction string
	// Kept with the post, so a later run knows the story is told.
	Key       StoryKey
	Video     VideoFile
	At        time.Time
	Platforms []Platform
	TikTok    TikTokWords
	YouTube   YouTubeWords
}

type Scheduled struct {
	// Empty when no platform took the post.
	ID      PostID
	Refused map[Platform]error
}

type Service interface {
	// For the post's time, never sooner, on each of its platforms that takes it. A post scheduled twice is one post there.
	Schedule(ctx context.Context, post Post) (Scheduled, error)
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
