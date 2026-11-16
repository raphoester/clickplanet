package chargesheld

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

func Encode(held charges.Held) *planetv1.ChargesHeld {
	return &planetv1.ChargesHeld{
		Refills:          uint32(held.Refills),
		Bombs:            uint32(held.Bombs),
		Enclosures:       uint32(held.Enclosures),
		SpreadClicksLeft: uint32(held.SpreadClicks),
		Shields:          uint32(held.Shields),
		Viruses:          uint32(held.Viruses),
		Mines:            uint32(held.Mines),
		Rallies:          uint32(held.Rallies),
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
