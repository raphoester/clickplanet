package posts

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
)

type PostID string

type Post struct {
	clip      Clip
	at        time.Time
	platforms []Platform
	wording   wording
	// From 1: the order the run's posts go out in, and how many it has.
	position int
	ofRun    int
}

// A time and the platforms a post goes out on then, as the publishing module's calendar gives it.
type Slot struct {
	At        time.Time
	Platforms []Platform
}

// The post at the slot: its time, and the platforms that take it then.
func (p Post) At(slot Slot) Post {
	p.at = slot.At
	p.platforms = slices.Clone(slot.Platforms)
	return p
}

func (p Post) TakeSnapshot() PostSnapshot {
	return PostSnapshot{
		Title: p.clip.title, CallToAction: p.wording.call.ID, Key: p.clip.key, Countries: p.clip.countryCodes(),
		Video: p.clip.video, At: p.at, Platforms: p.platforms, TikTok: p.wording.tiktok(p.clip), YouTube: p.wording.youtube(p.clip),
		Record: p.clip.record, Teaser: p.wording.teaserName, RunPosition: p.position, RunSize: p.ofRun,
		Variants: p.clip.variants,
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
	// The arm of each experiment its clip is in.
	Variants Variants
}

type Scheduled struct {
	// Empty when no platform took the post.
	ID      PostID
	Refused map[Platform]error
}

// The slot the post was planned at is no longer open: another post took its room or its spacing. Plan again.
var ErrTaken = errors.New("the slot was taken")

type Service interface {
	// At the post's slot, on each of its platforms that takes it, once: a *BusyError to try later, ErrTaken to plan again.
	// A post scheduled twice is one post there.
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
