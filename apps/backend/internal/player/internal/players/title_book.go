package players

import (
	"context"
	"fmt"
	"time"
)

type TitleStore interface {
	Titles(ctx context.Context, account AccountID) (TitleIDs, error)
	GrantTitles(ctx context.Context, grants Grants, at time.Time) error
}

type TitleBook struct {
	store   TitleStore
	catalog Catalog
}

func NewTitleBook(store TitleStore, catalog Catalog) TitleBook {
	return TitleBook{store: store, catalog: catalog}
}

func (b TitleBook) TitlesOf(ctx context.Context, account AccountID) ([]Title, error) {
	held, err := b.store.Titles(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to read the titles: %w", err)
	}
	return b.catalog.Of(held), nil
}

func (b TitleBook) Award(ctx context.Context, account AccountID, stats Stats, at time.Time) error {
	held, err := b.store.Titles(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the titles: %w", err)
	}

	earned := b.catalog.EarnedBy(stats).Without(held)
	if len(earned) == 0 {
		return nil
	}
	if err := b.store.GrantTitles(ctx, Grants{account: earned}, at); err != nil {
		return fmt.Errorf("failed to grant the titles: %w", err)
	}
	return nil
}
