package rpc_planet_rules

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/cards"
)

type Planet interface {
	GetRuleSetNames(context.Context, *connect.Request[planetv1.GetRuleSetNamesRequest]) (*connect.Response[planetv1.GetRuleSetNamesResponse], error)
	GetRuleSet(context.Context, *connect.Request[planetv1.GetRuleSetRequest]) (*connect.Response[planetv1.GetRuleSetResponse], error)
}

func New(planet Planet) *Adapter {
	return &Adapter{planet: planet}
}

// The sets planet keeps, and the values each plays, in planet's words.
type Adapter struct {
	planet Planet
}

func (a *Adapter) Names(ctx context.Context) ([]string, error) {
	res, err := a.planet.GetRuleSetNames(ctx, connect.NewRequest(&planetv1.GetRuleSetNamesRequest{}))
	if err != nil {
		return nil, fmt.Errorf("failed to ask planet which rule sets it keeps: %w", err)
	}
	return res.Msg.GetNames(), nil
}

// Played is cards.ErrUnknownSet for a set planet does not keep.
func (a *Adapter) Played(ctx context.Context, set string) (*planetv1.Rules, error) {
	res, err := a.planet.GetRuleSet(ctx, connect.NewRequest(&planetv1.GetRuleSetRequest{Name: set}))
	switch {
	case connect.CodeOf(err) == connect.CodeNotFound:
		return nil, fmt.Errorf("%w: %q", cards.ErrUnknownSet, set)
	case err != nil:
		return nil, fmt.Errorf("failed to ask planet what %q plays: %w", set, err)
	}
	if res.Msg.GetPlayed() == nil {
		return nil, errors.New("planet answered no values")
	}
	return res.Msg.GetPlayed(), nil
}
