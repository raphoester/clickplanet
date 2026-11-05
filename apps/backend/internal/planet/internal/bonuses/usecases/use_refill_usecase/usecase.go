package use_refill_usecase

import (
	"context"
	"errors"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

var (
	ErrNoRefill = errors.New("no refill to use")

	ErrBankFull = errors.New("the click bank is already full")
)

type Refills interface {
	Spend(ctx context.Context, holder bonuses.Holder, after func(bonuses.Held) (bonuses.Held, bool)) error
	Held(ctx context.Context, holder bonuses.Holder) (bonuses.Held, error)
}

type Bank interface {
	Peek(key cpratelimit.Key) cpratelimit.State
	Fill(key cpratelimit.Key) (bool, cpratelimit.State)
	Add(key cpratelimit.Key, n float64)
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

	bank, ceiling := u.buckets.Bank(payer), u.buckets.Ceiling(payer)
	lack := u.lack(bank, ceiling)
	if lack <= 0 {
		return Out{}, ErrBankFull
	}

	if err := u.refills.Spend(ctx, holder, bonuses.Held.AfterRefill); err != nil {
		if errors.Is(err, bonuses.ErrNotHeld) {
			return Out{}, ErrNoRefill
		}
		return Out{}, err
	}
	for _, key := range bank {
		_, _ = u.bank.Fill(key)
	}
	for _, key := range ceiling {
		u.bank.Add(key, lack)
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

// lack is how far what the payer can spend is from a full bank: the size of its own, less the emptiest bucket it spends from.
func (u *UseCase) lack(bank, ceiling []cpratelimit.Key) float64 {
	size, spendable := math.Inf(1), math.Inf(1)
	for _, key := range bank {
		state := u.bank.Peek(key)
		size, spendable = min(size, float64(state.Capacity)), min(spendable, state.Tokens)
	}
	for _, key := range ceiling {
		spendable = min(spendable, u.bank.Peek(key).Tokens)
	}

	return size - spendable
}
