package titles

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

var ErrNotWearable = errors.New("the account does not hold this title, or holds a higher rank of its track")

type Showcase struct {
	Worn  Standing
	Shown []Standing
}

type Dashboard struct {
	Worn     Standing
	Wearable []Standing
	Tracks   []TrackProgress
}

type Book struct {
	store   Store
	catalog Catalog
}

func NewBook(store Store, catalog Catalog) Book {
	return Book{store: store, catalog: catalog}
}

func (b Book) Showcase(ctx context.Context, account players.AccountID) (Showcase, error) {
	held, worn, err := b.holding(ctx, account)
	if err != nil {
		return Showcase{}, err
	}
	return Showcase{Worn: b.catalog.Worn(held, worn), Shown: b.catalog.Shown(held)}, nil
}

func (b Book) Dashboard(ctx context.Context, account players.AccountID, career Career) (Dashboard, error) {
	held, worn, err := b.holding(ctx, account)
	if err != nil {
		return Dashboard{}, err
	}
	return Dashboard{
		Worn:     b.catalog.Worn(held, worn),
		Wearable: b.catalog.Shown(held),
		Tracks:   b.catalog.Progress(career, held),
	}, nil
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

func (b Book) Wear(ctx context.Context, account players.AccountID, title ID, at time.Time) error {
	held, err := b.store.Held(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the titles: %w", err)
	}
	if !b.catalog.Wearable(held, title) {
		return fmt.Errorf("%w: %q", ErrNotWearable, title)
	}
	if err := b.store.Wear(ctx, account, title, at); err != nil {
		return fmt.Errorf("failed to wear the title: %w", err)
	}
	return nil
}

func (b Book) holding(ctx context.Context, account players.AccountID) (IDs, ID, error) {
	held, err := b.store.Held(ctx, account)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the titles: %w", err)
	}
	worn, err := b.store.Worn(ctx, account)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read the worn title: %w", err)
	}
	return held, worn, nil
}
