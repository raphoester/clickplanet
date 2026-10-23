package claim_bonus_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

// One error for every failure: telling them apart helps a script guessing tokens.
var ErrNoSuchBonus = errors.New("no bonus to claim")

type Registry interface {
	Claim(token string, entrant bonuses.Entrant, scope string) (bonuses.Reward, bool)
	Publish(taken bonuses.Taken)
}

type Charger interface {
	Grant(ctx context.Context, holder bonuses.Holder, kind bonuses.Kind, amount int) error
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

type In struct {
	Token     string
	CountryID string
}

type Out struct {
	Kind bonuses.Kind

	Amount int

	Held bonuses.Held
}

func New(registry Registry, charger Charger) *UseCase {
	return &UseCase{registry: registry, charger: charger}
}

type UseCase struct {
	registry Registry
	charger  Charger
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	payer := clicks.PayerOf(ctx)

	reward, ok := u.registry.Claim(in.Token, bonuses.EntrantOf(payer), payer.Scope)
	if !ok {
		return Out{}, ErrNoSuchBonus
	}

	holder := bonuses.HolderOf(payer)
	before, err := u.charger.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}
	if err := u.charger.Grant(ctx, holder, reward.Kind, reward.Amount); err != nil {
		return Out{}, err
	}
	held, err := u.charger.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}

	u.registry.Publish(bonuses.Taken{CountryID: in.CountryID, Kind: reward.Kind})

	return Out{Kind: reward.Kind, Amount: held.Count(reward.Kind) - before.Count(reward.Kind), Held: held}, nil
}
