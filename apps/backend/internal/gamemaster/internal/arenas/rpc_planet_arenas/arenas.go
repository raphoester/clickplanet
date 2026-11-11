package rpc_planet_arenas

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments/rpcrefusal"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
)

type Planet interface {
	GetRuleSet(ctx context.Context, req *connect.Request[planetv1.GetRuleSetRequest],
	) (*connect.Response[planetv1.GetRuleSetResponse], error)
	OpenArena(ctx context.Context, req *connect.Request[planetv1.OpenArenaRequest],
	) (*connect.Response[planetv1.OpenArenaResponse], error)
	CloseArena(ctx context.Context, req *connect.Request[planetv1.CloseArenaRequest],
	) (*connect.Response[planetv1.CloseArenaResponse], error)
}

const (
	askTimeout = 2 * time.Second
	// A swap writes the whole map; planet goes on when the call gives up, and the next call reads its answer.
	swapTimeout = 30 * time.Second
)

func New(planet Planet) *Arenas {
	return &Arenas{planet: planet}
}

// Arenas reads whether a set makes an arena battle, and opens and closes arenas, over planet's internal service.
type Arenas struct {
	planet Planet
}

// Arena is false for a set planet keeps none of: planet refuses it and plays on, on the map as it is.
func (s *Arenas) Arena(ctx context.Context, set schedule.SetName) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := s.planet.GetRuleSet(ctx, connect.NewRequest(&planetv1.GetRuleSetRequest{Name: string(set)}))
	switch {
	case connect.CodeOf(err) == connect.CodeNotFound:
		return false, nil
	case err != nil:
		return false, fmt.Errorf("failed to ask planet whether %q is fought in an arena: %w", set, err)
	}
	return res.Msg.GetPlayed().GetArena().GetOn(), nil
}

func (s *Arenas) Ask(ctx context.Context, command moments.ArenaCommand) error {
	ctx, cancel := context.WithTimeout(ctx, swapTimeout)
	defer cancel()

	if err := s.ask(ctx, command); err != nil {
		return rpcrefusal.Of(fmt.Errorf("failed to %s %s: %w", command.Action, command.Arena, err)) //nolint:wrapcheck // wrapped here.
	}
	return nil
}

func (s *Arenas) ask(ctx context.Context, command moments.ArenaCommand) error {
	switch command.Action {
	case moments.OpenArena:
		_, err := s.planet.OpenArena(ctx, connect.NewRequest(&planetv1.OpenArenaRequest{
			ArenaId: string(command.Arena), EndsAt: timestamppb.New(command.EndsAt), RuleSet: string(command.Set),
		}))
		return err //nolint:wrapcheck // Ask names the arena.
	case moments.CloseArena:
		_, err := s.planet.CloseArena(ctx, connect.NewRequest(&planetv1.CloseArenaRequest{ArenaId: string(command.Arena)}))
		return err //nolint:wrapcheck // Ask names the arena.
	}
	return fmt.Errorf("planet has no arena call to %s", command.Action)
}
