// Package teaser_job is the teasers job's: it decides nothing. It lists the battles promo says are due, hands each
// teaser to the frontend's renderer, uploads what the renderer made to the posting service's storage, and asks promo to
// post it. Promo decides when, where, in what words, and whether at all; the job waits for a busy service.
package teaser_job

import (
	"context"
	"regexp"
	"time"
)

// The id promo names a battle by: an entry's id, or a season's finale's round (season-1/finale).
type BattleID string

// A file the renderer wrote, by its path.
type File string

// The address the posting service's storage names an uploaded file by.
type MediaURL string

// A battle promo says is due.
type Due struct {
	Battle   BattleID
	StartsAt time.Time
	Cards    int
	// promo.v1.Teaser in JSON, as promo answered it: what the renderer reads.
	Teaser []byte
}

var unsafe = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// The name of the teaser's folder in a run: its battle's id, with a mark that is no letter, digit, dot, dash or
// underscore an underscore (season-1/finale is season-1_finale). The folder keeps the id itself in its teaser.json.
func (d Due) FolderName() string {
	name := unsafe.ReplaceAllString(string(d.Battle), "_")
	if name == "" || name == "." || name == ".." {
		return "_" + name
	}
	return name
}

// What the renderer made of a teaser: its cards as JPEGs, in their order, and a slideshow of them.
type Rendered struct {
	Battle BattleID
	Photos []File
	Video  File
}

// The same files, uploaded.
type Media struct {
	Photos []MediaURL
	Video  MediaURL
}

// The server or the storage asked to be tried later, or could not be reached.
type BusyError struct {
	// Zero when it did not say.
	RetryAfter time.Duration
	Reason     string
}

func (e *BusyError) Error() string {
	return e.Reason
}

type Result string

const (
	Posted Result = "posted"
	// No platform takes it before its battle.
	Held Result = "held"
	// No longer due.
	Skipped Result = "skipped"
	Failed  Result = "failed"
)

// What became of a battle's teaser, as promo answered it.
type Outcome struct {
	Battle BattleID
	Result Result
	// Why it was held, skipped or failed.
	Why string
	// Posted: its title, when, on which platforms, the post, and the platforms that refused it.
	Title     string
	At        time.Time
	Platforms []string
	Post      string
	Refused   map[string]string
}

type Report []Outcome

// A teaser failed, or every platform of its post refused it.
func (r Report) Failed() bool {
	for _, outcome := range r {
		if outcome.Result == Failed || (outcome.Result == Posted && len(outcome.Refused) == len(outcome.Platforms)) {
			return true
		}
	}
	return false
}

type Pause interface {
	Wait(ctx context.Context, d time.Duration) error
}

type Backoff []time.Duration

// failed: the tries made so far, from 1. A wait asked for longer than the last of them is not waited.
func (b Backoff) Wait(failed int, asked time.Duration) (time.Duration, bool) {
	if failed < 1 || failed > len(b) || asked > b[len(b)-1] {
		return 0, false
	}
	return max(b[failed-1], asked), true
}

// The pause of a real run.
type SystemPause struct{}

func (SystemPause) Wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the context's own error says it ended.
	case <-timer.C:
		return nil
	}
}
