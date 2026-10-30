package claim_ad_refill_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpatomic"
)

var (
	ErrAdRefillOff = errors.New("the rules in force give no refill for an ad")

	ErrNoAccount = errors.New("only an account has a refill for an ad")
)

type AdRefills interface {
	Record(ctx context.Context, payer clicks.Payer) error
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

func New(
	transactioner cpatomic.Transactioner, adRefills AdRefills, refilling Refilling, pricer Pricer, rules func() bonuses.Rules,
) *UseCase {
	return &UseCase{transactioner: transactioner, adRefills: adRefills, refilling: refilling, pricer: pricer, rules: rules}
}

type UseCase struct {
	transactioner cpatomic.Transactioner
	adRefills     AdRefills
	refilling     Refilling
	pricer        Pricer
	rules         func() bonuses.Rules
}

func (u *UseCase) Execute(ctx context.Context, in In) (clicks.Budget, error) {
	if !u.rules().AdRefill.Enabled {
		return clicks.Budget{}, ErrAdRefillOff
	}

	payer := clicks.PayerOf(ctx)
	if bonuses.HolderOf(payer) == bonuses.NoHolder {
		return clicks.Budget{}, ErrNoAccount
	}

	if u.refilling.Full(payer) {
		return clicks.Budget{}, clicks.ErrBankFull
	}

	if err := u.transactioner.InTx(ctx, func(ctx context.Context) error {
		return u.adRefills.Record(ctx, payer) //nolint:wrapcheck // the ad refills named it.
	}); err != nil {
		return clicks.Budget{}, err
	}
	u.refilling.Refill(payer)

	return u.refilling.Budget(payer, u.pricer.Price(in.CountryID)), nil
}
