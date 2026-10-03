package calendar

import (
	"fmt"
	"time"
)

type Number uint32

type Season struct {
	Number         Number
	FinaleStartsAt time.Time
	EndsAt         time.Time
}

type Entry struct {
	Number Number
	EndsAt time.Time
	Finale time.Duration
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
		if entry.Finale <= 0 || (i > 0 && entry.Finale >= entry.EndsAt.Sub(previous)) {
			return fmt.Errorf("list[%d].finale is %s: a finale must be above 0 and shorter than its season", i, entry.Finale)
		}
		previous = entry.EndsAt
	}

	return nil
}

func New(config Config) Calendar {
	seasons := make([]Season, 0, len(config.List))
	for _, entry := range config.List {
		seasons = append(seasons, Season{
			Number:         entry.Number,
			FinaleStartsAt: entry.EndsAt.Add(-entry.Finale),
			EndsAt:         entry.EndsAt,
		})
	}

	return Calendar{seasons: seasons}
}

type Calendar struct {
	seasons []Season
}

func (c Calendar) Current(now time.Time) (Season, bool) {
	for _, season := range c.seasons {
		if season.EndsAt.After(now) {
			return season, true
		}
	}

	return Season{}, false
}

func (c Calendar) Season(number Number) (Season, bool) {
	for _, season := range c.seasons {
		if season.Number == number {
			return season, true
		}
	}

	return Season{}, false
}
