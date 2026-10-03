package dressing_wear_title

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID, title titles.ID) (titles.Standing, error)
}

type Visits interface {
	Wear(account players.AccountID, worn titles.Standing)
}

func New(inner UseCase, visits Visits) *Dressing {
	return &Dressing{inner: inner, visits: visits}
}

type Dressing struct {
	inner  UseCase
	visits Visits
}

var _ UseCase = (*Dressing)(nil)

func (d *Dressing) Execute(ctx context.Context, account players.AccountID, title titles.ID) (titles.Standing, error) {
	worn, err := d.inner.Execute(ctx, account, title)
	if err != nil {
		return worn, err //nolint:wrapcheck // a decorator adds a roster change, not a sentence.
	}

	d.visits.Wear(account, worn)
	return worn, nil
}
