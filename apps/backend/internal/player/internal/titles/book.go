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

func (b Book) Shown(ctx context.Context, account players.AccountID) ([]Standing, error) {
	held, err := b.store.Held(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	return b.catalog.Shown(held), nil
}

func (b Book) ShownBy(ctx context.Context, accounts []players.AccountID) (map[players.AccountID][]Standing, error) {
	holdings, err := b.store.Holdings(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}

	shown := make(map[players.AccountID][]Standing, len(holdings))
	for account, held := range holdings {
		shown[account] = b.catalog.Shown(held)
	}
	return shown, nil
}

func (b Book) Progress(ctx context.Context, account players.AccountID, career Career) ([]TrackProgress, error) {
	held, err := b.store.Held(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	return b.catalog.Progress(career, held), nil
}

func (b Book) Unheld(ctx context.Context, account players.AccountID, career Career) (IDs, error) {
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

func (b Book) Grant(ctx context.Context, account players.AccountID, titles IDs, at time.Time) error {
	if err := b.store.Grant(ctx, Holdings{account: titles}, at); err != nil {
		return fmt.Errorf("failed to grant the titles: %w", err)
	}
	return nil
}
