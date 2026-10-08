package unban_player_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
)

var (
	ErrAntiBotOff = errors.New("antiBot is off, so there is no ban to lift")
	ErrNotBanned  = errors.New("no ban is running on that caller")
)

type Unbanner interface {
	Sentence(scope, account string) (antibot.Sentence, bool)
	Unban(scope, account string)
	Enabled() bool
}

type In struct {
	Scope   string
	Account string
}

type Out struct {
	Scope   string
	Account string
	Offence int
	Until   time.Time
}

func New(unbanner Unbanner) *UseCase {
	return &UseCase{unbanner: unbanner}
}

type UseCase struct {
	unbanner Unbanner
}

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	if !u.unbanner.Enabled() {
		return Out{}, ErrAntiBotOff
	}

	caller, err := ledger.ParseCaller(in.Scope, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot unban: %w", err)
	}

	sentence, banned := u.unbanner.Sentence(caller.Scope, caller.Account)
	if !banned {
		return Out{}, ErrNotBanned
	}

	u.unbanner.Unban(caller.Scope, caller.Account)

	return Out{
		Scope:   caller.Scope,
		Account: caller.Account,
		Offence: sentence.Offence,
		Until:   sentence.Until,
	}, nil
}
