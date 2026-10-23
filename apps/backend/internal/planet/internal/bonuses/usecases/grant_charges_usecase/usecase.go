package grant_charges_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

type Charger interface {
	Grant(ctx context.Context, holder bonuses.Holder, kind bonuses.Kind, amount int) error
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

type In struct {
	Account string
	Grant   bonuses.Held
}

type Out struct {
	Holder bonuses.Holder
	Before bonuses.Held
	After  bonuses.Held
}

func New(charger Charger) *UseCase {
	return &UseCase{charger: charger}
}

type UseCase struct {
	charger Charger
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	holder, err := bonuses.ParseHolder(in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot grant: %w", err)
	}
	if err := in.Grant.GrantError(); err != nil {
		return Out{}, fmt.Errorf("cannot grant: %w", err)
	}

	before, err := u.charger.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}
	for _, kind := range bonuses.Kinds {
		if amount := in.Grant.Count(kind); amount > 0 {
			if err := u.charger.Grant(ctx, holder, kind, amount); err != nil {
				return Out{}, err
			}
		}
	}
	after, err := u.charger.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}

	return Out{Holder: holder, Before: before, After: after}, nil
}
