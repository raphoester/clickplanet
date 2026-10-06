package ledger

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpsession"
)

var (
	ErrInvalidScope   = errors.New("not an address or a scope")
	ErrInvalidAccount = errors.New("not an account id")
	ErrNoCaller       = errors.New("name a scope or an account, and only one")
)

type Taking struct {
	Tile     uint32
	Scope    string
	Account  string
	Country  string
	Previous string
	At       time.Time
	Bombed   bool
}

type Position uint64

type AccountID = cpsession.AccountID

func AccountIDOf(value string) (AccountID, error) {
	id, err := uuid.Parse(value)
	if err != nil || id == uuid.Nil {
		return AccountID{}, fmt.Errorf("%w: %q", ErrInvalidAccount, value)
	}
	return AccountID(id), nil
}

type Caller struct {
	Scope   string
	Account string
}

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

func (t Taking) Cleared() bool {
	return t.Country == ""
}

func (c Caller) Made(taking Taking) bool {
	if c.Account != "" {
		return taking.Account == c.Account
	}

	return taking.Scope == c.Scope
}
