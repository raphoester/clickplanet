package keep_appearance_usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/appearances"
	"github.com/raphoester/clickplanet.lol-backend/internal/clips/internal/casts"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Casts interface {
	Cast(ctx context.Context, id casts.CastID) (casts.Cast, error)
}

type Store interface {
	Keep(ctx context.Context, named []appearances.Appearance) error
	ScheduledBefore(ctx context.Context, at time.Time) ([]appearances.Key, error)
	Forget(ctx context.Context, key appearances.Key) error
}

func New(casts Casts, store Store, clock cptime.Clock) *UseCase {
	return &UseCase{casts: casts, store: store, clock: clock}
}

type UseCase struct {
	casts Casts
	store Store
	clock cptime.Clock
}

// The players kept as named. The appearances no player is told of any more are forgotten.
func (u *UseCase) Execute(ctx context.Context, cast casts.CastID, fighters []casts.Number, post appearances.Post) (int, error) {
	found, err := u.casts.Cast(ctx, cast)
	if err != nil {
		return 0, fmt.Errorf("failed to read the cast: %w", err)
	}
	accounts, err := found.AccountsOf(fighters)
	if err != nil {
		return 0, err //nolint:wrapcheck // the cast named the number it does not have.
	}

	named := post.AppearancesOf(accounts)
	if err := u.store.Keep(ctx, named); err != nil {
		return 0, fmt.Errorf("failed to keep who the post names: %w", err)
	}
	if err := u.forget(ctx); err != nil {
		return 0, err
	}
	return len(named), nil
}

func (u *UseCase) forget(ctx context.Context) error {
	stale, err := u.store.ScheduledBefore(ctx, appearances.ShownSince(u.clock.Now()))
	if err != nil {
		return fmt.Errorf("failed to find the appearances to forget: %w", err)
	}
	for _, key := range stale {
		if err := u.store.Forget(ctx, key); err != nil {
			return fmt.Errorf("failed to forget that %s named a player: %w", key.Post, err)
		}
	}
	return nil
}
