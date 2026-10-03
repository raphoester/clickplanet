package subscriptions

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

type AccountID = cpsession.AccountID

var ErrInvalidAccount = errors.New("not an account id")

func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}

type Account struct {
	ID        AccountID
	Linked    bool
	Addresses []Address
}

func (a Account) Verified(address Address) bool {
	return slices.Contains(a.Addresses, address)
}

func (a Account) Suggestion() Address {
	if len(a.Addresses) == 0 {
		return ""
	}
	return a.Addresses[0]
}
