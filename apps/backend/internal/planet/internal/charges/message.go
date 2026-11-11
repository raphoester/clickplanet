package charges

import (
	"errors"
	"fmt"

	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cprules"
)

// Part is the wallet's rules in a set (charges).
var Part = cprules.PartOf[*planetv1.Rules, Rules]{Key: "charges", Read: func(m *planetv1.Rules) Rules {
	return RulesOf(m.GetCharges())
}}

func RulesOf(m *planetv1.ChargeRules) Rules {
	return Rules{
		Spread: SpreadRules{Clicks: int(m.GetSpread().GetClicks())},
		Bomb:   BombRules{Rings: m.GetBomb().GetRings(), Off: m.GetBomb().GetOff()},
		Enclose: EncloseRules{
			MaxTiles:  int(m.GetEnclose().GetMaxTiles()),
			Held:      int(m.GetEnclose().GetHeld()),
			Unlimited: m.GetEnclose().GetUnlimited(),
		},
		Shield: ShieldRules{Held: int(m.GetShield().GetHeld()), PerTile: int(m.GetShield().GetPerTile())},
	}
}

var ErrUnknownKind = errors.New("not a kind of charge")

var kinds = map[Kind]planetv1.BonusKind{
	KindRefill:        planetv1.BonusKind_BONUS_KIND_REFILL,
	KindSpreadClicks:  planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS,
	KindBomb:          planetv1.BonusKind_BONUS_KIND_BOMB,
	KindEncloseClicks: planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS,
	KindShields:       planetv1.BonusKind_BONUS_KIND_SHIELDS,
}

// KindOf is the wallet's kind for planet.v1's word for it.
func KindOf(kind planetv1.BonusKind) (Kind, error) {
	for ours, theirs := range kinds {
		if theirs == kind {
			return ours, nil
		}
	}
	return "", fmt.Errorf("%w: %v", ErrUnknownKind, kind)
}

// EncodeKind is planet.v1's word for a kind.
func EncodeKind(kind Kind) planetv1.BonusKind {
	return kinds[kind]
}
