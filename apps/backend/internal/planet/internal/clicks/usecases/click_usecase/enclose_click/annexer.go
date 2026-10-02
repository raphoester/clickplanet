package enclose_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type TileStorage interface {
	Owner(tile uint32) (string, bool)
	Set(ctx context.Context, tile uint32, value string) error
}

// Rule is the home-soil rule: what a click for flag does to a tile owner holds.
type Rule interface {
	Outcome(tile uint32, owner, flag string) clicks.Outcome
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
	rule      Rule
	spender   Spender
	publisher Publisher
}

func NewAnnexer(storage TileStorage, rule Rule, spender Spender, publisher Publisher) Annexer {
	return Annexer{storage: storage, rule: rule, spender: spender, publisher: publisher}
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

// take claims each tile inside as a click on it would: a native tile of another country is cleared, not taken.
func (a Annexer) take(ctx context.Context, pocket bonuses.Pocket, country string) error {
	for _, tile := range pocket.Inside() {
		owner, _ := a.storage.Owner(tile)
		after := a.rule.Outcome(tile, owner, country).OwnerAfter(owner, country)

		if err := a.storage.Set(ctx, tile, after); err != nil {
			return fmt.Errorf("failed to take enclosed tile %d: %w", tile, err)
		}
	}

	return nil
}
