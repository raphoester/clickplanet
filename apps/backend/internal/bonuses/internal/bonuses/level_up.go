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

// LevelUp is a level an account reached by playing, which progression told: one bonus, under progression's grant id.
// Unlocked is every kind the level unlocks, told with it.
type LevelUp struct {
	Account  Account
	Grant    GrantID
	Unlocked Unlocked
}

// Reward is drawn as a box's is, among the kinds the account may be drawn and the hand has room for. It is false when
// there is nothing the level-up could be worth.
func (l LevelUp) Reward(rules Rules, full []Kind) (Reward, bool) {
	return drawReward(rules, full, l.Unlocked, "")
}
