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
	added := Season{number: c.next(), brk: draft.Break, length: draft.Length, finale: draft.Finale}
	if window := c.With(added).windowOf(added.number); !window.StartsAt.After(now) {
		return Season{}, refused("it would start at %s, not after now", stamp(window.StartsAt))
	}
	return added, nil
}

// Edited is a season that has not ended, planned again; the seasons after it move with it. One that started keeps its
// break, and its end moves only to a time after now. Once its finale started, its end and its finale stay; a finale
// that changes or moves starts after now.
func (c Calendar) Edited(number Number, draft Draft, now time.Time) (Season, error) {
	kept, err := c.season(number)
	if err != nil {
		return Season{}, err
	}
	if err := draft.check(); err != nil {
		return Season{}, err
	}
	window := c.windowOf(number)
	if window.Ended(now) {
		return Season{}, fmt.Errorf("season %d, since %s: %w", number, stamp(window.EndsAt), ErrEnded)
	}
	edited := Season{number: number, brk: draft.Break, length: draft.Length, finale: draft.Finale}
	moved := c.With(edited).windowOf(number)
	if window.Started(now) && draft.Break != kept.brk {
		return Season{}, fmt.Errorf("season %d keeps its start, %s: %w", number, stamp(window.StartsAt), ErrStarted)
	}
	if !window.Started(now) && !moved.StartsAt.After(now) {
		return Season{}, refused("it would start at %s, not after now", stamp(moved.StartsAt))
	}
	if !moved.EndsAt.Equal(window.EndsAt) && !moved.EndsAt.After(now) {
		return Season{}, refused("its end would move to %s, not after now", stamp(moved.EndsAt))
	}
	unmoved := moved.EndsAt.Equal(window.EndsAt) && draft.Finale == kept.finale
	if window.finaleStarted(now) && !unmoved {
		return Season{}, fmt.Errorf("season %d, since %s: %w", number, stamp(window.FinaleStartsAt()), ErrFinaleStarted)
	}
	if !unmoved && moved.Finale.Planned() && !moved.FinaleStartsAt().After(now) {
		return Season{}, refused("its finale would start at %s, not after now", stamp(moved.FinaleStartsAt()))
	}
	return edited, nil
}

// Deletable is nil for the last season when it has not started: numbers keep counting up with no gap.
func (c Calendar) Deletable(number Number, now time.Time) error {
	if _, err := c.season(number); err != nil {
		return err
	}
	if window := c.windowOf(number); window.Started(now) {
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
		windows = append(windows, Window{Number: season.number, StartsAt: start, EndsAt: end, Finale: season.finale})
	}
	return windows
}

func (c Calendar) RunningAt(at time.Time) (Window, bool) {
	for _, window := range c.Windows() {
		if window.Runs(at) {
			return window, true
		}
	}
	return Window{}, false
}

func (c Calendar) NextAfter(at time.Time) (Window, bool) {
	for _, window := range c.Windows() {
		if !window.Started(at) {
			return window, true
		}
	}
	return Window{}, false
}

func (c Calendar) EndedBy(at time.Time) []Window {
	var ended []Window
	for _, window := range c.Windows() {
		if window.Ended(at) {
			ended = append(ended, window)
		}
	}
	return ended
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

// With is the calendar with a season added or changed.
func (c Calendar) With(changed Season) Calendar {
	seasons := slices.DeleteFunc(slices.Clone(c.seasons), func(s Season) bool { return s.number == changed.number })
	return NewCalendar(append(seasons, changed)...)
}
