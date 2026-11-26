package submissions

import (
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/flags/internal/catalog"
)

var ErrGuest = errors.New("only a signed-in account submits a flag: a guest's is deleted after 90 days away")

// FounderOf is the account a call came from, when it may found a flag: a flag needs a founder that lasts.
func FounderOf(account string, linked bool) (catalog.AccountID, error) {
	id, err := catalog.ParseAccountID(account)
	if err != nil {
		return catalog.AccountID{}, fmt.Errorf("%w: %w", ErrGuest, err)
	}
	if !linked {
		return catalog.AccountID{}, fmt.Errorf("%w: %s", ErrGuest, id)
	}
	return id, nil
}
