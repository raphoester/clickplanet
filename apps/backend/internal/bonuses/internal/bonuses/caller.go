package bonuses

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrInvalidAccount = errors.New("not an account id")

// Account is the account a win pays.
type Account string

const NoAccount Account = ""

func ParseAccount(account string) (Account, error) {
	id, err := uuid.Parse(account)
	if err != nil || id == uuid.Nil {
		return NoAccount, fmt.Errorf("%w: %q", ErrInvalidAccount, account)
	}

	return Account(id.String()), nil
}
