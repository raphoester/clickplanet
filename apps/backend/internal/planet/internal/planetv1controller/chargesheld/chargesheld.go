package chargesheld

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

func Encode(held charges.Held) *planetv1.ChargesHeld {
	return &planetv1.ChargesHeld{
		Refill:           held.Refill,
		Bomb:             held.Bomb,
		Enclosures:       uint32(held.Enclosures),
		SpreadClicksLeft: uint32(held.SpreadClicks),
		Shields:          uint32(held.Shields),
	}
}

// Gained is each kind a grant added to the hand, and how many.
func Gained(gain charges.Gain) []*planetv1.ChargeGained {
	var gained []*planetv1.ChargeGained
	for _, kind := range charges.Kinds {
		if added := gain.Count(kind); added > 0 {
			gained = append(gained, &planetv1.ChargeGained{Kind: charges.EncodeKind(kind), Amount: uint32(added)})
		}
	}
	return gained
}
