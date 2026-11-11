package bonuses

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var ErrInvalidAccount = errors.New("not an account id")

// Account is the account a click token names: whom an offer pays. A caller with none is NoAccount, and is paid nothing.
type Account string

const NoAccount Account = ""

func ParseAccount(account string) (Account, error) {
	id, err := uuid.Parse(account)
	if err != nil || id == uuid.Nil {
		return NoAccount, fmt.Errorf("%w: %q", ErrInvalidAccount, account)
	}

	return Account(id.String()), nil
}

// Caller is who a request comes from, as its click token and its address say. The offers read no request: the edge
// that serves one hands its caller over.
type Caller struct {
	Scope   string
	Account Account
	Linked  bool
}
