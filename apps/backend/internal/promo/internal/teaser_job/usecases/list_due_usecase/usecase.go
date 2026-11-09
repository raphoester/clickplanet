package list_due_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/promo/internal/teaser_job"
)

// Promo's ListDueTeasers.
type Promo interface {
	Due(ctx context.Context) ([]teaser_job.Due, error)
}

// Where the renderer reads each teaser.
type Store interface {
	// The folder's name.
	Keep(ctx context.Context, due teaser_job.Due) (string, error)
}

// A teaser to render, in its folder.
type Kept struct {
	Folder string
	Due    teaser_job.Due
}

type Executor interface {
	Execute(ctx context.Context) ([]Kept, error)
}

func New(promo Promo, store Store) *UseCase {
	return &UseCase{promo: promo, store: store}
}

type UseCase struct {
	promo Promo
	store Store
}

var _ Executor = (*UseCase)(nil)

// Each battle promo says is due, kept for the renderer.
func (u *UseCase) Execute(ctx context.Context) ([]Kept, error) {
	due, err := u.promo.Due(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list the battles due their teaser: %w", err)
	}
	kept := make([]Kept, 0, len(due))
	for _, one := range due {
		folder, err := u.store.Keep(ctx, one)
		if err != nil {
			return nil, err //nolint:wrapcheck // the store already named the teaser it could not keep.
		}
		kept = append(kept, Kept{Folder: folder, Due: one})
	}
	return kept, nil
}
