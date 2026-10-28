package missions

import (
	"fmt"
	"time"
)

type Config struct {
	Cutoff time.Duration
}

func (c Config) Validate() error {
	if c.Cutoff < 0 || c.Cutoff >= 24*time.Hour {
		return fmt.Errorf("missions.cutoff is %v: a time of day, from 0 up to 24h", c.Cutoff)
	}
	return nil
}

func NewCalendar(config Config) Calendar {
	return Calendar{cutoff: config.Cutoff}
}

type Calendar struct {
	cutoff time.Duration
}

func (c Calendar) DayOf(at time.Time) Day {
	year, month, date := at.UTC().Date()
	end := time.Date(year, month, date, 0, 0, 0, 0, time.UTC).Add(c.cutoff)
	if !at.Before(end) {
		end = end.AddDate(0, 0, 1)
	}
	return Day{end: end}
}

type Day struct {
	end time.Time
}

func (d Day) EndsAt() time.Time { return d.end }

func (d Day) Previous() Day { return Day{end: d.end.AddDate(0, 0, -1)} }

// The date the day ends on: its key in the store.
func (d Day) String() string { return d.end.Format(time.DateOnly) }
