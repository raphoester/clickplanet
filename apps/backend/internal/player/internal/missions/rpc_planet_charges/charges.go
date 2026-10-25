package rpc_planet_charges

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Planet interface {
	GrantCharge(
		ctx context.Context,
		req *connect.Request[planetv1.GrantChargeRequest],
	) (*connect.Response[planetv1.GrantChargeResponse], error)
}

const askTimeout = 5 * time.Second

func New(planet Planet) Charger {
	return Charger{planet: planet}
}

type Charger struct {
	planet Planet
}

func (c Charger) Grant(ctx context.Context, account players.AccountID, reward missions.Reward) (uint32, error) {
	ctx, cancel := context.WithTimeout(ctx, askTimeout)
	defer cancel()

	res, err := c.planet.GrantCharge(ctx, connect.NewRequest(&planetv1.GrantChargeRequest{
		AccountId: account.String(), Kind: kindOf(reward.Kind), Amount: reward.Amount,
	}))
	if err != nil {
		return 0, fmt.Errorf("failed to ask planet to grant %d %s: %w", reward.Amount, reward.Kind, err)
	}
	return res.Msg.GetKept(), nil
}

func kindOf(kind missions.RewardKind) planetv1.BonusKind {
	switch kind {
	case missions.RewardShields:
		return planetv1.BonusKind_BONUS_KIND_SHIELDS
	case missions.RewardSpreadClicks:
		return planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS
	case missions.RewardBomb:
		return planetv1.BonusKind_BONUS_KIND_BOMB
	}
	return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED
}
