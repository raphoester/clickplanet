package calendar

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"time"
)

type Number uint32

const day = 24 * time.Hour

type Season struct {
	Number   Number
	StartsAt time.Time
	// LastDayEndsAt is the season's last cutoff: what follows it, to the season's end, is no day's.
	LastDayEndsAt time.Time
	EndsAt        time.Time
}

type Config struct {
	// DailyCutoff is the time of day, UTC, every day ends at.
	DailyCutoff time.Duration
}

func (c Config) Validate() error {
	if c.DailyCutoff < 0 || c.DailyCutoff >= day {
		return fmt.Errorf("dailyCutoff is %s: a time of day, from 0s to under 24h", c.DailyCutoff)
	}
	return nil
}

// Window is a season's start and end, as the gamemaster told them.
type Window struct {
	Number   Number
	StartsAt time.Time
	EndsAt   time.Time
}

// ErrInvalidSeason refuses a window that ends at or before its start.
var ErrInvalidSeason = errors.New("a season ends after its start")

func (w Window) Check() error {
	if !w.EndsAt.After(w.StartsAt) {
		return fmt.Errorf("%w: season %d starts at %s and ends at %s", ErrInvalidSeason, w.Number,
			w.StartsAt.UTC().Format(time.RFC3339), w.EndsAt.UTC().Format(time.RFC3339))
	}
	return nil
}

// Decision is a season's window and when the gamemaster decided it.
type Decision struct {
	Window    Window
	DecidedAt time.Time
}

// Overrules is true for a decision made after the one kept: one sent again, or older, changes nothing.
func (d Decision) Overrules(kept Decision) bool {
	return d.DecidedAt.After(kept.DecidedAt)
}

// New is the calendar of the seasons told so far. Every copy reads the seasons Play gives it last.
func New(config Config, windows ...Window) Calendar {
	c := Calendar{seasons: &atomic.Pointer[[]Season]{}, cutoff: config.DailyCutoff}
	c.Play(windows...)
	return c
}

type Calendar struct {
	seasons *atomic.Pointer[[]Season]
	cutoff  time.Duration
}

// Play replaces every season with the windows told.
func (c Calendar) Play(windows ...Window) {
	seasons := make([]Season, 0, len(windows))
	for _, window := range windows {
		seasons = append(seasons, Season{
			Number:        window.Number,
			StartsAt:      window.StartsAt.UTC(),
			LastDayEndsAt: lastCutoff(window.EndsAt, c.cutoff),
			EndsAt:        window.EndsAt.UTC(),
		})
	}
	slices.SortFunc(seasons, func(a, b Season) int { return cmp.Compare(a.Number, b.Number) })
	c.seasons.Store(&seasons)
}

// Current is the season that runs at a time: from its start, until its end.
func (c Calendar) Current(now time.Time) (Season, bool) {
	for _, season := range *c.seasons.Load() {
		if !now.Before(season.StartsAt) && season.EndsAt.After(now) {
			return season, true
		}
	}

	return Season{}, false
}

// Ended is every season whose end is at or before a time, first first.
func (c Calendar) Ended(now time.Time) []Season {
	var ended []Season
	for _, season := range *c.seasons.Load() {
		if !season.EndsAt.After(now) {
			ended = append(ended, season)
		}
	}
	return ended
}

// CutoffAfter is the first end of a day after a time.
func (c Calendar) CutoffAfter(at time.Time) time.Time {
	return lastCutoff(at, c.cutoff).Add(day)
}

// lastCutoff is the last end of a day at or before a time.
func lastCutoff(at time.Time, cutoff time.Duration) time.Time {
	at = at.UTC()
	today := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC).Add(cutoff)
	if today.After(at) {
		return today.Add(-day)
	}
	return today
}
