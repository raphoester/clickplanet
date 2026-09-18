package bonuses

import (
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Holder is the account a charge belongs to. Only an account holds charges: a charge is scarce and kept
// until it is used, so it follows the player across tabs, addresses and devices, and survives a restart in
// postgres.
// Every client mints a guest account, so this leaves out only a caller with no token at all.
type Holder string

// NoHolder is a caller with no account. It holds nothing and is offered no charge.
const NoHolder Holder = ""

// HolderOf reads the holder off the payer the throttle already derives, so the claim, the click and the
// drop cannot disagree on whose charge it is.
func HolderOf(payer clicks.Payer) Holder {
	return Holder(payer.Account)
}

// Held is what one holder has in hand: a refill and a bomb at most, a stack of enclosures and a pool of
// spread clicks, each up to its size. Nothing lapses: a charge is kept until the player uses it. It is a
// value: every change is a new Held, and the storage swaps it in.
type Held struct {
	Refill bool
	Bomb   bool

	// The enclose charges stacked, up to ChargesConfig.Enclosures. Each closes one shape.
	Enclosures int

	// The spread clicks in the pool, up to ChargesConfig.SpreadClicks.
	SpreadClicks int
}

// ChargesConfig is the part of Config the charges read: how much each pool holds.
type ChargesConfig struct {
	SpreadClicks      int
	Enclosures        int
	EnclosureMaxTiles int
}

// Full is every kind that another box would add nothing to, which the schedule does not offer: a refill or
// a bomb held, and a stack or a pool already at its size.
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

// Empty says nothing is held, so it need not be kept.
func (h Held) Empty() bool {
	return h == (Held{})
}

// Granted is what is held once a box of kind worth amount is granted. A refill or a bomb is one, and a
// second replaces the first: nobody holds two, which is what stops a stockpile being dropped all at once.
// Enclosures and spread clicks add up, to their size and no further.
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

// AfterRefill is what is held once the refill filled the bank, and whether there was one.
func (h Held) AfterRefill() (Held, bool) {
	ok := h.Refill
	h.Refill = false

	return h, ok
}

// AfterBomb is what is held once the bomb is dropped, and whether there was one to drop.
func (h Held) AfterBomb() (Held, bool) {
	ok := h.Bomb
	h.Bomb = false

	return h, ok
}

// AfterEnclose is what is held once one enclose charge closed a shape, and whether there was one.
func (h Held) AfterEnclose() (Held, bool) {
	if h.Enclosures <= 0 {
		return h, false
	}
	h.Enclosures--

	return h, true
}

// AfterSpreadClick is what is held once one spread click is spent, and whether there was one.
func (h Held) AfterSpreadClick() (Held, bool) {
	if h.SpreadClicks <= 0 {
		return h, false
	}
	h.SpreadClicks--

	return h, true
}
