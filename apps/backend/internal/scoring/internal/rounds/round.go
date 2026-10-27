package rounds

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
)

type Country string

const Length = 24 * time.Hour

// Round is a day, or a round of its own: the window of a schedule entry whose set asked for one. Each is scored from
// the map at its end, and kept by its season, its end and whether it is of its own.
type Round struct {
	Season calendar.Number
	EndsAt time.Time
	// Finale is the round that ends the season.
	Finale bool
	Own    bool
	// StartsAt is a round of its own's; a day has none.
	StartsAt time.Time
	// Times is how many times a day's points the round scores: 1 for a day.
	Times uint32
}

// Day is the day in progress at a time: it ends at the next cutoff. None runs from the season's last cutoff to its end.
func Day(seasons calendar.Calendar, at time.Time) (Round, bool) {
	season, ok := seasons.Current(at)
	if !ok {
		return Round{}, false
	}
	end := seasons.CutoffAfter(at)
	if end.After(season.LastDayEndsAt) {
		return Round{}, false
	}
	return Round{Season: season.Number, EndsAt: end, Finale: end.Equal(season.EndsAt), Times: 1}, true
}

// Own is the round of its own of an entry's window: of the season the window starts in, and cut at that season's end.
// None once its end is past: a round never opened is not played.
func Own(seasons calendar.Calendar, times uint32, startsAt, endsAt, now time.Time) (Round, bool) {
	season, ok := seasons.Current(startsAt)
	if !ok {
		return Round{}, false
	}
	end := endsAt.UTC()
	if end.After(season.EndsAt) {
		end = season.EndsAt
	}
	if !end.After(now) {
		return Round{}, false
	}
	return Round{
		Season: season.Number, EndsAt: end, Finale: end.Equal(season.EndsAt), Own: true, StartsAt: startsAt.UTC(), Times: times,
	}, true
}

type Snapshot struct {
	Tiles uint32
	Held  map[Country]uint32
}
