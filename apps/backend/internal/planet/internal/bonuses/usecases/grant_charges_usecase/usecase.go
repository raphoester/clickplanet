package grant_charges_usecase

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
)

type Charger interface {
	Grant(holder bonuses.Holder, kind bonuses.Kind, amount int)
	Held(holder bonuses.Holder) bonuses.Held
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

func (u *UseCase) Execute(_ context.Context, in In) (Out, error) {
	holder, err := bonuses.ParseHolder(in.Account)
	if err != nil {
		return Out{}, fmt.Errorf("cannot grant: %w", err)
	}
	if err := in.Grant.GrantError(); err != nil {
		return Out{}, fmt.Errorf("cannot grant: %w", err)
	}

	before := u.charger.Held(holder)
	for _, kind := range bonuses.Kinds {
		if amount := in.Grant.Count(kind); amount > 0 {
			u.charger.Grant(holder, kind, amount)
		}
	}

	return Out{Holder: holder, Before: before, After: u.charger.Held(holder)}, nil
}
