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
	worn  titles.Standing
	shown []titles.Standing
}

func ShowcaseOf(catalog titles.Catalog, held titles.IDs, choice titles.ID) Showcase {
	shown := catalog.Shown(held)
	chosen, _ := catalog.StandingOf(choice)
	return Showcase{worn: WornOf(shown, chosen), shown: shown}
}

func (s Showcase) Worn() titles.Standing { return s.worn }

func (s Showcase) Shown() []titles.Standing { return s.shown }

type author = players.Author

type Author struct {
	author
	worn titles.Standing
}

func AuthorOf(shown players.Author, worn titles.Standing) Author {
	if shown.Guest() {
		return Author{author: shown}
	}
	return Author{author: shown, worn: worn}
}

func (a Author) Worn() titles.Standing { return a.worn }

func (a Author) Wearing(worn titles.Standing) Author { return AuthorOf(a.author, worn) }

func (a Author) Renamed(username players.Name) Author {
	return AuthorOf(a.author.Renamed(username), a.worn)
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
	return Showcase{worn: WornOf(shown, chosen), shown: shown}, nil
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
			if standing.Title().ID() == chosen.Title().ID() || (chosen.Place().Ranked() && standing.Place().Track() == chosen.Place().Track()) {
				return standing
			}
		}
	}
	for _, standing := range shown {
		if standing.Place().Ranked() {
			return standing
		}
	}
	if len(shown) > 0 {
		return shown[0]
	}
	return titles.Standing{}
}

func Wearable(shown []titles.Standing, title titles.ID) bool {
	return slices.ContainsFunc(shown, func(standing titles.Standing) bool { return standing.Title().ID() == title })
}
