package posts

import (
	"context"
	"fmt"
	"time"
)

type PostID string

// How far ahead a post is scheduled at the soonest: a service publishes a time already past at once, inside the request,
// and the job counts a post only when the service keeps it as scheduled.
const leastLead = 5 * time.Minute

type Post struct {
	clip      Clip
	at        time.Time
	platforms []Platform
	wording   wording
	// From 1: the order the run's posts go out in, and how many it has.
	position int
	ofRun    int
	// The arms it was moved to when it was posted, by experiment, over its clip's; nil for none.
	moved Variants
	// The trending track attached on TikTok; nil for none, the clip's own sound.
	sound *Sound
	// Why an arm it was drawn for could not be applied, when it fell back.
	missed error
}

// The post as it is sent at now: at its time, or leastLead from now when a wait ate the lead.
func (p Post) Rescheduled(now time.Time) Post {
	if p.at.Sub(now) < leastLead {
		p.at = wholeMinuteFrom(now.Add(leastLead))
	}
	return p
}

func (p Post) TakeSnapshot() PostSnapshot {
	return PostSnapshot{
		Title: p.clip.title, CallToAction: p.wording.call.ID, Key: p.clip.key, Countries: p.clip.countryCodes(),
		Video: p.clip.video, At: p.at, Platforms: p.platforms, TikTok: p.wording.tiktok(p.clip), YouTube: p.wording.youtube(p.clip),
		Record: p.clip.record, Teaser: p.wording.teaserName, RunPosition: p.position, RunSize: p.ofRun,
		Variants: p.variants(), Moved: p.moved, Sound: p.sound,
	}
}

type PostSnapshot struct {
	Title        string
	CallToAction string
	// Kept with the post, so a later run knows the story is told and which flags it showed.
	Key       StoryKey
	Countries []string
	Video     VideoFile
	At        time.Time
	Platforms []Platform
	TikTok    TikTokWords
	YouTube   YouTubeWords
	// What the clip was, and what this job made of it, kept with the post to compare posts by: the name of the teaser
	// rolled ("invades", or "unnamed"), and the post's place in its run, from 1, of how many.
	Record      ClipRecord
	Teaser      string
	RunPosition int
	RunSize     int
	// The arm of each experiment the post goes out in: its clip's, and those it was moved to when it was posted.
	Variants Variants
	// The arms it was moved to when it was posted ({"sound": "fell-back"}); none when it goes out in its clip's.
	Moved Variants
	// The trending track to attach on TikTok, for the arm `trending` of the experiment "sound"; nil for none, the
	// control, which an adapter sends as it was before the experiment.
	Sound *Sound
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
