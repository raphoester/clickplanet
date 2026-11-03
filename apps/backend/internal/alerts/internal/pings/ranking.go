package pings

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/alerts/internal/nations"
)

// A day's round, as the gamemaster names it (season-N/day-K): what its ranking is told once by.
type RoundID string

const (
	// The flags placed this high on the day are pinged, ties included.
	Podium = 3
	// The season's table is told down to this place, ties included.
	TableRows = 10
	// A ranking holds until the next round is scored, but an hour late it is old news: a new consumer group, which
	// reads the stream from its start, or a boot after a long downtime, tells no backlog.
	RankingFresh = time.Hour
)

type EndedDay struct {
	Round   RoundID
	Season  uint32
	Number  uint32
	EndedAt time.Time
}

func (d EndedDay) Stale(now time.Time) bool {
	return now.Sub(d.EndedAt) > RankingFresh
}

// A flag on a list: its rank, ties sharing one, and its points.
type Place struct {
	Flag   nations.Flag
	Rank   uint32
	Points uint32
}

// What a day scored: the points each flag of its top 10 won, best first, and the season's table once they were
// granted, best first.
type Ranking struct {
	Won   []Place
	Table []Place
}

// Nobody held a tile at the day's end.
func (r Ranking) Empty() bool {
	return len(r.Won) == 0
}
