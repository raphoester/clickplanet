package rpc_planet_rule_sets

import (
	"context"
	"fmt"
	"slices"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/metagame/internal/schedule"
)

type Planet interface {
	GetRuleSetNames(
		ctx context.Context,
		req *connect.Request[planetv1.GetRuleSetNamesRequest],
	) (*connect.Response[planetv1.GetRuleSetNamesResponse], error)
}

const askTimeout = 2 * time.Second

func New(planet Planet) *RuleSets {
	return &RuleSets{planet: planet}
}

type RuleSets struct {
	planet Planet
}

var _ schedule.Keeper = (*RuleSets)(nil)

func (r *RuleSets) Keeps(ctx context.Context, set schedule.SetName) error {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := r.planet.GetRuleSetNames(ctx, connect.NewRequest(&planetv1.GetRuleSetNamesRequest{}))
	if err != nil {
		return fmt.Errorf("failed to ask the planet module which rule sets it keeps: %w", err)
	}
	if !slices.Contains(res.Msg.GetNames(), string(set)) {
		return fmt.Errorf("planet %w: %q", schedule.ErrUnknownSet, set)
	}
	return nil
}
