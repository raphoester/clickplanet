package use_refill_usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

var (
	ErrNoRefill = errors.New("no refill to use")

	ErrBankFull = errors.New("the click bank is already full")
)

type Refills interface {
	SpendRefill(holder bonuses.Holder) bool
	Held(holder bonuses.Holder) bonuses.Held
}

type Bank interface {
	Peek(key cpratelimit.Key) cpratelimit.State
	Fill(key cpratelimit.Key) (bool, cpratelimit.State)
}

type Pricer interface {
	PriceFor(ctx context.Context, payer clicks.Payer, country string) (clicks.Price, error)
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

	if !u.refills.SpendRefill(holder) {
		return Out{}, ErrNoRefill
	}
	for _, key := range bank {
		_, _ = u.bank.Fill(key)
	}

	price, err := u.pricer.PriceFor(ctx, payer, in.CountryID)
	if err != nil {
		return Out{}, fmt.Errorf("failed to price the refilled budget: %w", err)
	}
	keys := u.buckets.Keys(payer, price)
	states := make([]cpratelimit.State, len(keys))
	for i, key := range keys {
		states[i] = u.bank.Peek(key)
	}

	return Out{Budget: u.buckets.BudgetOf(payer, states, price), Held: u.refills.Held(holder)}, nil
}

func (u *UseCase) full(bank []cpratelimit.Key) bool {
	for _, key := range bank {
		if state := u.bank.Peek(key); state.Tokens < float64(state.Capacity) {
			return false
		}
	}
	return true
}
