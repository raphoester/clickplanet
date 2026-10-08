package rpc_planet_rules

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale"
)

type Planet interface {
	SetRules(
		ctx context.Context,
		req *connect.Request[planetv1.SetRulesRequest],
	) (*connect.Response[planetv1.SetRulesResponse], error)
}

const askTimeout = 2 * time.Second

func New(planet Planet) *Rules {
	return &Rules{planet: planet}
}

type Rules struct {
	planet Planet
}

func (r *Rules) Set(ctx context.Context, switches finale.Switches) error {
	req := &planetv1.SetRulesRequest{
		RefillMultiplier: switches.RefillMultiplier(),
		BoxIntervalMs:    switches.BoxInterval().Milliseconds(),
		Frozen:           switches.Frozen(),
	}
	if tag, madeBefore, giving := switches.Gift(); giving {
		req.Gift = &planetv1.Gift{Tag: tag}
		if !madeBefore.IsZero() {
			req.Gift.AccountsMadeBeforeUnixMs = madeBefore.UnixMilli()
		}
	}

	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	if _, err := r.planet.SetRules(ctx, connect.NewRequest(req)); err != nil {
		return fmt.Errorf("failed to call planet.v1.InternalService/SetRules: %w", err)
	}

	return nil
}
