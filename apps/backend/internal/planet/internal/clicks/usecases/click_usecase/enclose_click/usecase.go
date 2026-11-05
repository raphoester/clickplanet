package enclose_click

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
)

type Enclosures interface {
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

func New(implementation click_usecase.IUseCase, enclosures Enclosures, terrain bonuses.Terrain, annexer Annexer,
	rules func() bonuses.Rules,
) *UseCase {
	return &UseCase{implementation: implementation, enclosures: enclosures, terrain: terrain, annexer: annexer, rules: rules}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	enclosures     Enclosures
	terrain        bonuses.Terrain
	annexer        Annexer
	rules          func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	if !in.Enclose {
		return u.implementation.Execute(ctx, in)
	}

	payer := clicks.PayerOf(ctx)
	holder := bonuses.HolderOf(payer)
	rules := u.rules().Enclose

	armed, err := u.armed(ctx, holder, rules)
	if err != nil {
		return click_usecase.Out{}, err
	}

	out, err := u.implementation.Execute(ctx, in)
	if err != nil || !armed || out.Outcome != clicks.Taken {
		return out, err
	}

	pocket := u.terrain.PocketClosedBy(in.TileID, in.CountryID, rules.MaxTiles)

	return out, u.annexer.Annex(ctx, holder, in, pocket, rules)
}

func (u *UseCase) armed(ctx context.Context, holder bonuses.Holder, rules bonuses.EncloseRules) (bool, error) {
	if rules.Unlimited {
		return true, nil
	}

	held, err := u.enclosures.Held(ctx, holder)
	if err != nil {
		return false, err
	}

	return held.Enclosures > 0, nil
}
