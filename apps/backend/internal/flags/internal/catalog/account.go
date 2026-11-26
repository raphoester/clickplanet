package catalog

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrInvalidAccount = errors.New("not the id of an account")

// AccountID is a player's account: the founder of a submission, the holder of a flag.
type AccountID uuid.UUID

func ParseAccountID(s string) (AccountID, error) {
	id, err := uuid.Parse(s)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, s)
	}
	return AccountID(id), nil
}

func (id AccountID) String() string {
	return uuid.UUID(id).String()
}
