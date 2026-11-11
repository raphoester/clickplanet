package rpc_planet_resets

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/rpcrefusal"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

type Planet interface {
	ResetMap(ctx context.Context, req *connect.Request[planetv1.ResetMapRequest],
	) (*connect.Response[planetv1.ResetMapResponse], error)
}

// The wipe writes the whole map; planet goes on when the call gives up, and the next call reads its answer.
const resetTimeout = 30 * time.Second

func New(planet Planet) Resets {
	return Resets{planet: planet}
}

// Resets asks planet, in its words, to reset the map as a season starts: the reset is named by the season's book.
type Resets struct {
	planet Planet
}

func (r Resets) Reset(ctx context.Context, starts season.Window) error {
	ctx, cancel := context.WithTimeout(ctx, resetTimeout)
	defer cancel()

	id := fmt.Sprintf("season-%d", starts.Number)
	if _, err := r.planet.ResetMap(ctx, connect.NewRequest(&planetv1.ResetMapRequest{
		ResetId: id, DecidedAt: timestamppb.New(starts.StartsAt),
	})); err != nil {
		return rpcrefusal.Of(fmt.Errorf("failed to reset the map for %q: %w", id, err)) //nolint:wrapcheck // wrapped here.
	}
	return nil
}
