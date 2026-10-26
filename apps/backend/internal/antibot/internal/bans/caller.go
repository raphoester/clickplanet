package bans

import (
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

var (
	ErrInvalidScope   = errors.New("not an address or a scope")
	ErrInvalidAccount = errors.New("not an account id")
	ErrNoCaller       = errors.New("name a scope or an account, and only one")
)

// An address is banned as its scope.
func ParseCaller(scope, account string) (Caller, error) {
	switch {
	case (scope == "") == (account == ""):
		return Caller{}, ErrNoCaller
	case account != "":
		id, err := uuid.Parse(account)
		if err != nil {
			return Caller{}, fmt.Errorf("%w: %q", ErrInvalidAccount, account)
		}
		return Caller{Account: id.String()}, nil
	}

	parsed, ok := cpipscope.Parse(scope)
	if !ok {
		return Caller{}, fmt.Errorf("%w: %q", ErrInvalidScope, scope)
	}
	return Caller{Scope: parsed}, nil
}
