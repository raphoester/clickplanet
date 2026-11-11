// Package planetwords is what the offers say in planet's language: a charge's kind and a hand, as planet.v1 writes
// them. The offers grant planet's charges, so they speak its kinds on the wire and to planet itself.
package planetwords

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/bonuses/internal/bonuses"
)

var kinds = map[bonuses.Kind]planetv1.BonusKind{
	bonuses.KindRefill:        planetv1.BonusKind_BONUS_KIND_REFILL,
	bonuses.KindSpreadClicks:  planetv1.BonusKind_BONUS_KIND_SPREAD_CLICKS,
	bonuses.KindBomb:          planetv1.BonusKind_BONUS_KIND_BOMB,
	bonuses.KindEncloseClicks: planetv1.BonusKind_BONUS_KIND_ENCLOSE_CLICKS,
	bonuses.KindShields:       planetv1.BonusKind_BONUS_KIND_SHIELDS,
}

// Kind is planet's word for a kind the offers draw; unspecified for none.
func Kind(kind bonuses.Kind) planetv1.BonusKind {
	return kinds[kind]
}

// KindOf is the offers' word for one of planet's kinds; false for one they never draw.
func KindOf(kind planetv1.BonusKind) (bonuses.Kind, bool) {
	for ours, theirs := range kinds {
		if theirs == kind {
			return ours, true
		}
	}
	return "", false
}

func Hand(hand bonuses.Hand) *planetv1.ChargesHeld {
	return &planetv1.ChargesHeld{
		Refill:     hand.Refill,
		Bomb:       hand.Bomb,
		Enclosures: uint32(hand.Enclosures), //nolint:gosec // a stack's count, never negative.
		//nolint:gosec // a pool's count, never negative.
		SpreadClicksLeft: uint32(hand.SpreadClicks),
		Shields:          uint32(hand.Shields), //nolint:gosec // a stack's count, never negative.
	}
}

func HandOf(held *planetv1.ChargesHeld) bonuses.Hand {
	return bonuses.Hand{
		Refill:       held.GetRefill(),
		Bomb:         held.GetBomb(),
		Enclosures:   int(held.GetEnclosures()),
		SpreadClicks: int(held.GetSpreadClicksLeft()),
		Shields:      int(held.GetShields()),
	}
}
