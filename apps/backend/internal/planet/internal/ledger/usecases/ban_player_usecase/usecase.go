// Package ban_player_usecase is the operator's shadow ban: the same sentence the antibot passes, on a scope or an account a person picked.
package ban_player_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

var (
	ErrNegativeDuration = errors.New("a ban cannot last a negative time")
	ErrAntiBotOff       = errors.New("antiBot is off, so there is no ban to pass")
)

type Banner interface {
	Ban(scope, account string, duration time.Duration) antibot.Sentence
	Enforcing() bool
	Enabled() bool
}

type In struct {
	// Scope is a scope as FindPlayers lists it, or any address, which is banned as its scope.
	Scope string
	// Account is an account id, banned alone. Name a scope or an account, not both.
	Account string
	// Zero takes the ladder's step for the offence.
	Duration time.Duration
}

type Out struct {
	Scope    string
	Account  string
	Offence  int
	Until    time.Time
	Enforced bool
}

func New(banner Banner) *UseCase {
	return &UseCase{banner: banner}
}

type UseCase struct {
	banner Banner
}

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	if !u.banner.Enabled() {
		return Out{}, ErrAntiBotOff
	}

	caller, err := ledger.ParseCaller(in.Scope, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot ban: %w", err)
	}
	if in.Duration < 0 {
		return Out{}, fmt.Errorf("%w: %s", ErrNegativeDuration, in.Duration)
	}

	sentence := u.banner.Ban(caller.Scope, caller.Account, in.Duration)

	return Out{
		Scope:    caller.Scope,
		Account:  caller.Account,
		Offence:  sentence.Offence,
		Until:    sentence.Until,
		Enforced: u.banner.Enforcing(),
	}, nil
}
