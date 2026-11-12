package warn_player_usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
)

var ErrNotEnforced = errors.New("the antibot does not enforce its bans (the rule bans.enforce is off), so it warns nobody")

type Warner interface {
	Enforcing() bool
	Warn(ctx context.Context, account string) (guard.Warning, bool, error)
	Warning(ctx context.Context, account string) (guard.Warning, bool, error)
}

type In struct {
	Account string
}

type Out struct {
	Account        string
	WarnedAt       time.Time
	AlreadyWarned  bool
	AcknowledgedAt time.Time
}

func New(warner Warner) *UseCase {
	return &UseCase{warner: warner}
}

type UseCase struct {
	warner Warner
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	account, err := bans.ParseAccount(in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot warn: %w", err)
	}
	if !u.warner.Enforcing() {
		return Out{Account: account}, ErrNotEnforced
	}

	given, warned, err := u.warner.Warn(ctx, account)
	if err != nil {
		return Out{Account: account}, fmt.Errorf("failed to warn: %w", err)
	}
	if warned {
		return Out{Account: account, WarnedAt: given.At}, nil
	}

	first, found, err := u.warner.Warning(ctx, account)
	if err != nil {
		return Out{Account: account}, fmt.Errorf("failed to read the warning: %w", err)
	}
	// None given and none kept: the set played since the check stopped enforcing.
	if !found {
		return Out{Account: account}, ErrNotEnforced
	}

	return Out{Account: account, WarnedAt: first.At, AlreadyWarned: true, AcknowledgedAt: first.AcknowledgedAt}, nil
}
