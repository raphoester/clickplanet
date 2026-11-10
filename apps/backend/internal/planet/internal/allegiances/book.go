package allegiances

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

const readTimeout = time.Second

var ErrUnread = errors.New("the flag an account plays for could not be read")

type Store interface {
	Allegiance(ctx context.Context, account Account) (Allegiance, error)
	// Change runs under a lock on the account, and keeps what it answers when that differs from what it was handed.
	Change(ctx context.Context, account Account, change func(Allegiance) (Allegiance, error)) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

// Tells the other modules a flag an account set or changed, once it is written: no previous for a first flag.
type Teller interface {
	Pledged(account Account, flag string, previous string)
}

func NewBook(
	store Store, countries CountryChecker, lock func() Lock, standing Standing, clock cptime.Clock, teller Teller,
) Book {
	return Book{store: store, countries: countries, lock: lock, standing: standing, clock: clock, teller: teller}
}

type Book struct {
	store     Store
	countries CountryChecker
	lock      func() Lock
	standing  Standing
	clock     cptime.Clock
	teller    Teller
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
	var previous string
	changed := false
	if err := b.store.Change(ctx, account, func(allegiance Allegiance) (Allegiance, error) {
		pledged, err := allegiance.Pledged(flag, at, lock, b.standing)
		previous, changed = allegiance.flag, err == nil && pledged.flag != allegiance.flag
		return pledged, err
	}); err != nil {
		return fmt.Errorf("account %s may not play for %q: %w", account, flag, err)
	}

	if changed {
		b.teller.Pledged(account, flag, previous)
	}
	return nil
}

func (b Book) Release(ctx context.Context, account Account) error {
	if err := b.store.Change(ctx, account, func(allegiance Allegiance) (Allegiance, error) {
		return allegiance.Released(), nil
	}); err != nil {
		return fmt.Errorf("could not release account %s from its wait: %w", account, err)
	}

	return nil
}

func (b Book) PlayError(ctx context.Context, payer clicks.Payer, flag string) error {
	account := AccountOf(payer)
	if account == NoAccount || !b.countries.CheckCountry(flag) {
		return nil
	}

	read, cancel := context.WithTimeout(ctx, readTimeout)
	defer cancel()
	allegiance, err := b.store.Allegiance(read, account)
	if err != nil {
		return fmt.Errorf("%w: account %s: %w", ErrUnread, account, err)
	}
	if err := allegiance.PlayError(flag, b.lock()); err != nil {
		return fmt.Errorf("account %s may not play for %q: %w", account, flag, err)
	}

	return nil
}
