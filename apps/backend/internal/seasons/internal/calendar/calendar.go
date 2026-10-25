package calendar

import (
	"fmt"
	"time"
)

type Number uint32

type Season struct {
	Number         Number
	StartsAt       time.Time
	FinaleStartsAt time.Time
	EndsAt         time.Time
}

type Entry struct {
	Number   Number
	StartsAt time.Time
	EndsAt   time.Time
	Finale   time.Duration
}

type Config struct {
	List []Entry
}

func (c Config) Validate() error {
	var previous time.Time

	for i, entry := range c.List {
		if int(entry.Number) != i {
			return fmt.Errorf("list[%d].number is %d: numbers must count up from 0", i, entry.Number)
		}
		if !entry.EndsAt.After(previous) {
			return fmt.Errorf("list[%d].endsAt is %s: ends must go up", i, entry.EndsAt.Format(time.RFC3339))
		}
		start := entry.startAfter(previous)
		if start.Before(previous) || (!start.IsZero() && !entry.EndsAt.After(start)) {
			return fmt.Errorf("list[%d].startsAt is %s: a season starts once the one before it ended, and before it ends", i,
				entry.StartsAt.Format(time.RFC3339))
		}
		if entry.Finale <= 0 || (!start.IsZero() && entry.Finale >= entry.EndsAt.Sub(start)) {
			return fmt.Errorf("list[%d].finale is %s: a finale must be above 0 and shorter than its season", i, entry.Finale)
		}
		previous = entry.EndsAt
	}

	return nil
}

func (e Entry) startAfter(previous time.Time) time.Time {
	if e.StartsAt.IsZero() {
		return previous
	}
	return e.StartsAt
}

func New(config Config) Calendar {
	seasons := make([]Season, 0, len(config.List))
	var previous time.Time
	for _, entry := range config.List {
		seasons = append(seasons, Season{
			Number:         entry.Number,
			StartsAt:       entry.startAfter(previous),
			FinaleStartsAt: entry.EndsAt.Add(-entry.Finale),
			EndsAt:         entry.EndsAt,
		})
		previous = entry.EndsAt
	}

	return Calendar{seasons: seasons}
}

type Calendar struct {
	seasons []Season
}

func (c Calendar) Current(now time.Time) (Season, bool) {
	for _, season := range c.seasons {
		if season.EndsAt.After(now) && !now.Before(season.StartsAt) {
			return season, true
		}
	}

	return Season{}, false
}

func (c Calendar) Next(now time.Time) (Season, bool) {
	for _, season := range c.seasons {
		if season.StartsAt.After(now) {
			return season, true
		}
	}

	return Season{}, false
}

func (c Calendar) ended(now time.Time) (Season, bool) {
	for i := len(c.seasons) - 1; i >= 0; i-- {
		if !now.Before(c.seasons[i].EndsAt) {
			return c.seasons[i], true
		}
	}

	return Season{}, false
}
