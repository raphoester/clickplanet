package rules

import (
	"errors"
	"fmt"
	"time"
)

// maxOverride is the longest an override runs, so none is left on by mistake for long.
const maxOverride = 24 * time.Hour

var ErrInvalidOverride = errors.New("this is not an override")

// Override is a set an operator puts over the baseline, from a time until another.
type Override struct {
	set   SetName
	from  time.Time
	until time.Time
}

func NewOverride(set Set, now, until time.Time) (Override, error) {
	switch {
	case !until.After(now):
		return Override{}, fmt.Errorf("%w: it would end at %s, which is not after now", ErrInvalidOverride,
			until.UTC().Format(time.RFC3339))
	case until.Sub(now) > maxOverride:
		return Override{}, fmt.Errorf("%w: it would run %s, and runs %s at most", ErrInvalidOverride,
			until.Sub(now).Round(time.Second), maxOverride)
	}

	return Override{set: set.name, from: now, until: until}, nil
}

// EndedAt is the override ended at now, or the same override when it was not running then.
func (o Override) EndedAt(now time.Time) Override {
	if !o.runningAt(now) {
		return o
	}

	o.until = now
	return o
}

func (o Override) runningAt(now time.Time) bool {
	return o.set != "" && !now.Before(o.from) && now.Before(o.until)
}

func (o Override) TakeSnapshot() OverrideSnapshot {
	return OverrideSnapshot{Set: o.set, StartsAt: o.from, EndsAt: o.until}
}

type OverrideSnapshot struct {
	Set      SetName
	StartsAt time.Time
	EndsAt   time.Time
}

func (s OverrideSnapshot) Restore() Override {
	return Override{set: s.Set, from: s.StartsAt, until: s.EndsAt}
}
