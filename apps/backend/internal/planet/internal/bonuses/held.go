package bonuses

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var (
	ErrInvalidHolder  = errors.New("not an account id")
	ErrNothingToGrant = errors.New("a grant names no charge")
	ErrNegativeGrant  = errors.New("a grant cannot take charges away")
)

type Holder string

const NoHolder Holder = ""

func HolderOf(payer clicks.Payer) Holder {
	return Holder(payer.Account)
}

func ParseHolder(account string) (Holder, error) {
	id, err := uuid.Parse(account)
	if err != nil || id == uuid.Nil {
		return NoHolder, fmt.Errorf("%w: %q", ErrInvalidHolder, account)
	}

	return Holder(id.String()), nil
}

type Held struct {
	Refill bool
	Bomb   bool

	Enclosures int

	SpreadClicks int

	Shields int
}

func (h Held) Full(rules Rules) []Kind {
	var kinds []Kind
	if h.Refill {
		kinds = append(kinds, KindRefill)
	}
	if h.Bomb {
		kinds = append(kinds, KindBomb)
	}
	if h.Enclosures >= rules.Enclose.Held {
		kinds = append(kinds, KindEncloseClicks)
	}
	if h.SpreadClicks >= rules.Spread.Clicks {
		kinds = append(kinds, KindSpreadClicks)
	}
	if h.Shields >= rules.Shield.Held {
		kinds = append(kinds, KindShields)
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
	case KindShields:
		return h.Shields
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

func (h Held) GrantError() error {
	switch {
	case h.Enclosures < 0 || h.SpreadClicks < 0 || h.Shields < 0:
		return fmt.Errorf("%w: %+v", ErrNegativeGrant, h)
	case h.Empty():
		return ErrNothingToGrant
	}

	return nil
}

func GrantOf(kind Kind, amount int) Held {
	switch kind {
	case KindRefill:
		return Held{Refill: amount > 0}
	case KindBomb:
		return Held{Bomb: amount > 0}
	case KindEncloseClicks:
		return Held{Enclosures: amount}
	case KindSpreadClicks:
		return Held{SpreadClicks: amount}
	case KindShields:
		return Held{Shields: amount}
	}

	return Held{}
}

// Granted caps a stack at its size, and never takes from a stack held above a size made smaller.
func (h Held) Granted(kind Kind, amount int, rules Rules) Held {
	switch kind {
	case KindRefill:
		h.Refill = true
	case KindBomb:
		h.Bomb = true
	case KindEncloseClicks:
		h.Enclosures = stacked(h.Enclosures, amount, rules.Enclose.Held)
	case KindSpreadClicks:
		h.SpreadClicks = stacked(h.SpreadClicks, amount, rules.Spread.Clicks)
	case KindShields:
		h.Shields = stacked(h.Shields, amount, rules.Shield.Held)
	}

	return h
}

func stacked(held, amount, size int) int {
	return max(held, min(held+amount, size))
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

func (h Held) AfterShield() (Held, bool) {
	if h.Shields <= 0 {
		return h, false
	}
	h.Shields--

	return h, true
}
