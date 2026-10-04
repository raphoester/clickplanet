package rebuild_stats_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/takes"
)

type Tally interface {
	Rewind(ctx context.Context) (takes.Position, error)
}

type Executor interface {
	Execute(ctx context.Context) (Out, error)
}

type Out struct {
	From takes.Position
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
		return Out{}, fmt.Errorf("failed to put the stats back to where they start: %w", err)
	}
	return Out{From: from}, nil
}
