package make_cast_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpcolls"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Ledger interface {
	Takes(ctx context.Context, window casts.Window) ([]casts.Take, error)
}

type Names interface {
	// The name each account may be shown under in the clips; an account with none is left out.
	Shown(ctx context.Context, accounts []casts.AccountID) (map[casts.AccountID]string, error)
}

type Store interface {
	Keep(ctx context.Context, cast casts.Cast) error
	MadeBefore(ctx context.Context, at time.Time) ([]casts.CastID, error)
	Forget(ctx context.Context, cast casts.CastID) error
}

func New(ledger Ledger, names Names, store Store, ids casts.IDProvider, clock cptime.Clock) *UseCase {
	return &UseCase{ledger: ledger, names: names, store: store, ids: ids, clock: clock}
}

type UseCase struct {
	ledger Ledger
	names  Names
	store  Store
	ids    casts.IDProvider
	clock  cptime.Clock
}

// The cast kept, and the takes of its fighters under their numbers. The casts too old to be posted are forgotten.
func (u *UseCase) Execute(ctx context.Context, since, until time.Time) (casts.Cast, []casts.NumberedTake, error) {
	now := u.clock.Now()
	window, err := casts.WindowOf(since, until, now)
	if err != nil {
		return casts.Cast{}, nil, err //nolint:wrapcheck // the domain named what is wrong.
	}
	if err := u.forget(ctx, now); err != nil {
		return casts.Cast{}, nil, err
	}

	takes, err := u.ledger.Takes(ctx, window)
	if err != nil {
		return casts.Cast{}, nil, fmt.Errorf("failed to read who took what: %w", err)
	}
	names, err := u.names.Shown(ctx, accountsOf(takes))
	if err != nil {
		return casts.Cast{}, nil, fmt.Errorf("failed to read the players' names: %w", err)
	}
	id, err := u.ids.NewID()
	if err != nil {
		return casts.Cast{}, nil, err //nolint:wrapcheck // the provider named what failed.
	}

	cast := casts.CastOf(id, now, window, takes, names)
	if err := u.store.Keep(ctx, cast); err != nil {
		return casts.Cast{}, nil, fmt.Errorf("failed to keep the cast: %w", err)
	}
	return cast, cast.Numbered(takes), nil
}

func (u *UseCase) forget(ctx context.Context, now time.Time) error {
	stale, err := u.store.MadeBefore(ctx, casts.ForgottenBefore(now))
	if err != nil {
		return fmt.Errorf("failed to find the casts to forget: %w", err)
	}
	for _, cast := range stale {
		if err := u.store.Forget(ctx, cast); err != nil {
			return fmt.Errorf("failed to forget the cast %s: %w", cast, err)
		}
	}
	return nil
}

func accountsOf(takes []casts.Take) []casts.AccountID {
	seen := cpcolls.NewSet[casts.AccountID]()
	var accounts []casts.AccountID
	for _, take := range takes {
		if !seen.Contains(take.Account) {
			seen.Add(take.Account)
			accounts = append(accounts, take.Account)
		}
	}
	return accounts
}
