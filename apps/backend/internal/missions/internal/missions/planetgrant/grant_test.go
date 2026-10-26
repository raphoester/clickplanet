package planetgrant_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions/planetgrant"
)

var (
	ada     = missions.AccountID{15: 1}
	bob     = missions.AccountID{15: 2}
	bank    = missions.MissionID{15: 1}
	rival   = missions.MissionID{15: 2}
	shields = missions.Reward{Kind: missions.RewardShields, Amount: 10}
)

func TestAPaymentNamesOneGrantEveryTime(t *testing.T) {
	grant := planetgrant.Of(ada, missions.Payment{Mission: bank, Reward: shields})

	assert.Equal(t, grant.GetGrantId(), planetgrant.Of(ada, missions.Payment{Mission: bank, Reward: shields}).GetGrantId())
	assert.NotEqual(t, grant.GetGrantId(), planetgrant.Of(bob, missions.Payment{Mission: bank, Reward: shields}).GetGrantId())
	assert.NotEqual(t, grant.GetGrantId(), planetgrant.Of(ada, missions.Payment{Mission: rival, Reward: shields}).GetGrantId())
	assert.Equal(t, ada.String(), grant.GetAccountId())
	assert.Equal(t, uint32(10), grant.GetAmount())
}

func TestEachRewardIsItsCharge(t *testing.T) {
	for kind, wire := range map[missions.RewardKind]planetv1.BonusKind{
		missions.RewardShields:      planetv1.BonusKind_BONUS_KIND_SHIELDS,
		missions.RewardSpreadClicks: planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS,
		missions.RewardBomb:         planetv1.BonusKind_BONUS_KIND_BOMB,
	} {
		assert.Equal(t, wire, planetgrant.KindOf(kind), string(kind))
	}
}
