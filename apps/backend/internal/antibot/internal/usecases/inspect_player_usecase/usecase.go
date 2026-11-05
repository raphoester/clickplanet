package inspect_player_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/bans"
	"github.com/raphoester/clickplanet.lol-backend/internal/antibot/internal/guard"
)

type Examiner interface {
	Examine(ctx context.Context, scope, account string) (guard.Examination, error)
}

type In struct {
	Scope   string
	Account string
}

func New(examiner Examiner) *UseCase {
	return &UseCase{examiner: examiner}
}

type UseCase struct {
	examiner Examiner
}

func (u *UseCase) Execute(ctx context.Context, in In) (guard.Examination, error) {
	caller, err := bans.ParseCaller(in.Scope, in.Account)
	if err != nil {
		return guard.Examination{}, fmt.Errorf("cannot inspect: %w", err)
	}

	examination, err := u.examiner.Examine(ctx, caller.Scope, caller.Account)
	if err != nil {
		return guard.Examination{}, fmt.Errorf("failed to inspect: %w", err)
	}

	return examination, nil
}
