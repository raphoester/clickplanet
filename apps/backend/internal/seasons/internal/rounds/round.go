package rounds

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

type Country string

const Length = 24 * time.Hour

type Round struct {
	Season calendar.Number
	EndsAt time.Time
	Finale bool
}

func Current(seasons calendar.Calendar, at time.Time) (Round, bool) {
	season, ok := seasons.Current(at)
	if !ok {
		return Round{}, false
	}
	if !at.Before(season.FinaleStartsAt) {
		return Round{Season: season.Number, EndsAt: season.EndsAt, Finale: true}, true
	}

	days := int64((season.FinaleStartsAt.Sub(at) - 1) / Length)
	return Round{Season: season.Number, EndsAt: season.FinaleStartsAt.Add(-time.Duration(days * int64(Length)))}, true
}

type Census struct {
	Tiles uint32
	Held  map[Country]uint32
}
