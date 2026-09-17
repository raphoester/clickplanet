// Package get_roster_usecase reads who is playing.
package get_roster_usecase

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Visits interface {
	Visits() []presence.Visit
}

type UseCase struct {
	visits Visits
	clock  cptime.Clock
}

func New(visits Visits, clock cptime.Clock) *UseCase {
	return &UseCase{visits: visits, clock: clock}
}

// Execute builds the roster on every call: it is a sort of the few hundred visits in memory.
func (u *UseCase) Execute() []presence.Entry {
	return presence.RosterOf(u.visits.Visits(), u.clock.Now())
}
