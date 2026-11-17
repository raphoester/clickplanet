package flagmissions

import (
	"context"
	"errors"
	"maps"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/days"
)

var ErrUnsettled = errors.New("a round of the season before the day is not scored yet")

// How long into its day a flag's draw waits for the season's rounds before the day to be scored. A take or a strike
// that finds its flag's day not drawn is told again, and dropped after about four minutes of tries, so the wait stays
// well under that; past it, the day is drawn from the table as it is.
const settleWithin = 2 * time.Minute

type StandingSnapshot struct {
	// The season's points of each flag that has some.
	Points map[string]uint32
	// Every round of the season that ran before the day is scored: its points are in.
	Settled bool
}

// The season's table as the day's draw reads it: what each flag has won in the season so far.
type Standing struct {
	points  map[string]uint32
	settled bool
}

func StandingOf(snapshot StandingSnapshot) Standing {
	return Standing{points: maps.Clone(snapshot.Points), settled: snapshot.Settled}
}

func (s Standing) Snapshot() StandingSnapshot {
	return StandingSnapshot{Points: maps.Clone(s.points), Settled: s.settled}
}

func (s Standing) Settled() bool { return s.settled }

func (s Standing) leader() uint32 {
	var most uint32
	for _, points := range s.points {
		most = max(most, points)
	}
	return most
}

// Where the season's table is read: scoring's book of the day's season.
type Standings interface {
	Standing(ctx context.Context, day days.Day) (Standing, error)
}
