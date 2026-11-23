package charges

import (
	"errors"
	"fmt"
	"slices"

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

// Held is how many of each kind a hand holds: every kind a count, capped by the rules in force when it is granted.
type Held struct {
	Refills int
	Bombs   int

	Enclosures int

	SpreadClicks int

	Shields int

	Viruses int

	Mines int

	Rallies int

	Traps int
}

func (h Held) Count(kind Kind) int {
	switch kind {
	case KindRefill:
		return h.Refills
	case KindBomb:
		return h.Bombs
	case KindEncloseClicks:
		return h.Enclosures
	case KindSpreadClicks:
		return h.SpreadClicks
	case KindShields:
		return h.Shields
	case KindVirus:
		return h.Viruses
	case KindMine:
		return h.Mines
	case KindRally:
		return h.Rallies
	case KindTrap:
		return h.Traps
	}

	return 0
}

func (h Held) Empty() bool {
	return h == (Held{})
}

func (h Held) GrantError() error {
	for _, kind := range Kinds {
		if h.Count(kind) < 0 {
			return fmt.Errorf("%w: %+v", ErrNegativeGrant, h)
		}
	}
	if h.Empty() {
		return ErrNothingToGrant
	}

	return nil
}

func GrantOf(kind Kind, amount int) Held {
	var held Held
	if count := held.count(kind); count != nil {
		*count = amount
	}

	return held
}

// Granted caps a stack at its size, never takes from a stack held above a size made smaller, and keeps none of a kind
// the rules turn off.
func (h Held) Granted(kind Kind, amount int, rules Rules) Held {
	count := h.count(kind)
	if count == nil || rules.Off(kind) {
		return h
	}
	*count = max(*count, min(*count+amount, rules.Size(kind)))

	return h
}

// Gain pays the kind into a full stack too, which keeps none, and never a kind the rules turn off.
func (h Held) Gain(kind Kind, amount int, rules Rules) Gain {
	after := h.Granted(kind, amount, rules)
	if h.count(kind) == nil || rules.Off(kind) {
		return Gain{Before: h, After: after}
	}

	return Gain{Paid: []Kind{kind}, Before: h, After: after}
}

// Filling tops each kind up to its size: a full stack is paid too, and keeps none.
func (h Held) Filling(kinds []Kind, rules Rules) Gain {
	filling := Gain{Before: h, After: h}
	for _, kind := range Kinds {
		if slices.Contains(kinds, kind) {
			gain := filling.After.Gain(kind, rules.Size(kind), rules)
			filling.Paid, filling.After = append(filling.Paid, gain.Paid...), gain.After
		}
	}

	return filling
}

// count is the field that holds the kind, nil for no kind.
func (h *Held) count(kind Kind) *int {
	switch kind {
	case KindRefill:
		return &h.Refills
	case KindBomb:
		return &h.Bombs
	case KindEncloseClicks:
		return &h.Enclosures
	case KindSpreadClicks:
		return &h.SpreadClicks
	case KindShields:
		return &h.Shields
	case KindVirus:
		return &h.Viruses
	case KindMine:
		return &h.Mines
	case KindRally:
		return &h.Rallies
	case KindTrap:
		return &h.Traps
	}

	return nil
}

// spent takes one of the kind, false when none is held.
func (h Held) spent(kind Kind) (Held, bool) {
	count := h.count(kind)
	if count == nil || *count <= 0 {
		return h, false
	}
	*count--

	return h, true
}

// back gives one of the kind back, for a charge refused once spent: the same charge comes back, so no size applies.
func (h Held) back(kind Kind) Held {
	if count := h.count(kind); count != nil {
		*count++
	}

	return h
}

func (h Held) AfterRefill() (Held, bool) {
	return h.spent(KindRefill)
}

func (h Held) AfterBomb() (Held, bool) {
	return h.spent(KindBomb)
}

func (h Held) AfterVirus() (Held, bool) {
	return h.spent(KindVirus)
}

// WithVirusBack undoes AfterVirus, for a virus the map refused.
func (h Held) WithVirusBack() Held {
	return h.back(KindVirus)
}

func (h Held) AfterMine() (Held, bool) {
	return h.spent(KindMine)
}

// WithMineBack undoes AfterMine, for a mine the map refused.
func (h Held) WithMineBack() Held {
	return h.back(KindMine)
}

func (h Held) AfterTrap() (Held, bool) {
	return h.spent(KindTrap)
}

// WithTrapBack undoes AfterTrap, for a trap the map refused.
func (h Held) WithTrapBack() Held {
	return h.back(KindTrap)
}

func (h Held) AfterRally() (Held, bool) {
	return h.spent(KindRally)
}

// WithRallyBack undoes AfterRally, for a rally refused once spent.
func (h Held) WithRallyBack() Held {
	return h.back(KindRally)
}

func (h Held) AfterEnclose() (Held, bool) {
	return h.spent(KindEncloseClicks)
}

func (h Held) AfterSpreadClick() (Held, bool) {
	return h.spent(KindSpreadClicks)
}

func (h Held) AfterShield() (Held, bool) {
	return h.spent(KindShields)
}

// WithShieldBack undoes AfterShield.
func (h Held) WithShieldBack() Held {
	return h.back(KindShields)
}
