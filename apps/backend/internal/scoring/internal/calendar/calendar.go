package calendar

import (
	"fmt"
	"time"
)

type Number uint32

const day = 24 * time.Hour

type Season struct {
	Number Number
	// LastDayEndsAt is the season's last cutoff: what follows it, to the season's end, is no day's.
	LastDayEndsAt time.Time
	EndsAt        time.Time
}

type Entry struct {
	Number Number
	EndsAt time.Time
}

type Config struct {
	List []Entry
	// DailyCutoff is the time of day, UTC, every day ends at.
	DailyCutoff time.Duration
}

func (c Config) Validate() error {
	if c.DailyCutoff < 0 || c.DailyCutoff >= day {
		return fmt.Errorf("dailyCutoff is %s: a time of day, from 0s to under 24h", c.DailyCutoff)
	}

	var previous time.Time
	for i, entry := range c.List {
		if int(entry.Number) != i {
			return fmt.Errorf("list[%d].number is %d: numbers must count up from 0", i, entry.Number)
		}
		if !entry.EndsAt.After(previous) {
			return fmt.Errorf("list[%d].endsAt is %s: ends must go up", i, entry.EndsAt.Format(time.RFC3339))
		}
		if i > 0 && !lastCutoff(entry.EndsAt, c.DailyCutoff).After(previous) {
			return fmt.Errorf("list[%d].endsAt is %s: a season holds the end of a day", i, entry.EndsAt.Format(time.RFC3339))
		}
		previous = entry.EndsAt
	}

	return nil
}

func New(config Config) Calendar {
	seasons := make([]Season, 0, len(config.List))
	for _, entry := range config.List {
		seasons = append(seasons, Season{
			Number:        entry.Number,
			LastDayEndsAt: lastCutoff(entry.EndsAt, config.DailyCutoff),
			EndsAt:        entry.EndsAt,
		})
	}

	return Calendar{seasons: seasons, cutoff: config.DailyCutoff}
}

type Calendar struct {
	seasons []Season
	cutoff  time.Duration
}

func (c Calendar) Current(now time.Time) (Season, bool) {
	for _, season := range c.seasons {
		if season.EndsAt.After(now) {
			return season, true
		}
	}

	return Season{}, false
}

// Ended is every season whose end is at or before a time, first first.
func (c Calendar) Ended(now time.Time) []Season {
	var ended []Season
	for _, season := range c.seasons {
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
