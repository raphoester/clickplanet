package allegiances

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

type Store interface {
	// Change runs under a lock on the account, and keeps what it answers when that differs from what it was handed.
	Change(ctx context.Context, account Account, change func(Allegiance) (Allegiance, error)) error
}

type CountryChecker interface {
	CheckCountry(country string) bool
}

func NewBook(store Store, countries CountryChecker, clock cptime.Clock) Book {
	return Book{store: store, countries: countries, clock: clock}
}

type Book struct {
	store     Store
	countries CountryChecker
	clock     cptime.Clock
}

func (b Book) Pledge(ctx context.Context, payer clicks.Payer, flag string) error {
	account := AccountOf(payer)
	if account == NoAccount || !b.countries.CheckCountry(flag) {
		return nil
	}

	at := b.clock.Now()
	if err := b.store.Change(ctx, account, func(allegiance Allegiance) (Allegiance, error) {
		return allegiance.Pledged(flag, at)
	}); err != nil {
		return fmt.Errorf("account %s may not play for %q: %w", account, flag, err)
	}

	return nil
}
