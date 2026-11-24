package submissions

import (
	"errors"
	"fmt"
)

var ErrGuest = errors.New("only a signed-in account submits a flag: a guest's is deleted after 90 days away")

// FounderOf is the account a call came from, when it may found a flag: a flag needs a founder that lasts.
func FounderOf(account string, linked bool) (AccountID, error) {
	id, err := ParseAccountID(account)
	if err != nil {
		return AccountID{}, fmt.Errorf("%w: %w", ErrGuest, err)
	}
	if !linked {
		return AccountID{}, fmt.Errorf("%w: %s", ErrGuest, id)
	}
	return id, nil
}
