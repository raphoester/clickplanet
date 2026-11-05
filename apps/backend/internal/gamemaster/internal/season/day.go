package season

import (
	"fmt"
	"time"
)

const day = 24 * time.Hour

// Cutoff is the time of day, UTC, every day of a season ends at.
type Cutoff time.Duration

func (c Cutoff) Check() error {
	if c < 0 || time.Duration(c) >= day {
		return fmt.Errorf("the daily cutoff is %s: a time of day, from 0s to under 24h", time.Duration(c))
	}
	return nil
}

func (c Cutoff) Is(at time.Time) bool {
	at = at.UTC()
	return at.Sub(time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)) == time.Duration(c)
}

func (c Cutoff) String() string {
	return time.Time{}.Add(time.Duration(c)).Format("15:04") + " UTC"
}

// DayNumber counts a season's days up from 1.
type DayNumber uint32

// Day K of a season ends at the K-th cutoff after the season's start and starts at the cutoff before it, or at the
// season's start for day 1. No day runs from the season's last cutoff to its end.
type Day struct {
	Season   Number
	Number   DayNumber
	StartsAt time.Time
	EndsAt   time.Time
}

func (d Day) Runs(at time.Time) bool {
	return !at.Before(d.StartsAt) && at.Before(d.EndsAt)
}

// Days are the season's days, by number.
func (w Window) Days(cutoff Cutoff) []Day {
	var days []Day
	start := w.StartsAt.UTC()
	for end := cutoff.after(start); !end.After(w.EndsAt); end = end.Add(day) {
		days = append(days, Day{Season: w.Number, Number: DayNumber(len(days) + 1), StartsAt: start, EndsAt: end})
		start = end
	}
	return days
}

func (c Cutoff) after(at time.Time) time.Time {
	at = at.UTC()
	today := time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC).Add(time.Duration(c))
	if today.After(at) {
		return today
	}
	return today.Add(day)
}

// CutDay is the day that runs at the season's end, cut at it: the one after its last cutoff, when the end is no cutoff.
func (w Window) CutDay(cutoff Cutoff) (Day, bool) {
	days := w.Days(cutoff)
	start := w.StartsAt.UTC()
	if len(days) > 0 {
		start = days[len(days)-1].EndsAt
	}
	if !start.Before(w.EndsAt) {
		return Day{}, false
	}
	return Day{Season: w.Number, Number: DayNumber(len(days) + 1), StartsAt: start, EndsAt: w.EndsAt}, true
}
