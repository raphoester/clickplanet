package posts

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
)

// The service's id of a post: one post, with a target on each of its platforms.
type PostID string

// Where the service keeps a video uploaded to it, which a post names.
type MediaURL string

// The same key is the same post on the service, however many times it is sent.
type IdempotencyKey string

// The longest key the service takes: a SHA-256 in hex is 64.
const maxKeyLength = 255

var ErrInvalidPost = errors.New("invalid post")

type TikTokWords struct {
	Caption string
}

type YouTubeWords struct {
	Title       string
	Description string
	Tags        []string
}

// What a job asks to post, as it asked it.
type Draft struct {
	Slot  calendar.Slot
	Media MediaURL
	Key   IdempotencyKey
	// Its name in the service's dashboard. Never published.
	Title   string
	TikTok  *TikTokWords
	YouTube *YouTubeWords
	// Kept with the post, never published.
	Metadata map[string]any
}

// A video to post at a slot, with the words of each of its platforms.
type Post struct {
	slot     calendar.Slot
	media    MediaURL
	key      IdempotencyKey
	title    string
	tiktok   TikTokWords
	youtube  YouTubeWords
	metadata map[string]any
}

// Every reason at once, as ErrInvalidPost.
func NewPost(draft Draft) (Post, error) {
	var reasons []error
	if len(draft.Slot.Platforms) == 0 {
		reasons = append(reasons, errors.New("the slot names no platform"))
	}
	if draft.Slot.At.IsZero() {
		reasons = append(reasons, errors.New("the slot has no time"))
	}
	if media, err := url.Parse(string(draft.Media)); err != nil || media.Scheme != "https" || media.Host == "" {
		reasons = append(reasons, errors.New("the video is no https address"))
	}
	if draft.Key == "" || len(draft.Key) > maxKeyLength {
		reasons = append(reasons, fmt.Errorf("the idempotency key is empty or longer than %d", maxKeyLength))
	}
	for _, platform := range draft.Slot.Platforms {
		if err := draft.wordsError(platform); err != nil {
			reasons = append(reasons, err)
		}
	}
	if len(reasons) > 0 {
		return Post{}, fmt.Errorf("%w: %w", ErrInvalidPost, errors.Join(reasons...))
	}

	post := Post{
		slot: calendar.Slot{At: draft.Slot.At, Platforms: slices.Clone(draft.Slot.Platforms)}, media: draft.Media,
		key: draft.Key, title: draft.Title, metadata: draft.Metadata,
	}
	if draft.TikTok != nil {
		post.tiktok = *draft.TikTok
	}
	if draft.YouTube != nil {
		post.youtube = *draft.YouTube
		post.youtube.Tags = slices.Clone(draft.YouTube.Tags)
	}
	return post, nil
}

func (d Draft) wordsError(platform calendar.Platform) error {
	switch platform {
	case calendar.TikTok:
		if d.TikTok == nil || d.TikTok.Caption == "" {
			return errors.New("the post goes out on tiktok with no caption")
		}
	case calendar.YouTube:
		if d.YouTube == nil || d.YouTube.Title == "" {
			return errors.New("the post goes out on youtube with no title")
		}
	default:
		return fmt.Errorf("%w %q", calendar.ErrUnknownPlatform, platform)
	}
	return nil
}

func (p Post) Slot() calendar.Slot {
	return calendar.Slot{At: p.slot.At, Platforms: slices.Clone(p.slot.Platforms)}
}

func (p Post) TakeSnapshot() PostSnapshot {
	return PostSnapshot{
		At: p.slot.At, Platforms: slices.Clone(p.slot.Platforms), Media: p.media, Key: p.key, Title: p.title,
		TikTok: p.tiktok, YouTube: p.youtube, Metadata: p.metadata,
	}
}

type PostSnapshot struct {
	At        time.Time
	Platforms []calendar.Platform
	Media     MediaURL
	Key       IdempotencyKey
	Title     string
	TikTok    TikTokWords
	YouTube   YouTubeWords
	Metadata  map[string]any
}

// What the service made of a post.
type Scheduled struct {
	// Empty when no platform took the post.
	ID      PostID
	Refused map[calendar.Platform]error
}

// The service asked to be tried later: a rate limit, a server error, a try made while the one before it is in flight,
// or no answer at all.
type BusyError struct {
	// Zero when it named no time.
	RetryAfter time.Duration
	Reason     string
}

func (e *BusyError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("the posting service is busy for %s: %s", e.RetryAfter, e.Reason)
	}
	return "the posting service is busy: " + e.Reason
}
