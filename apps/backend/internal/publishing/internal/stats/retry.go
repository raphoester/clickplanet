package stats

import (
	"context"
	"fmt"
	"time"
)

// The platforms whose names a row is read by.
const (
	TikTok  = "tiktok"
	YouTube = "youtube"
)

// The posting service asked to be tried later, or could not be reached.
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

type SystemPause struct{}

var _ Pause = SystemPause{}

func (SystemPause) Wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err() //nolint:wrapcheck // the context's own error says why the wait ended.
	}
}
