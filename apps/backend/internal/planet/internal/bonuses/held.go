package bonuses

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

type Holder string

const NoHolder Holder = ""

func HolderOf(payer clicks.Payer) Holder {
	return Holder(payer.Account)
}

type Held struct {
	Refill bool
	Bomb   bool

	Enclosures int

	SpreadClicks int
}

type ChargesConfig struct {
	SpreadClicks      int
	Enclosures        int
	EnclosureMaxTiles int
}

func (h Held) Full(config ChargesConfig) []Kind {
	var kinds []Kind
	if h.Refill {
		kinds = append(kinds, KindRefill)
	}
	if h.Bomb {
		kinds = append(kinds, KindBomb)
	}
	if h.Enclosures >= config.Enclosures {
		kinds = append(kinds, KindEncloseClicks)
	}
	if h.SpreadClicks >= config.SpreadClicks {
		kinds = append(kinds, KindSpreadClicks)
	}

	return kinds
}

func (h Held) Count(kind Kind) int {
	switch kind {
	case KindRefill:
		return oneIf(h.Refill)
	case KindBomb:
		return oneIf(h.Bomb)
	case KindEncloseClicks:
		return h.Enclosures
	case KindSpreadClicks:
		return h.SpreadClicks
	}

	return 0
}

func oneIf(held bool) int {
	if held {
		return 1
	}

	return 0
}

func (h Held) Empty() bool {
	return h == (Held{})
}

func (h Held) Granted(kind Kind, amount int, config ChargesConfig) Held {
	switch kind {
	case KindRefill:
		h.Refill = true
	case KindBomb:
		h.Bomb = true
	case KindEncloseClicks:
		h.Enclosures = min(h.Enclosures+amount, config.Enclosures)
	case KindSpreadClicks:
		h.SpreadClicks = min(h.SpreadClicks+amount, config.SpreadClicks)
	}

	return h
}

func (h Held) AfterRefill() (Held, bool) {
	ok := h.Refill
	h.Refill = false

	return h, ok
}

func (h Held) AfterBomb() (Held, bool) {
	ok := h.Bomb
	h.Bomb = false

	return h, ok
}

func (h Held) AfterEnclose() (Held, bool) {
	if h.Enclosures <= 0 {
		return h, false
	}
	h.Enclosures--

	return h, true
}

func (h Held) AfterSpreadClick() (Held, bool) {
	if h.SpreadClicks <= 0 {
		return h, false
	}
	h.SpreadClicks--

	return h, true
}
