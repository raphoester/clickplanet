package wearing

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/titles"
)

var ErrNotWearable = errors.New("the account does not show this title")

type Store interface {
	Choice(ctx context.Context, account players.AccountID) (titles.ID, error)
	Wear(ctx context.Context, account players.AccountID, title titles.ID, at time.Time) error
	DeleteAccount(ctx context.Context, account players.AccountID) error
}

type Titles interface {
	Shown(ctx context.Context, account players.AccountID) ([]titles.Standing, error)
}

type Showcase struct {
	Worn  titles.Standing
	Shown []titles.Standing
}

func ShowcaseOf(catalog titles.Catalog, held titles.IDs, choice titles.ID) Showcase {
	shown := catalog.Shown(held)
	chosen, _ := catalog.StandingOf(choice)
	return Showcase{Worn: WornOf(shown, chosen), Shown: shown}
}

type Author struct {
	players.Author
	Worn titles.Standing
}

func AuthorOf(author players.Author, worn titles.Standing) Author {
	if author.Guest {
		return Author{Author: author}
	}
	return Author{Author: author, Worn: worn}
}

type Wardrobe struct {
	store   Store
	titles  Titles
	catalog titles.Catalog
}

func NewWardrobe(store Store, titles Titles, catalog titles.Catalog) Wardrobe {
	return Wardrobe{store: store, titles: titles, catalog: catalog}
}

func (w Wardrobe) Showcase(ctx context.Context, account players.AccountID) (Showcase, error) {
	shown, err := w.titles.Shown(ctx, account)
	if err != nil {
		return Showcase{}, fmt.Errorf("failed to read the titles shown: %w", err)
	}
	choice, err := w.store.Choice(ctx, account)
	if err != nil {
		return Showcase{}, fmt.Errorf("failed to read the title chosen: %w", err)
	}
	chosen, _ := w.catalog.StandingOf(choice)
	return Showcase{Worn: WornOf(shown, chosen), Shown: shown}, nil
}

func (w Wardrobe) Wear(ctx context.Context, account players.AccountID, title titles.ID, at time.Time) error {
	shown, err := w.titles.Shown(ctx, account)
	if err != nil {
		return fmt.Errorf("failed to read the titles shown: %w", err)
	}
	if !Wearable(shown, title) {
		return fmt.Errorf("%w: %q", ErrNotWearable, title)
	}
	if err := w.store.Wear(ctx, account, title, at); err != nil {
		return fmt.Errorf("failed to wear the title: %w", err)
	}
	return nil
}

func WornOf(shown []titles.Standing, chosen titles.Standing) titles.Standing {
	if !chosen.Empty() {
		for _, standing := range shown {
			if standing.Title.ID() == chosen.Title.ID() || (chosen.Place.Ranked() && standing.Place.Track == chosen.Place.Track) {
				return standing
			}
		}
	}
	for _, standing := range shown {
		if standing.Place.Ranked() {
			return standing
		}
	}
	if len(shown) > 0 {
		return shown[0]
	}
	return titles.Standing{}
}

func Wearable(shown []titles.Standing, title titles.ID) bool {
	return slices.ContainsFunc(shown, func(standing titles.Standing) bool { return standing.Title.ID() == title })
}
