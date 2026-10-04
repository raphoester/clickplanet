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

type Rule interface {
	Outcome(tile uint32, owner, flag string) clicks.Outcome
}

type Spender interface {
	SpendEnclose(holder bonuses.Holder) bool
}

type Publisher interface {
	PublishEnclosed(entrant bonuses.Entrant, enclosed bonuses.Enclosed)
}

type Annexer struct {
	storage   TileStorage
	rule      Rule
	spender   Spender
	publisher Publisher
}

func NewAnnexer(storage TileStorage, rule Rule, spender Spender, publisher Publisher) Annexer {
	return Annexer{storage: storage, rule: rule, spender: spender, publisher: publisher}
}

func (a Annexer) Annex(ctx context.Context, entrant bonuses.Entrant, holder bonuses.Holder, closing click_usecase.In, pockets []bonuses.Pocket) error {
	if len(pockets) == 0 || !a.spender.SpendEnclose(holder) {
		return nil
	}

	pocket := pockets[0]
	if err := a.take(ctx, pocket, closing.CountryID); err != nil {
		return err
	}

	a.publisher.PublishEnclosed(entrant, pocket.Announcement(closing.CountryID, closing.TileID))

	return nil
}

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
