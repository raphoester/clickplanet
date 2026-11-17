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

// Gained is each kind a grant paid, and how many of it the hand kept.
func Gained(gain charges.Gain) []*planetv1.ChargeGained {
	gained := make([]*planetv1.ChargeGained, 0, len(gain.Paid))
	for _, kind := range gain.Paid {
		gained = append(gained, &planetv1.ChargeGained{Kind: charges.EncodeKind(kind), Amount: uint32(max(gain.Count(kind), 0))})
	}
	return gained
}
