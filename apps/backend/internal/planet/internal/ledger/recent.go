package ledger

import (
	"time"
)

const (
	recentSpan = 24 * time.Hour
	recentStep = 30 * time.Minute
)

// Recent is the map's history anyone may read: the 24 hours up to the last :00 or :30 UTC, a frame each half hour.
type Recent struct {
	Since time.Time
	Until time.Time
	Step  time.Duration
}

func RecentAt(now time.Time) Recent {
	until := now.UTC().Truncate(recentStep)
	return Recent{Since: until.Add(-recentSpan), Until: until, Step: recentStep}
}

func (r Recent) Next() time.Time {
	return r.Until.Add(r.Step)
}

// Wiped starts the history at the first step at or after a reset in it: the map before the reset is gone.
func (r Recent) Wiped(at time.Time) Recent {
	if !at.After(r.Since) {
		return r
	}
	since := at.UTC().Truncate(r.Step)
	if since.Before(at) {
		since = since.Add(r.Step)
	}
	r.Since = since
	return r
}

func (r Recent) Empty() bool {
	return r.Since.After(r.Until)
}
