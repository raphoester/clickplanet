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

// A file the renderer wrote, by its path.
type File string

// What the renderer made of a teaser: its cards as JPEG photos, in their order, and a slideshow of them.
type Media struct {
	Photos []File
	Video  File
}

// A folder of a run: the battle it renders, and what it holds.
type Rendered struct {
	Battle BattleID
	Media  Media
}

// A teaser to post, at its slot once the calendar gives one.
type Post struct {
	Teaser Teaser
	Media  Media
	Slot   Slot
}

// The same battle at the same start is the same post on the service, however many times a run tries it, or a run is
// made again.
func (p Post) Key() string {
	sum := sha256.Sum256([]byte("teaser\n" + string(p.Teaser.Battle) + "\n" + p.Teaser.StartsAt.UTC().Format(time.RFC3339)))
	return hex.EncodeToString(sum[:])
}

// Kept with the post, never published: which battle it teases, so no run posts it twice.
func (p Post) Metadata() map[string]any {
	return map[string]any{
		MetadataBattle: string(p.Teaser.Battle), "teaserStartsAt": p.Teaser.StartsAt.UTC().Format(time.RFC3339),
	}
}

// The metadata that marks a post as a battle's teaser.
const MetadataBattle = "teaserBattle"

// What the publishing module answered of a post it made: its id, and why each platform it was refused on refused it.
type Receipt struct {
	ID      string
	Refused map[Platform]string
}

// The service or the backend asked to be tried later.
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

type Result string

const (
	Scheduled Result = "scheduled"
	// No platform takes it before its battle: their days are full, or their hours past.
	Held Result = "held"
	// Not rendered, or no longer due.
	Skipped Result = "skipped"
	Failed  Result = "failed"
)

// What became of a battle's teaser in a run.
type Outcome struct {
	Battle BattleID
	Title  string
	Result Result
	// Why it was held, skipped or failed.
	Why string
	// Scheduled: when, on which platforms, the post, and the platforms that refused it.
	At        time.Time
	Platforms []Platform
	Post      string
	Refused   map[Platform]string
}

type Report []Outcome

func (r Report) With(post Post, receipt Receipt, held bool, err error) Report {
	outcome := Outcome{Battle: post.Teaser.Battle, Title: post.Teaser.Words().Title}
	switch {
	case held:
		outcome.Result, outcome.Why = Held, "no platform takes it before its battle: their days are full, or their hours past"
	case err != nil:
		outcome.Result, outcome.Why = Failed, err.Error()
	default:
		outcome.Result, outcome.At, outcome.Post = Scheduled, post.Slot.At, receipt.ID
		outcome.Platforms, outcome.Refused = post.Slot.Platforms, maps.Clone(receipt.Refused)
	}
	return append(r, outcome)
}

func (r Report) WithSkipped(battle BattleID, title, why string) Report {
	return append(r, Outcome{Battle: battle, Title: title, Result: Skipped, Why: why})
}

// A teaser failed, or was refused on every platform of its slot.
func (r Report) Failed() bool {
	for _, outcome := range r {
		if outcome.Result == Failed || (outcome.Result == Scheduled && len(outcome.Refused) == len(outcome.Platforms)) {
			return true
		}
	}
	return false
}

func (o Outcome) String() string {
	if o.Result == Scheduled {
		return fmt.Sprintf("%s: scheduled at %s on %v", o.Title, o.At.UTC().Format(time.RFC3339), o.Platforms)
	}
	return fmt.Sprintf("%s: %s, %s", o.Title, o.Result, o.Why)
}
