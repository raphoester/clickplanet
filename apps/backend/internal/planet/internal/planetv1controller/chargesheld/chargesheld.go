// Package chargesheld is how what a caller holds becomes the ChargesHeld message. Two procedures send
// one — the claim that granted a charge, and the stream that follows every change — so the shape is
// agreed here, as clickbudget agrees the allowance's.
package chargesheld

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

// Encoder carries the sizes a client needs to use a charge, which are the same for every caller: a
// client that reloads with a bomb in hand learns the blast radius from here, having missed the claim.
type Encoder struct {
	BlastRadius       float64
	EnclosureMaxTiles int
}

func (e Encoder) Encode(held bonuses.Held) *planetv1.ChargesHeld {
	return &planetv1.ChargesHeld{
		Bomb:              held.Bomb,
		Enclose:           held.Enclose,
		SpreadClicksLeft:  uint32(held.SpreadClicks),
		BlastRadius:       e.BlastRadius,
		EnclosureMaxTiles: uint32(e.EnclosureMaxTiles),
	}
}
