package rpc_planet_charges_test

import (
	"context"
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/rpc_planet_charges"
)

type stubPlanet struct {
	asked []*planetv1.GrantChargeRequest
	kept  uint32
	err   error
}

func (p *stubPlanet) GrantCharge(
	_ context.Context, req *connect.Request[planetv1.GrantChargeRequest],
) (*connect.Response[planetv1.GrantChargeResponse], error) {
	p.asked = append(p.asked, req.Msg)
	if p.err != nil {
		return nil, p.err
	}
	return connect.NewResponse(&planetv1.GrantChargeResponse{Kept: p.kept}), nil
}

var account = missions.AccountID{15: 7}

func TestEachRewardIsGrantedAsItsCharge(t *testing.T) {
	for kind, wire := range map[missions.RewardKind]planetv1.BonusKind{
		missions.RewardShields:      planetv1.BonusKind_BONUS_KIND_SHIELDS,
		missions.RewardSpreadClicks: planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS,
		missions.RewardBomb:         planetv1.BonusKind_BONUS_KIND_BOMB,
	} {
		planet := &stubPlanet{kept: 3}

		kept, err := rpc_planet_charges.New(planet).Grant(t.Context(), account, missions.Reward{Kind: kind, Amount: 4})

		require.NoError(t, err)
		assert.Equal(t, uint32(3), kept)
		require.Len(t, planet.asked, 1)
		assert.True(t, proto.Equal(&planetv1.GrantChargeRequest{AccountId: account.String(), Kind: wire, Amount: 4},
			planet.asked[0]), string(kind))
	}
}

func TestAPlanetThatFailsIsAnError(t *testing.T) {
	_, err := rpc_planet_charges.New(&stubPlanet{err: errors.New("down")}).
		Grant(t.Context(), account, missions.Reward{Kind: missions.RewardBomb, Amount: 1})

	assert.Error(t, err)
}
