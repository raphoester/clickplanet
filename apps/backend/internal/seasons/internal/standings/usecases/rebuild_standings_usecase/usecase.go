package rebuild_standings_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type Tally interface {
	Rewind(ctx context.Context) (standings.Position, error)
}

type Executor interface {
	Execute(ctx context.Context) (Out, error)
}

type Out struct {
	From standings.Position
}

type UseCase struct {
	tally Tally
}

var _ Executor = (*UseCase)(nil)

func New(tally Tally) *UseCase {
	return &UseCase{tally: tally}
}

func (u *UseCase) Execute(ctx context.Context) (Out, error) {
	from, err := u.tally.Rewind(ctx)
	if err != nil {
		return Out{}, fmt.Errorf("failed to put the standings back to the first take: %w", err)
	}
	return Out{From: from}, nil
}
