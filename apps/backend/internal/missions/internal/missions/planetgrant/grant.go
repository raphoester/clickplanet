package planetgrant

import (
	"github.com/google/uuid"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/missions/internal/missions"
)

var grantSpace = uuid.MustParse("4d697373-696f-4e73-8000-6772616e7473")

// A payment told twice names the same grant, which planet grants once.
func Of(account missions.AccountID, payment missions.Payment) *planetv1.GrantCharges {
	name := make([]byte, 0, len(account)+len(payment.Mission))
	name = append(name, account[:]...)
	name = append(name, payment.Mission[:]...)

	return &planetv1.GrantCharges{
		GrantId:   uuid.NewSHA1(grantSpace, name).String(),
		AccountId: account.String(),
		Kind:      KindOf(payment.Reward.Kind),
		Amount:    payment.Reward.Amount,
	}
}

func KindOf(kind missions.RewardKind) planetv1.BonusKind {
	switch kind {
	case missions.RewardShields:
		return planetv1.BonusKind_BONUS_KIND_SHIELDS
	case missions.RewardSpreadClicks:
		return planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS
	case missions.RewardBomb:
		return planetv1.BonusKind_BONUS_KIND_BOMB
	case missions.RewardEncloseClicks:
		return planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS
	}
	return planetv1.BonusKind_BONUS_KIND_UNSPECIFIED
}
