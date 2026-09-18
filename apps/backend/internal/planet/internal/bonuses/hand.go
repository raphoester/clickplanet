package bonuses

import (
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// Holder is the account a charge belongs to. Only an account holds charges: a charge is scarce and kept for
// a day, so it follows the player across tabs, addresses and devices, and survives a restart in postgres.
// Every client mints a guest account, so this leaves out only a caller with no token at all.
type Holder string

// NoHolder is a caller with no account. It holds nothing and is offered no charge.
const NoHolder Holder = ""

// HolderOf reads the holder off the payer the throttle already derives, so the claim, the click and the
// drop cannot disagree on whose charge it is.
func HolderOf(payer clicks.Payer) Holder {
	return Holder(payer.Account)
}

// Held is what one holder has in hand: at most one charge of each kind.
type Held struct {
	Refill  bool
	Bomb    bool
	Enclose bool

	// The clicks left on a spread charge. Zero is no spread charge.
	SpreadClicks int
}

// Kinds is every kind held, which the schedule does not offer again until it is spent.
func (h Held) Kinds() []Kind {
	var kinds []Kind
	if h.Refill {
		kinds = append(kinds, KindRefill)
	}
	if h.Bomb {
		kinds = append(kinds, KindBomb)
	}
	if h.Enclose {
		kinds = append(kinds, KindEncloseClicks)
	}
	if h.SpreadClicks > 0 {
		kinds = append(kinds, KindSpreadClicks)
	}

	return kinds
}

// ChargesConfig is the part of Config the charges read.
type ChargesConfig struct {
	TTL               time.Duration
	SpreadClicks      int
	EnclosureMaxTiles int
}

// Hand is one holder's charges as they are kept: each with the moment it lapses, a zero time for none. It
// is a value: every change is a new Hand, and the storage swaps it in.
type Hand struct {
	Refill  time.Time
	Bomb    time.Time
	Enclose time.Time
	Spread  time.Time

	SpreadClicks int
}

// Held is what the hand holds at now: a charge past its time is not held.
func (h Hand) Held(now time.Time) Held {
	held := Held{Refill: now.Before(h.Refill), Bomb: now.Before(h.Bomb), Enclose: now.Before(h.Enclose)}
	if now.Before(h.Spread) {
		held.SpreadClicks = h.SpreadClicks
	}

	return held
}

// Empty says the hand holds nothing at now, so it need not be kept.
func (h Hand) Empty(now time.Time) bool {
	return h.Held(now) == (Held{})
}

// Granted is the hand with one charge of kind, lapsing config.TTL from now. A second of a kind held
// replaces it rather than adding to it: nobody holds two, which is what stops a stockpile being dropped
// all at once.
func (h Hand) Granted(kind Kind, now time.Time, config ChargesConfig) Hand {
	lapses := now.Add(config.TTL)

	switch kind {
	case KindRefill:
		h.Refill = lapses
	case KindBomb:
		h.Bomb = lapses
	case KindEncloseClicks:
		h.Enclose = lapses
	case KindSpreadClicks:
		h.Spread = lapses
		h.SpreadClicks = config.SpreadClicks
	}

	return h
}

// AfterRefill is the hand once its refill filled the bank, and whether there was one.
func (h Hand) AfterRefill(now time.Time) (Hand, bool) {
	if !now.Before(h.Refill) {
		return h, false
	}
	h.Refill = time.Time{}

	return h, true
}

// AfterBomb is the hand once its bomb is dropped, and whether there was one to drop.
func (h Hand) AfterBomb(now time.Time) (Hand, bool) {
	if !now.Before(h.Bomb) {
		return h, false
	}
	h.Bomb = time.Time{}

	return h, true
}

// AfterEnclose is the hand once its enclose charge closed a shape, and whether there was one.
func (h Hand) AfterEnclose(now time.Time) (Hand, bool) {
	if !now.Before(h.Enclose) {
		return h, false
	}
	h.Enclose = time.Time{}

	return h, true
}

// AfterSpreadClick is the hand once one click of its spread charge is spent, and whether there was one.
func (h Hand) AfterSpreadClick(now time.Time) (Hand, bool) {
	if h.SpreadClicks <= 0 || !now.Before(h.Spread) {
		return h, false
	}
	h.SpreadClicks--
	if h.SpreadClicks == 0 {
		h.Spread = time.Time{}
	}

	return h, true
}
