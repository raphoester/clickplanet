package posts

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"
	"unicode/utf8"

	"github.com/raphoester/clickplanet.lol-backend/internal/publishing/internal/calendar"
)

// The service's id of a post: one post, with a target on each of its platforms.
type PostID string

// Where the service keeps a video uploaded to it, which a post names.
type MediaURL string

// The same key is the same post on the service, however many times it is sent.
type IdempotencyKey string

const (
	// The longest key the service takes: a SHA-256 in hex is 64.
	maxKeyLength = 255
	// TikTok's carousel, and its title over the photos.
	maxPhotos     = 35
	maxPhotoTitle = 90
)

var ErrInvalidPost = errors.New("invalid post")

type TikTokWords struct {
	Caption string
	// A photo post's, over its photos: TikTok strips its hashtags and links. Unused on a video.
	Title string
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
	// Photos posted on TikTok in place of the video, a carousel in this order; none is the video.
	TikTokPhotos []MediaURL
}

// A video to post at a slot, with the words of each of its platforms, and on TikTok photos in its place or not.
type Post struct {
	slot         calendar.Slot
	media        MediaURL
	key          IdempotencyKey
	title        string
	tiktok       TikTokWords
	youtube      YouTubeWords
	metadata     map[string]any
	tiktokPhotos []MediaURL
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
	if !secure(draft.Media) {
		reasons = append(reasons, errors.New("the video is no https address"))
	}
	if err := draft.photosError(); err != nil {
		reasons = append(reasons, err)
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
		key: draft.Key, title: draft.Title, metadata: draft.Metadata, tiktokPhotos: slices.Clone(draft.TikTokPhotos),
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

func secure(media MediaURL) bool {
	address, err := url.Parse(string(media))
	return err == nil && address.Scheme == "https" && address.Host != ""
}

// 35 photos at most, each an https address, and a title of 90 characters at most over them on TikTok.
func (d Draft) photosError() error {
	if len(d.TikTokPhotos) == 0 {
		return nil
	}
	var reasons []error
	if len(d.TikTokPhotos) > maxPhotos {
		reasons = append(reasons, fmt.Errorf("the post has %d photos, and TikTok takes %d at most", len(d.TikTokPhotos), maxPhotos))
	}
	if slices.ContainsFunc(d.TikTokPhotos, func(photo MediaURL) bool { return !secure(photo) }) {
		reasons = append(reasons, errors.New("a photo is no https address"))
	}
	if d.TikTok != nil && utf8.RuneCountInString(d.TikTok.Title) > maxPhotoTitle {
		reasons = append(reasons, fmt.Errorf("the photos' title is longer than %d characters", maxPhotoTitle))
	}
	if slices.Contains(d.Slot.Platforms, calendar.TikTok) && (d.TikTok == nil || d.TikTok.Title == "") {
		reasons = append(reasons, errors.New("the photos have no title"))
	}
	return errors.Join(reasons...)
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
		TikTok: p.tiktok, YouTube: p.youtube, Metadata: p.metadata, TikTokPhotos: slices.Clone(p.tiktokPhotos),
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
	// None is the video on TikTok too.
	TikTokPhotos []MediaURL
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
