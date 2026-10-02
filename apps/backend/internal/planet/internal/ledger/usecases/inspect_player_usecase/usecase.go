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

type Ledger interface {
	Replay(see func(ledger.Taking)) ledger.Position
}

type In struct {
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
