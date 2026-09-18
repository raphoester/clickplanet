// Package publishing_drop_bomb tells the other modules of every bomb that went off: planet.v1.BombLanded.
package publishing_drop_bomb

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses/usecases/drop_bomb_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type UseCase interface {
	Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error)
}

// Publisher is the event bus. Publish never blocks, so a drop never waits on a listener.
type Publisher interface {
	Publish(event proto.Message)
}

// Borders says whose ground a tile sits on.
type Borders interface {
	CountryOf(tile uint32) string
}

func New(implementation UseCase, borders Borders, events Publisher, clock cptime.Clock) *Decorator {
	return &Decorator{implementation: implementation, borders: borders, events: events, clock: clock}
}

type Decorator struct {
	implementation UseCase
	borders        Borders
	events         Publisher
	clock          cptime.Clock
}

// Execute publishes a bomb once it went off. A refused drop and a dud publish nothing.
func (d *Decorator) Execute(ctx context.Context, in drop_bomb_usecase.In) (clicks.Blast, error) {
	blast, err := d.implementation.Execute(ctx, in)
	if err != nil || in.Dud {
		return blast, err //nolint:wrapcheck // a decorator adds an event, not a sentence.
	}

	landed := &planetv1.BombLanded{
		Country:  blast.CountryID,
		TileId:   blast.Tile,
		Cleared:  uint32(len(blast.Cleared)), //nolint:gosec // bounded by the map.
		LandedAt: timestamppb.New(d.clock.Now()),
	}
	if blast.Tile != 0 {
		landed.Ground = d.borders.CountryOf(blast.Tile)
	}
	d.events.Publish(landed)

	return blast, nil
}
