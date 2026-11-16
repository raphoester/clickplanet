// Package planetwords is what the bonuses module says in planet's language: a charge's kind, as planet.v1 writes it. The
// bonuses module grants planet's charges, so it speaks its kinds to planet itself.
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
	bonuses.KindVirus:         planetv1.BonusKind_BONUS_KIND_VIRUS,
	bonuses.KindMine:          planetv1.BonusKind_BONUS_KIND_MINE,
}

// Kind is planet's word for a kind the bonuses module draws; unspecified for none.
func Kind(kind bonuses.Kind) planetv1.BonusKind {
	return kinds[kind]
}

// KindOf is the bonuses module's word for one of planet's kinds; false for one it never draws.
func KindOf(kind planetv1.BonusKind) (bonuses.Kind, bool) {
	for ours, theirs := range kinds {
		if theirs == kind {
			return ours, true
		}
	}
	return "", false
}
