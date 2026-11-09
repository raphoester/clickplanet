package teasers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"time"
)

type Platform string

const (
	TikTok  Platform = "tiktok"
	YouTube Platform = "youtube"
)

// The platforms a teaser asks the calendar for, in turn, until one takes it before its battle: both; else TikTok's
// carousel alone, the teaser's own form; else YouTube's slideshow alone.
var choices = [][]Platform{{TikTok, YouTube}, {TikTok}, {YouTube}}

// A time and the platforms a post goes out on then, as the publishing module's calendar gives it.
type Slot struct {
	At        time.Time
	Platforms []Platform
}

// The address the posting service's storage names an uploaded file by.
type MediaURL string

// The job's render of a teaser, uploaded: its cards as JPEG photos, in their order, and a slideshow of them.
type Media struct {
	Photos []MediaURL
	Video  MediaURL
}

var ErrInvalidPost = errors.New("the teaser's media are not its battle's")

// A teaser to post, at its slot once the calendar gives one.
type Post struct {
	teaser Teaser
	media  Media
	slot   Slot
}

// One photo a card, and a video.
func NewPost(teaser Teaser, media Media) (Post, error) {
	var reasons []error
	if media.Video == "" {
		reasons = append(reasons, errors.New("no video"))
	}
	if len(media.Photos) != len(teaser.Cards) {
		reasons = append(reasons, fmt.Errorf("%d photos for %d cards", len(media.Photos), len(teaser.Cards)))
	}
	if len(reasons) > 0 {
		return Post{}, fmt.Errorf("%w: %w", ErrInvalidPost, errors.Join(reasons...))
	}
	return Post{teaser: teaser, media: media}, nil
}

// The post at another slot.
func (p Post) At(slot Slot) Post {
	p.slot = slot
	return p
}

// The same battle at the same start is the same post on the service, however many times it is tried.
func (p Post) Key() string {
	sum := sha256.Sum256([]byte("teaser\n" + string(p.teaser.Battle) + "\n" + p.teaser.StartsAt.UTC().Format(time.RFC3339)))
	return hex.EncodeToString(sum[:])
}

// Kept with the post, never published: which battle it teases, so it is never teased twice.
func (p Post) Metadata() map[string]any {
	return map[string]any{
		MetadataBattle: string(p.teaser.Battle), "teaserStartsAt": p.teaser.StartsAt.UTC().Format(time.RFC3339),
	}
}

// The metadata that marks a post as a battle's teaser.
const MetadataBattle = "teaserBattle"

func (p Post) Teaser() Teaser { return p.teaser }
func (p Post) Media() Media   { return p.media }
func (p Post) Slot() Slot     { return p.slot }
func (p Post) Words() Words   { return p.teaser.Words() }

// What the publishing module answered of a post it made: its id, and why each platform it was refused on refused it.
type Receipt struct {
	ID      string
	Refused map[Platform]string
}

// The posting service, or publishing, asked to be tried later.
type BusyError struct {
	// Zero when it did not say.
	RetryAfter time.Duration
	Reason     string
}

func (e *BusyError) Error() string {
	return e.Reason
}

// The slot was taken since it was planned, or its lead passed: plan again.
var ErrTaken = errors.New("the slot is no longer open")

// The posting service refused the post for good.
type RefusedError struct {
	Reason string
}

func (e *RefusedError) Error() string {
	return e.Reason
}

type Result string

const (
	Posted Result = "posted"
	// No platform takes it before its battle: their days are full, or their hours past.
	Held Result = "held"
	// No longer due: teased already, or no longer on the schedule.
	Skipped Result = "skipped"
	// The service asked to be tried later: the caller calls again after RetryAfter.
	Busy   Result = "busy"
	Failed Result = "failed"
)

// What became of a teaser.
type Outcome struct {
	Result Result
	// Why it was held, skipped, busy or failed.
	Why string
	// Busy: zero when the service did not say.
	RetryAfter time.Duration
	// Posted: its title, when, on which platforms, the post, and the platforms that refused it.
	Title     string
	At        time.Time
	Platforms []Platform
	Post      string
	Refused   map[Platform]string
}

func PostedOf(post Post, receipt Receipt) Outcome {
	return Outcome{
		Result: Posted, Title: post.Words().Title, At: post.slot.At, Platforms: post.slot.Platforms, Post: receipt.ID,
		Refused: maps.Clone(receipt.Refused),
	}
}

func NotPosted(result Result, why string) Outcome {
	return Outcome{Result: result, Why: why}
}

func BusyOf(err *BusyError) Outcome {
	return Outcome{Result: Busy, Why: err.Reason, RetryAfter: err.RetryAfter}
}
