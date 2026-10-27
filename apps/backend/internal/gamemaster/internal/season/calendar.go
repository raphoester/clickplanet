package season

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// Calendar is every season, which follow each other by number: each starts at or after the end of the one before.
type Calendar struct {
	seasons []Season
}

func NewCalendar(seasons ...Season) Calendar {
	sorted := slices.Clone(seasons)
	slices.SortFunc(sorted, func(a, b Season) int { return cmp.Compare(a.number, b.number) })
	return Calendar{seasons: sorted}
}

// Added is the season of a draft under the next number: one to come, after the last season.
func (c Calendar) Added(draft Draft, now time.Time) (Season, error) {
	if !draft.StartsAt.After(now) {
		return Season{}, refused("it starts at %s, not after now", stamp(draft.StartsAt))
	}
	added := Season{number: c.next(), startsAt: draft.StartsAt, endsAt: draft.EndsAt}
	if err := c.with(added).check(); err != nil {
		return Season{}, err
	}
	return added, nil
}

// Edited is a season that has not ended with a draft's window. One that started keeps its start, and its end moves
// only to a time after now.
func (c Calendar) Edited(number Number, draft Draft, now time.Time) (Season, error) {
	kept, err := c.season(number)
	if err != nil {
		return Season{}, err
	}
	if kept.ended(now) {
		return Season{}, fmt.Errorf("season %d, since %s: %w", number, stamp(kept.endsAt), ErrEnded)
	}
	if kept.started(now) && !draft.StartsAt.Equal(kept.startsAt) {
		return Season{}, fmt.Errorf("season %d keeps its start, %s: %w", number, stamp(kept.startsAt), ErrStarted)
	}
	if !kept.started(now) && !draft.StartsAt.After(now) {
		return Season{}, refused("it starts at %s, not after now", stamp(draft.StartsAt))
	}
	if !draft.EndsAt.Equal(kept.endsAt) && !draft.EndsAt.After(now) {
		return Season{}, refused("its end moves to %s, not after now", stamp(draft.EndsAt))
	}
	edited := Season{number: number, startsAt: draft.StartsAt, endsAt: draft.EndsAt}
	if err := c.with(edited).check(); err != nil {
		return Season{}, err
	}
	return edited, nil
}

// Deletable is nil for the last season when it has not started: numbers keep counting up with no gap.
func (c Calendar) Deletable(number Number, now time.Time) error {
	kept, err := c.season(number)
	if err != nil {
		return err
	}
	if kept.started(now) {
		return fmt.Errorf("season %d, since %s: %w", number, stamp(kept.startsAt), ErrStarted)
	}
	if number != c.seasons[len(c.seasons)-1].number {
		return fmt.Errorf("season %d: %w", number, ErrNotLast)
	}
	return nil
}

// RunningAt is the season that runs at a time.
func (c Calendar) RunningAt(at time.Time) (Season, bool) {
	for _, season := range c.seasons {
		if season.runs(at) {
			return season, true
		}
	}
	return Season{}, false
}

// NextAfter is the first season that has not started at a time.
func (c Calendar) NextAfter(at time.Time) (Season, bool) {
	for _, season := range c.seasons {
		if !season.started(at) {
			return season, true
		}
	}
	return Season{}, false
}

// StartedBy is every season that has started at a time, the ended ones too: the ones scoring is told.
func (c Calendar) StartedBy(at time.Time) []Season {
	var started []Season
	for _, season := range c.seasons {
		if season.started(at) {
			started = append(started, season)
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

func (c Calendar) with(changed Season) Calendar {
	seasons := slices.DeleteFunc(slices.Clone(c.seasons), func(s Season) bool { return s.number == changed.number })
	return NewCalendar(append(seasons, changed)...)
}

func (c Calendar) check() error {
	for i, season := range c.seasons {
		if !season.endsAt.After(season.startsAt) {
			return refused("season %d ends at %s, not after its start", season.number, stamp(season.endsAt))
		}
		if i > 0 && season.startsAt.Before(c.seasons[i-1].endsAt) {
			return refused("season %d starts at %s, before season %d ends", season.number, stamp(season.startsAt),
				c.seasons[i-1].number)
		}
	}
	return nil
}
