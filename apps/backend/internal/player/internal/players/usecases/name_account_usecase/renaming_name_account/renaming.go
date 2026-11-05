package renaming_name_account

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type UseCase interface {
	Execute(ctx context.Context, account players.AccountID, suggested string) (players.Profile, error)
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

func (r *Renaming) Execute(ctx context.Context, account players.AccountID, suggested string) (players.Profile, error) {
	profile, err := r.inner.Execute(ctx, account, suggested)
	if err != nil {
		return profile, err //nolint:wrapcheck // a decorator adds a rename, not a sentence.
	}
	named := profile.TakeSnapshot()
	r.visits.Rename(named.Account, named.Name)
	return profile, nil
}
