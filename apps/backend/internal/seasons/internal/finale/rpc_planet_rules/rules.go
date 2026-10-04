package rpc_planet_rules

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1/planetv1connect"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/finale"
)

type Dialer interface {
	Dial() (connect.HTTPClient, string, error)
}

const askTimeout = 2 * time.Second

func New(dial Dialer) *Rules {
	return &Rules{dial: dial}
}

type Rules struct {
	dial Dialer
}

func (r *Rules) Set(ctx context.Context, switches finale.Switches) error {
	client, baseURL, err := r.dial.Dial()
	if err != nil {
		return fmt.Errorf("failed to reach the planet module: %w", err)
	}

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

	if _, err := planetv1connect.NewInternalServiceClient(client, baseURL).SetRules(ctx, connect.NewRequest(req)); err != nil {
		return fmt.Errorf("failed to call planet.v1.InternalService/SetRules: %w", err)
	}

	return nil
}
