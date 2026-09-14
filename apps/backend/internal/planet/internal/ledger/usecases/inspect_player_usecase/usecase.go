package inspect_player_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/antibot"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/ledger"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
)

var ErrAntiBotOff = errors.New("antiBot is off, so there is nothing to inspect")

type Examiner interface {
	Examine(scope string) antibot.Examination
	Enabled() bool
}

type In struct {
	Scope string
}

func New(examiner Examiner) *UseCase {
	return &UseCase{examiner: examiner}
}

type UseCase struct {
	examiner Examiner
}

func (u *UseCase) Execute(_ context.Context, in In) (antibot.Examination, error) {
	if !u.examiner.Enabled() {
		return antibot.Examination{}, ErrAntiBotOff
	}

	scope, ok := cpipscope.Parse(in.Scope)
	if !ok {
		return antibot.Examination{}, fmt.Errorf("%w: %q", ledger.ErrInvalidScope, in.Scope)
	}

	return u.examiner.Examine(scope), nil
}
