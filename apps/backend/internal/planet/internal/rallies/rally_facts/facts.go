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

type Nearest interface {
	Nearest(point clicks.Vec3) (uint32, float64)
}

func New(facts Facts, borders Borders, nearest Nearest) Calls {
	return Calls{facts: facts, borders: borders, nearest: nearest}
}

// Calls tells the other modules of each rally called, with the ground and the landmass of the tile nearest its centre
// when that tile is inside the ring, which the chat names: none for a ring on open sea.
type Calls struct {
	facts   Facts
	borders Borders
	nearest Nearest
}

var _ rallies.Teller = Calls{}

func (c Calls) Called(rally rallies.Rally) {
	kept := rally.TakeSnapshot()
	called := &planetv1.RallyCalled{
		RallyId:   kept.ID.String(),
		AccountId: kept.Account.String(),
		Country:   kept.Flag,
		Point:     &planetv1.GlobePoint{X: kept.Centre.X, Y: kept.Centre.Y, Z: kept.Centre.Z},
		CalledAt:  timestamppb.New(kept.CalledAt),
	}
	if tile, arc := c.nearest.Nearest(kept.Centre); tile != 0 && arc <= kept.Radius {
		called.Ground, called.LandmassId = c.borders.CountryOf(tile), uint32(c.borders.LandmassOf(tile))
	}
	c.facts.Tell(func(time.Time) proto.Message { return called })
}
