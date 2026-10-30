package use_refill_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
)

var ErrNoRefill = errors.New("no refill to use")

type Refills interface {
	Spend(ctx context.Context, holder bonuses.Holder, after func(bonuses.Held) (bonuses.Held, bool)) error
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

type Refilling interface {
	Full(payer clicks.Payer) bool
	Refill(payer clicks.Payer)
	Budget(payer clicks.Payer, price clicks.Price) clicks.Budget
}

type Pricer interface {
	Price(country string) clicks.Price
}

type In struct {
	CountryID string
}

type Out struct {
	Budget clicks.Budget
	Held   bonuses.Held
}

func New(refills Refills, refilling Refilling, pricer Pricer) *UseCase {
	return &UseCase{refills: refills, refilling: refilling, pricer: pricer}
}

type UseCase struct {
	refills   Refills
	refilling Refilling
	pricer    Pricer
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	payer := clicks.PayerOf(ctx)
	holder := bonuses.HolderOf(payer)
	if holder == bonuses.NoHolder {
		return Out{}, ErrNoRefill
	}

	if u.refilling.Full(payer) {
		return Out{}, clicks.ErrBankFull
	}

	if err := u.refills.Spend(ctx, holder, bonuses.Held.AfterRefill); err != nil {
		if errors.Is(err, bonuses.ErrNotHeld) {
			return Out{}, ErrNoRefill
		}
		return Out{}, err
	}
	u.refilling.Refill(payer)

	budget := u.refilling.Budget(payer, u.pricer.Price(in.CountryID))

	held, err := u.refills.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}

	return Out{Budget: budget, Held: held}, nil
}
