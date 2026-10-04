package marking_forget_account

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

type UseCase interface {
	Execute(ctx context.Context, account standings.AccountID) error
}

type Boards interface {
	MarkForgotten(account standings.AccountID)
}

func New(implementation UseCase, boards Boards) *Decorator {
	return &Decorator{implementation: implementation, boards: boards}
}

type Decorator struct {
	implementation UseCase
	boards         Boards
}

func (d *Decorator) Execute(ctx context.Context, account standings.AccountID) error {
	if err := d.implementation.Execute(ctx, account); err != nil {
		return err //nolint:wrapcheck // a decorator adds a mark, not a sentence.
	}

	d.boards.MarkForgotten(account)
	return nil
}
