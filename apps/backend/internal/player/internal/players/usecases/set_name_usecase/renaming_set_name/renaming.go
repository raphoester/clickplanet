// Package renaming_set_name shows a new username on the roster as soon as it is kept, rather than at the
// player's next announce, which waits for a click token the page may not hold.
package renaming_set_name

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players/usecases/set_name_usecase"
)

type UseCase interface {
	Execute(ctx context.Context, in set_name_usecase.In) (players.Profile, error)
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

// Execute renames the visit only when the name was kept.
func (r *Renaming) Execute(ctx context.Context, in set_name_usecase.In) (players.Profile, error) {
	profile, err := r.inner.Execute(ctx, in)
	if err != nil {
		return profile, err //nolint:wrapcheck // a decorator adds a rename, not a sentence.
	}

	r.visits.Rename(profile.Account, profile.Name)
	return profile, nil
}
