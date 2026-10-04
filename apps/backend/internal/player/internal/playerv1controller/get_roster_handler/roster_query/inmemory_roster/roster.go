package inmemory_roster

import (
	playerv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/player/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/get_roster_handler/roster_query"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/playerv1controller/playermessage"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/presence"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Visits interface {
	Visits() []presence.Visit
}

func New(visits Visits, clock cptime.Clock) *Roster {
	return &Roster{visits: visits, clock: clock}
}

type Roster struct {
	visits Visits
	clock  cptime.Clock
}

var _ roster_query.Lines = (*Roster)(nil)

func (r *Roster) Lines() []*playerv1.RosterEntry {
	return playermessage.RosterEntries(presence.RosterOf(r.visits.Visits(), r.clock.Now()))
}
