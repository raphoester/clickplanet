package bonuses

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrInvalidGrant = errors.New("not a grant id")

// GrantIDOf reads a grant id another module named a payment by.
func GrantIDOf(grant string) (GrantID, error) {
	id, err := uuid.Parse(grant)
	if err != nil || id == uuid.Nil {
		return GrantID{}, fmt.Errorf("%w: %q", ErrInvalidGrant, grant)
	}
	return GrantID(id), nil
}

// LevelUp is a level progression told, with every kind the level unlocks.
type LevelUp struct {
	Account  Account
	Grant    GrantID
	Unlocked Unlocked
}

// Stacks is every kind the account may be drawn, whatever the hand holds. It is false when there is none.
func (l LevelUp) Stacks(rules Rules) (Stacks, bool) {
	drawable := rules.drawable(l.Unlocked)
	if drawable.Empty() {
		return Stacks{}, false
	}

	kinds := make([]Kind, 0, drawable.Len())
	for _, kind := range Kinds {
		if drawable.Contains(kind) {
			kinds = append(kinds, kind)
		}
	}
	return Stacks{Grant: l.Grant, Account: l.Account, Kinds: kinds}, true
}
