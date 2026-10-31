package allegiances

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	Allegiance(ctx context.Context, account Account) (Allegiance, error)
	// Change runs under a lock on the account, and keeps what it answers when that differs from what it was handed.
	Change(ctx context.Context, account Account, change func(Allegiance) (Allegiance, error)) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

func NewBook(store Store, countries CountryChecker, lock func() Lock, clock cptime.Clock) Book {
	return Book{store: store, countries: countries, lock: lock, clock: clock}
}

type Book struct {
	store     Store
	countries CountryChecker
	lock      func() Lock
	clock     cptime.Clock
}

func (b Book) Pledge(ctx context.Context, payer clicks.Payer, flag string) error {
	account := AccountOf(payer)
	if account == NoAccount {
		return ErrNoAccount
	}
	if !b.countries.CheckCountry(flag) {
		return fmt.Errorf("%q: %w", flag, clicks.ErrUnknownCountry)
	}

	at, lock := b.clock.Now(), b.lock()
	if err := b.store.Change(ctx, account, func(allegiance Allegiance) (Allegiance, error) {
		return allegiance.Pledged(flag, at, lock)
	}); err != nil {
		return fmt.Errorf("account %s may not play for %q: %w", account, flag, err)
	}

	return nil
}

func (b Book) PlayError(ctx context.Context, payer clicks.Payer, flag string) error {
	account := AccountOf(payer)
	if account == NoAccount || !b.countries.CheckCountry(flag) {
		return nil
	}

	allegiance, err := b.store.Allegiance(ctx, account)
	if err != nil {
		return fmt.Errorf("could not read the flag account %s plays for: %w", account, err)
	}
	if err := allegiance.PlayError(flag, b.lock()); err != nil {
		return fmt.Errorf("account %s may not play for %q: %w", account, flag, err)
	}

	return nil
}
