package live_grant_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/auth/internal/grants"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	Grant(ctx context.Context, id grants.GrantID) (grants.Grant, error)
}

func New(store Store, clock cptime.Clock) *UseCase {
	return &UseCase{store: store, clock: clock}
}

// The grant while it lives: grants.ErrGrantNotFound for one nobody kept, grants.ErrGrantExpired for one past its
// Lifetime that the prune has not deleted yet. Reading it changes nothing, so a join tried again reads it again.
type UseCase struct {
	store Store
	clock cptime.Clock
}

func (u *UseCase) Execute(ctx context.Context, id grants.GrantID) (grants.Grant, error) {
	grant, err := u.store.Grant(ctx, id)
	if err != nil {
		return grants.Grant{}, fmt.Errorf("failed to find the grant: %w", err)
	}
	if err := grant.ExpiryError(u.clock.Now()); err != nil {
		return grants.Grant{}, fmt.Errorf("failed to check the grant: %w", err)
	}
	return grant, nil
}
