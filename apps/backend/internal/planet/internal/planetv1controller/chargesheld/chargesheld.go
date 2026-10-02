package chargesheld

import (
	planetv1 "github.com/raphoester/clickplanet.lol-backend/generated/proto/planet/v1"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

func Encode(held bonuses.Held) *planetv1.ChargesHeld {
	return &planetv1.ChargesHeld{
		Refill:           held.Refill,
		Bomb:             held.Bomb,
		Enclosures:       uint32(held.Enclosures),
		SpreadClicksLeft: uint32(held.SpreadClicks),
	}
}
