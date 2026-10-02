package titles

import (
	"context"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

type Book struct {
	store   Store
	catalog Catalog
}

func NewBook(store Store, catalog Catalog) Book {
	return Book{store: store, catalog: catalog}
}

func (b Book) TitlesOf(ctx context.Context, account players.AccountID) ([]Title, error) {
	held, err := b.store.Held(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	return b.catalog.Of(held), nil
}

func (b Book) Award(ctx context.Context, account players.AccountID, career Career, at time.Time) error {
	earned := b.catalog.EarnedBy(career)
	if len(earned) == 0 {
		return nil
	}

	held, err := b.store.Held(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the titles: %w", err)
	}

	missing := earned.Without(held)
	if len(missing) == 0 {
		return nil
	}
	if err := b.store.Grant(ctx, Holdings{account: missing}, at); err != nil {
		return fmt.Errorf("failed to grant the titles: %w", err)
	}
	return nil
}
