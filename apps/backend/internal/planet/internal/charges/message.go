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
		Refill: RefillRules{Held: int(m.GetRefill().GetHeld())},
		Spread: SpreadRules{
			Clicks:    int(m.GetSpread().GetClicks()),
			Rings:     int(m.GetSpread().GetRings()),
			Unlimited: m.GetSpread().GetUnlimited(),
			Off:       m.GetSpread().GetOff(),
		},
		Bomb: BombRules{Rings: m.GetBomb().GetRings(), Off: m.GetBomb().GetOff(), Held: int(m.GetBomb().GetHeld())},
		Enclose: EncloseRules{
			MaxTiles:  int(m.GetEnclose().GetMaxTiles()),
			Held:      int(m.GetEnclose().GetHeld()),
			Unlimited: m.GetEnclose().GetUnlimited(),
			Off:       m.GetEnclose().GetOff(),
		},
		Shield: ShieldRules{Held: int(m.GetShield().GetHeld()), PerTile: int(m.GetShield().GetPerTile())},
		Virus: VirusRules{
			Infections: int(m.GetVirus().GetInfections()),
			Every:      m.GetVirus().GetEvery().AsDuration(),
			Held:       int(m.GetVirus().GetHeld()),
		},
		Mine: MineRules{Held: int(m.GetMine().GetHeld()), Rings: int(m.GetMine().GetRings())},
		Rally: RallyRules{
			Held:  int(m.GetRally().GetHeld()),
			Rings: m.GetRally().GetRings(),
			Lasts: m.GetRally().GetLasts().AsDuration(),
			Cost:  m.GetRally().GetCost(),
			Off:   m.GetRally().GetOff(),
		},
	}
}

var ErrUnknownKind = errors.New("not a kind of charge")

var kinds = map[Kind]planetv1.BonusKind{
	KindRefill:        planetv1.BonusKind_BONUS_KIND_REFILL,
	KindSpreadClicks:  planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS,
	KindBomb:          planetv1.BonusKind_BONUS_KIND_BOMB,
	KindEncloseClicks: planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS,
	KindShields:       planetv1.BonusKind_BONUS_KIND_SHIELDS,
	KindVirus:         planetv1.BonusKind_BONUS_KIND_VIRUS,
	KindMine:          planetv1.BonusKind_BONUS_KIND_MINE,
	KindRally:         planetv1.BonusKind_BONUS_KIND_RALLY,
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
