package enclose_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Encloser interface {
	Enclose(ctx context.Context, tile uint32, flag string, inside []uint32) error
}

type Spender interface {
	SpendEnclose(holder bonuses.Holder) bool
}

type Publisher interface {
	PublishEnclosed(entrant bonuses.Entrant, enclosed bonuses.Enclosed)
}

type Annexer struct {
	encloser  Encloser
	spender   Spender
	publisher Publisher
}

func NewAnnexer(encloser Encloser, spender Spender, publisher Publisher) Annexer {
	return Annexer{encloser: encloser, spender: spender, publisher: publisher}
}

func (a Annexer) Annex(ctx context.Context, entrant bonuses.Entrant, holder bonuses.Holder, closing click_usecase.In, pockets []bonuses.Pocket) error {
	if len(pockets) == 0 || !a.spender.SpendEnclose(holder) {
		return nil
	}

	pocket := pockets[0]
	if err := a.encloser.Enclose(ctx, closing.TileID, closing.CountryID, pocket.Inside()); err != nil {
		return fmt.Errorf("failed to take the pocket closed at tile %d: %w", closing.TileID, err)
	}

	a.publisher.PublishEnclosed(entrant, pocket.Announcement(closing.CountryID, closing.TileID))

	return nil
}
