package use_refill_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

var (
	ErrNoRefill = errors.New("no refill to use")

	ErrBankFull = errors.New("the click bank is already full")
)

type Refills interface {
	SpendRefill(ctx context.Context, holder bonuses.Holder) error
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

type Bank interface {
	Peek(key cpratelimit.Key) cpratelimit.State
	Fill(key cpratelimit.Key) (bool, cpratelimit.State)
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

func New(refills Refills, bank Bank, pricer Pricer, buckets clicks.Buckets) *UseCase {
	return &UseCase{refills: refills, bank: bank, pricer: pricer, buckets: buckets}
}

type UseCase struct {
	refills Refills
	bank    Bank
	pricer  Pricer
	buckets clicks.Buckets
}

func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	payer := clicks.PayerOf(ctx)
	holder := bonuses.HolderOf(payer)
	if holder == bonuses.NoHolder {
		return Out{}, ErrNoRefill
	}

	bank := u.buckets.Bank(payer)
	if u.full(bank) {
		return Out{}, ErrBankFull
	}

	if err := u.refills.SpendRefill(ctx, holder); err != nil {
		if errors.Is(err, bonuses.ErrNotHeld) {
			return Out{}, ErrNoRefill
		}
		return Out{}, err
	}
	for _, key := range bank {
		_, _ = u.bank.Fill(key)
	}

	price := u.pricer.Price(in.CountryID)
	keys := u.buckets.Keys(payer, price)
	states := make([]cpratelimit.State, len(keys))
	for i, key := range keys {
		states[i] = u.bank.Peek(key)
	}

	held, err := u.refills.Held(ctx, holder)
	if err != nil {
		return Out{}, err
	}

	return Out{Budget: u.buckets.BudgetOf(payer, states, price), Held: held}, nil
}

func (u *UseCase) full(bank []cpratelimit.Key) bool {
	for _, key := range bank {
		if state := u.bank.Peek(key); state.Tokens < float64(state.Capacity) {
			return false
		}
	}
	return true
}
