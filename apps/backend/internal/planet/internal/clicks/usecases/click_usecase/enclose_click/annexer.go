package enclose_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type TileStorage interface {
	Set(ctx context.Context, tile uint32, value string) error
}

// Spender spends a caller's enclose charge, and says whether there was one to spend.
type Spender interface {
	SpendEnclose(holder bonuses.Holder) bool
}

// Publisher tells the planet a shape was closed, so every client can show it.
type Publisher interface {
	PublishEnclosed(scope string, enclosed bonuses.Enclosed)
}

// Annexer takes the first pocket a click closed, for the one shape the charge is worth.
type Annexer struct {
	storage   TileStorage
	spender   Spender
	publisher Publisher
}

func NewAnnexer(storage TileStorage, spender Spender, publisher Publisher) Annexer {
	return Annexer{storage: storage, spender: spender, publisher: publisher}
}

// Annex spends the charge only on a click that closed a pocket, so a click that closes nothing keeps it.
// A click closing two shapes takes the first: the charge is one shape. Two clicks racing for the charge
// get one shape between them.
func (a Annexer) Annex(ctx context.Context, scope string, holder bonuses.Holder, closing click_usecase.In, pockets []bonuses.Pocket) error {
	if len(pockets) == 0 || !a.spender.SpendEnclose(holder) {
		return nil
	}

	pocket := pockets[0]
	if err := a.take(ctx, pocket, closing.CountryID); err != nil {
		return err
	}

	// After the tiles, so no client shows a shape filling that the map has not taken.
	a.publisher.PublishEnclosed(scope, pocket.Announcement(closing.CountryID, closing.TileID))

	return nil
}

func (a Annexer) take(ctx context.Context, pocket bonuses.Pocket, country string) error {
	for _, tile := range pocket.Inside() {
		if err := a.storage.Set(ctx, tile, country); err != nil {
			return fmt.Errorf("failed to take enclosed tile %d: %w", tile, err)
		}
	}

	return nil
}
