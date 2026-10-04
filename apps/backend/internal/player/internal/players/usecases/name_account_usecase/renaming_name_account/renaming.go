package renaming_name_account

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID) (players.Profile, error)
}

type Visits interface {
	Rename(account players.AccountID, username players.Name)
}

func New(inner UseCase, visits Visits) *Renaming {
	return &Renaming{inner: inner, visits: visits}
}

type Renaming struct {
	inner  UseCase
	visits Visits
}

var _ UseCase = (*Renaming)(nil)

func (r *Renaming) Execute(ctx context.Context, account players.AccountID) (players.Profile, error) {
	profile, err := r.inner.Execute(ctx, account)
	if err != nil {
		return profile, err //nolint:wrapcheck // a decorator adds a rename, not a sentence.
	}
	r.visits.Rename(profile.Account(), profile.Name())
	return profile, nil
}
