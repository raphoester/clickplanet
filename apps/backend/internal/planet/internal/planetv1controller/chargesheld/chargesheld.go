// Package chargesheld is how what a caller holds becomes the ChargesHeld message. Two procedures answer
// with one — the claim that granted a charge, and GetCharges — so the shape is agreed here, as clickbudget
// agrees the allowance's.
package chargesheld

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

// Encode says what is held and nothing else: how big each charge is, is GetBonusRules' to say.
func Encode(held bonuses.Held) *planetv1.ChargesHeld {
	return &planetv1.ChargesHeld{
		Bomb:             held.Bomb,
		Enclose:          held.Enclose,
		SpreadClicksLeft: uint32(held.SpreadClicks),
	}
}
