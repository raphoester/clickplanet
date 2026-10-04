package roster_query

import (
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Visits interface {
	Visits() []presence.Visit
}

func NewMemoryQuery(visits Visits, clock cptime.Clock) *MemoryQuery {
	return &MemoryQuery{visits: visits, clock: clock}
}

type MemoryQuery struct {
	visits Visits
	clock  cptime.Clock
}

func (q *MemoryQuery) Roster() *playerv1.GetRosterResponse {
	return &playerv1.GetRosterResponse{
		Entries: playermessage.RosterEntries(presence.RosterOf(q.visits.Visits(), q.clock.Now())),
	}
}
