package titles

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/progression/internal/accounts"
)

type Book struct {
	store   Store
	catalog Catalog
}

func NewBook(store Store, catalog Catalog) Book {
	return Book{store: store, catalog: catalog}
}

func (b Book) Shown(ctx context.Context, account accounts.AccountID) ([]Standing, error) {
	held, err := b.store.Held(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	return b.catalog.Shown(held), nil
}

func (b Book) Unheld(ctx context.Context, account accounts.AccountID, career Career) (IDs, error) {
	earned := b.catalog.EarnedBy(career)
	if len(earned) == 0 {
		return nil, nil
	}

	held, err := b.store.Held(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	return earned.Without(held), nil
}

func (b Book) Grant(ctx context.Context, account accounts.AccountID, titles IDs, at time.Time) error {
	if err := b.store.Grant(ctx, Holdings{account: titles}, at); err != nil {
		return fmt.Errorf("failed to grant the titles: %w", err)
	}
	return nil
}
