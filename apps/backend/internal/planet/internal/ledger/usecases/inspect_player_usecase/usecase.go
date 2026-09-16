package inspect_player_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

var ErrAntiBotOff = errors.New("antiBot is off, so there is nothing to inspect")

type Examiner interface {
	Examine(scope, account string) antibot.Examination
	Enabled() bool
}

// Ledger says which scope an account played from last: the watchdogs judge scopes.
type Ledger interface {
	Replay(see func(ledger.Taking)) ledger.Position
}

type In struct {
	// Scope is any address, read as its scope, or Account an account id: one of the two.
	Scope   string
	Account string
}

func New(examiner Examiner, ledger Ledger) *UseCase {
	return &UseCase{examiner: examiner, ledger: ledger}
}

type UseCase struct {
	examiner Examiner
	ledger   Ledger
}

// Execute reads an account on the scope of its latest take, with the bans on both. An account with no
// take inside the retention is read on its bans alone.
func (u *UseCase) Execute(_ context.Context, in In) (antibot.Examination, error) {
	if !u.examiner.Enabled() {
		return antibot.Examination{}, ErrAntiBotOff
	}

	caller, err := ledger.ParseCaller(in.Scope, in.Account)
	if err != nil {
		return antibot.Examination{}, fmt.Errorf("cannot inspect: %w", err)
	}

	scope := caller.Scope
	if caller.Account != "" {
		u.ledger.Replay(func(taking ledger.Taking) {
			if caller.Made(taking) {
				scope = taking.Scope
			}
		})
	}

	return u.examiner.Examine(scope, caller.Account), nil
}
