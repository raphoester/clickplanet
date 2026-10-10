package relay_fortifications_usecase

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const resubscribeAfter = time.Second

type Changes interface {
	Subscribe(ctx context.Context) (<-chan clicks.Change, error)
}

type Publisher interface {
	Publish(event proto.Message)
}

// Tells the other modules each fortify, as the tile map published it: the map is where it happened.
func New(changes Changes, events Publisher, clock cptime.Clock, logger *slog.Logger) *UseCase {
	return &UseCase{changes: changes, events: events, clock: clock, logger: logger}
}

type UseCase struct {
	changes Changes
	events  Publisher
	clock   cptime.Clock
	logger  *slog.Logger
}

func (u *UseCase) Name() string { return "planet-fortifications" }

func (u *UseCase) Run(ctx context.Context) {
	for ctx.Err() == nil {
		changes, err := u.changes.Subscribe(ctx)
		if err != nil {
			u.logger.Error("could not follow the tile map for fortifies", slog.Any("error", err))
		} else {
			u.relay(changes)
		}

		select {
		case <-ctx.Done():
		case <-time.After(resubscribeAfter):
		}
	}
}

// Returns when the tile map closes the channel: on shutdown, or because this fell behind.
func (u *UseCase) relay(changes <-chan clicks.Change) {
	for change := range changes {
		fortified := change.Fortification
		if fortified == nil {
			continue
		}

		u.events.Publish(&planetv1.Fortified{
			Country:     fortified.Country,
			Ground:      fortified.Ground,
			LandmassId:  uint32(fortified.Landmass),
			Tiles:       uint32(fortified.Tiles), //nolint:gosec // a landmass's tiles, bounded by the map.
			FortifiedAt: timestamppb.New(u.clock.Now()),
		})
	}
}
