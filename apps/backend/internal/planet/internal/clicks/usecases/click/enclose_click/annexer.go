package enclose_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/bonus"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
)

type TileStorage interface {
	Set(ctx context.Context, tile uint32, value string) error
}

// Publisher tells the planet a shape was closed, so every client can show it.
type Publisher interface {
	PublishEnclosed(scope string, enclosed bonus.Enclosed)
}

// Annexer takes the pockets a click closed, one shape of the bonus each.
type Annexer struct {
	storage   TileStorage
	publisher Publisher
}

func NewAnnexer(storage TileStorage, publisher Publisher) Annexer {
	return Annexer{storage: storage, publisher: publisher}
}

// Annex stops when the bonus runs out: a click closing two shapes with one left takes the first.
func (a Annexer) Annex(ctx context.Context, scope string, closing click.In, enclosure *bonus.Enclosure, pockets []Pocket) error {
	for _, pocket := range pockets {
		left, ok := enclosure.Spend()
		if !ok {
			return nil
		}

		if err := a.take(ctx, pocket, closing.CountryID); err != nil {
			return err
		}

		// After the tiles, so no client shows a shape filling that the map has not taken.
		a.publisher.PublishEnclosed(scope, pocket.announcement(closing, left))
	}

	return nil
}

func (a Annexer) take(ctx context.Context, pocket Pocket, country string) error {
	for _, tile := range pocket.inside {
		if err := a.storage.Set(ctx, tile, country); err != nil {
			return fmt.Errorf("failed to take enclosed tile %d: %w", tile, err)
		}
	}

	return nil
}
