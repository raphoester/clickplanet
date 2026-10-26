package unban_player_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
)

var ErrNotBanned = errors.New("that caller has no ban to lift")

type Unbanner interface {
	LastSentence(ctx context.Context, scope, account string) (guard.Sentence, bool, error)
	Unban(ctx context.Context, scope, account string) error
	Pardon(ctx context.Context, scope, account string) (guard.Pardon, bool, error)
}

type In struct {
	Scope   string
	Account string
}

type Out struct {
	Scope         string
	Account       string
	Offence       int
	Until         time.Time
	PardonedUntil time.Time
}

func New(unbanner Unbanner) *UseCase {
	return &UseCase{unbanner: unbanner}
}

type UseCase struct {
	unbanner Unbanner
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	caller, err := bans.ParseCaller(in.Scope, in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot unban: %w", err)
	}

	sentence, found, err := u.unbanner.LastSentence(ctx, caller.Scope, caller.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the ban: %w", err)
	}
	if !found {
		return Out{}, ErrNotBanned
	}

	if err := u.unbanner.Unban(ctx, caller.Scope, caller.Account); err != nil {
		return Out{}, fmt.Errorf("failed to unban: %w", err)
	}

	pardon, _, err := u.unbanner.Pardon(ctx, caller.Scope, caller.Account)
	if err != nil {
		return Out{}, fmt.Errorf("failed to read the pardon: %w", err)
	}

	return Out{
		Scope:         caller.Scope,
		Account:       caller.Account,
		Offence:       sentence.Offence,
		Until:         sentence.Until,
		PardonedUntil: pardon.Until,
	}, nil
}
