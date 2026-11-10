package refusals

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

var (
	ErrNoCaller       = errors.New("a refusal names a scope or an account, and only one")
	ErrInvalidScope   = errors.New("not an address or a scope")
	ErrInvalidAccount = errors.New("not an account id")
)

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

func callersOf(payer clicks.Payer) []Caller {
	callers := make([]Caller, 0, 2)
	if payer.Scope != "" {
		callers = append(callers, Caller{Scope: payer.Scope})
	}
	if payer.Account != "" {
		callers = append(callers, Caller{Account: payer.Account})
	}
	return callers
}

type Decision struct {
	Until time.Time
	At    time.Time
}

func Refused(until, at time.Time) Decision {
	return Decision{Until: until, At: at}
}

func Admitted(at time.Time) Decision {
	return Decision{Until: at, At: at}
}

type Refusal struct {
	until     time.Time
	decidedAt time.Time
}

// Only a later decision changes a refusal: one sent again, or after a newer one, keeps it as it is.
func (r Refusal) Decided(decision Decision) Refusal {
	if !decision.At.After(r.decidedAt) {
		return r
	}
	return Refusal{until: decision.Until, decidedAt: decision.At}
}

func (r Refusal) Standing(now time.Time) bool {
	return now.Before(r.until)
}

func (r Refusal) TakeSnapshot() Snapshot {
	return Snapshot{Until: r.until, DecidedAt: r.decidedAt}
}

type Snapshot struct {
	Until     time.Time
	DecidedAt time.Time
}

func (s Snapshot) Restore() Refusal {
	return Refusal{until: s.Until, decidedAt: s.DecidedAt}
}
