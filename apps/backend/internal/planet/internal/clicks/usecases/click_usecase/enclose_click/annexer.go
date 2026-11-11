package enclose_click

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/charges"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Encloser interface {
	Enclose(ctx context.Context, enclosure clicks.Enclosure, inside []uint32) error
}

type Spender interface {
	Spend(ctx context.Context, holder charges.Holder, after func(charges.Held) (charges.Held, bool)) error
}

type Annexer struct {
	encloser Encloser
	spender  Spender
}

func NewAnnexer(encloser Encloser, spender Spender) Annexer {
	return Annexer{encloser: encloser, spender: spender}
}

func (a Annexer) Annex(ctx context.Context, holder charges.Holder, closing click_usecase.In, pocket charges.Pocket,
	rules charges.EncloseRules,
) error {
	if pocket.Empty() {
		return nil
	}
	if !rules.Unlimited {
		if err := a.spender.Spend(ctx, holder, charges.Held.AfterEnclose); err != nil {
			if errors.Is(err, charges.ErrNotHeld) {
				return nil
			}
			return err
		}
	}

	if err := a.encloser.Enclose(ctx, pocket.Enclosure(closing.CountryID, closing.TileID), pocket.Inside()); err != nil {
		return fmt.Errorf("failed to take the pocket closed at tile %d: %w", closing.TileID, err)
	}

	return nil
}
