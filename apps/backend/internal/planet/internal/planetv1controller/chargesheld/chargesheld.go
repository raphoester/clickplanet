package chargesheld

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
)

// EncodeHand encodes a hand the offers answered: the wallet's, carried through bonuses as it was.
func EncodeHand(hand bonuses.Hand) *planetv1.ChargesHeld {
	return Encode(charges.Held(hand))
}

func Encode(held charges.Held) *planetv1.ChargesHeld {
	return &planetv1.ChargesHeld{
		Refill:           held.Refill,
		Bomb:             held.Bomb,
		Enclosures:       uint32(held.Enclosures),
		SpreadClicksLeft: uint32(held.SpreadClicks),
		Shields:          uint32(held.Shields),
	}
}
