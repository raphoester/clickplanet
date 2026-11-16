package rally_facts

import (
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/facts"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/rallies"
)

type Facts interface {
	Tell(fact facts.Fact)
}

type Borders interface {
	CountryOf(tile uint32) string
	LandmassOf(tile uint32) clicks.LandmassID
}

func New(facts Facts, borders Borders) Calls {
	return Calls{facts: facts, borders: borders}
}

// Calls tells the other modules of each rally called, with the ground and the landmass of its tile, which the chat names.
type Calls struct {
	facts   Facts
	borders Borders
}

var _ rallies.Teller = Calls{}

func (c Calls) Called(rally rallies.Rally) {
	kept := rally.TakeSnapshot()
	c.facts.Tell(func(time.Time) proto.Message {
		return &planetv1.RallyCalled{
			RallyId:    kept.ID.String(),
			AccountId:  kept.Account.String(),
			Country:    kept.Flag,
			TileId:     kept.Tile,
			Ground:     c.borders.CountryOf(kept.Tile),
			LandmassId: uint32(c.borders.LandmassOf(kept.Tile)),
			CalledAt:   timestamppb.New(kept.CalledAt),
		}
	})
}
