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
	Sentence(ctx context.Context, scope, account string) (antibot.Sentence, bool, error)
	Unban(ctx context.Context, scope, account string) error
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

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	if !u.unbanner.Enabled() {
		return Out{}, ErrAntiBotOff
	}

	caller, err := ledger.ParseCaller(in.Scope, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot unban: %w", err)
	}

	sentence, banned, err := u.unbanner.Sentence(ctx, caller.Scope, caller.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the ban: %w", err)
	}
	if !banned {
		return Out{}, ErrNotBanned
	}

	if err := u.unbanner.Unban(ctx, caller.Scope, caller.Account); err != nil {
		return Out{}, fmt.Errorf("failed to unban: %w", err)
	}

	return Out{
		Scope:   caller.Scope,
		Account: caller.Account,
		Offence: sentence.Offence,
		Until:   sentence.Until,
	}, nil
}
