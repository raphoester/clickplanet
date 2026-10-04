package marking_record_take

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type UseCase interface {
	Execute(ctx context.Context, take standings.Take) error
}

type Boards interface {
	MarkTaken(take standings.Take)
}

func New(implementation UseCase, boards Boards) *Decorator {
	return &Decorator{implementation: implementation, boards: boards}
}

type Decorator struct {
	implementation UseCase
	boards         Boards
}

func (d *Decorator) Execute(ctx context.Context, take standings.Take) error {
	if err := d.implementation.Execute(ctx, take); err != nil {
		return err //nolint:wrapcheck // a decorator adds a mark, not a sentence.
	}

	d.boards.MarkTaken(take)
	return nil
}
