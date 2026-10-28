package season

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// Calendar is every season, one after the other by number: each starts its break after the end of the one before.
type Calendar struct {
	seasons []Season
}

func NewCalendar(seasons ...Season) Calendar {
	sorted := slices.Clone(seasons)
	slices.SortFunc(sorted, func(a, b Season) int { return cmp.Compare(a.number, b.number) })
	return Calendar{seasons: sorted}
}

// Added is the season of a draft after the last one, under the next number. It starts after now.
func (c Calendar) Added(draft Draft, now time.Time) (Season, error) {
	if err := draft.check(); err != nil {
		return Season{}, err
	}
	added := Season{number: c.next(), brk: draft.Break, length: draft.Length}
	if window := c.with(added).windowOf(added.number); !window.StartsAt.After(now) {
		return Season{}, refused("it would start at %s, not after now", stamp(window.StartsAt))
	}
	return added, nil
}

// Edited is a season that has not ended, planned again; the seasons after it move with it. One that started keeps its
// break, and its end moves only to a time after now.
func (c Calendar) Edited(number Number, draft Draft, now time.Time) (Season, error) {
	kept, err := c.season(number)
	if err != nil {
		return Season{}, err
	}
	if err := draft.check(); err != nil {
		return Season{}, err
	}
	window := c.windowOf(number)
	if window.ended(now) {
		return Season{}, fmt.Errorf("season %d, since %s: %w", number, stamp(window.EndsAt), ErrEnded)
	}
	edited := Season{number: number, brk: draft.Break, length: draft.Length}
	moved := c.with(edited).windowOf(number)
	if window.started(now) && draft.Break != kept.brk {
		return Season{}, fmt.Errorf("season %d keeps its start, %s: %w", number, stamp(window.StartsAt), ErrStarted)
	}
	if !window.started(now) && !moved.StartsAt.After(now) {
		return Season{}, refused("it would start at %s, not after now", stamp(moved.StartsAt))
	}
	if !moved.EndsAt.Equal(window.EndsAt) && !moved.EndsAt.After(now) {
		return Season{}, refused("its end would move to %s, not after now", stamp(moved.EndsAt))
	}
	return edited, nil
}

// Deletable is nil for the last season when it has not started: numbers keep counting up with no gap.
func (c Calendar) Deletable(number Number, now time.Time) error {
	if _, err := c.season(number); err != nil {
		return err
	}
	if window := c.windowOf(number); window.started(now) {
		return fmt.Errorf("season %d, since %s: %w", number, stamp(window.StartsAt), ErrStarted)
	}
	if number != c.seasons[len(c.seasons)-1].number {
		return fmt.Errorf("season %d: %w", number, ErrNotLast)
	}
	return nil
}

func (c Calendar) Windows() []Window {
	windows := make([]Window, 0, len(c.seasons))
	end := Origin
	for _, season := range c.seasons {
		start := end.Add(season.brk)
		end = start.Add(season.length)
		windows = append(windows, Window{Number: season.number, StartsAt: start, EndsAt: end})
	}
	return windows
}

func (c Calendar) RunningAt(at time.Time) (Window, bool) {
	for _, window := range c.Windows() {
		if window.runs(at) {
			return window, true
		}
	}
	return Window{}, false
}

func (c Calendar) NextAfter(at time.Time) (Window, bool) {
	for _, window := range c.Windows() {
		if !window.started(at) {
			return window, true
		}
	}
	return Window{}, false
}

// StartedBy is every season that has started at a time, the ended ones too: the ones scoring is told.
func (c Calendar) StartedBy(at time.Time) []Window {
	var started []Window
	for _, window := range c.Windows() {
		if window.started(at) {
			started = append(started, window)
		}
	}
	return started
}

func (c Calendar) TakeSnapshot() []Snapshot {
	snapshots := make([]Snapshot, 0, len(c.seasons))
	for _, season := range c.seasons {
		snapshots = append(snapshots, season.TakeSnapshot())
	}
	return snapshots
}

func (c Calendar) next() Number {
	if len(c.seasons) == 0 {
		return 0
	}
	return c.seasons[len(c.seasons)-1].number + 1
}

func (c Calendar) season(number Number) (Season, error) {
	for _, season := range c.seasons {
		if season.number == number {
			return season, nil
		}
	}
	return Season{}, fmt.Errorf("%d: %w", number, ErrUnknownSeason)
}

func (c Calendar) windowOf(number Number) Window {
	for _, window := range c.Windows() {
		if window.Number == number {
			return window
		}
	}
	return Window{}
}

func (c Calendar) with(changed Season) Calendar {
	seasons := slices.DeleteFunc(slices.Clone(c.seasons), func(s Season) bool { return s.number == changed.number })
	return NewCalendar(append(seasons, changed)...)
}
