// Package use_refill_usecase spends a caller's refill charge: its click bank is filled to capacity.
package use_refill_usecase

import (
	"context"
	"errors"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

var (
	// ErrNoRefill covers never won, already used and held past its expiry, as ErrNoSuchBonus does.
	ErrNoRefill = errors.New("no refill to use")

	// ErrBankFull refuses a refill that would fill nothing, and spends nothing.
	ErrBankFull = errors.New("the click bank is already full")
)

// Refills spends the refill charge a caller holds, and says what is left.
type Refills interface {
	SpendRefill(holder bonuses.Holder) bool
	Held(holder bonuses.Holder) bonuses.Held
}

// Bank is the limiter the throttle spends: a refill that did not fill that bucket would not be a refill.
type Bank interface {
	Peek(key cpratelimit.Key) cpratelimit.State
	Fill(key cpratelimit.Key) (bool, cpratelimit.State)
}

type Pricer interface {
	Price(country string) clicks.Price
}

type In struct {
	// Prices the allowance answered.
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

// Execute fills the caller's own bucket, never the scope's, which the scope's other players share. A full
// bank is refused before the charge is touched, so a press on a full meter wastes nothing. A click landing
// between the check and the fill can make the fill a little short; it is never a fill of nothing.
func (u *UseCase) Execute(ctx context.Context, in In) (Out, error) {
	payer := clicks.PayerOf(ctx)
	holder := bonuses.HolderOf(payer)
	if holder == bonuses.NoHolder {
		return Out{}, ErrNoRefill
	}

	own := u.buckets.Own(payer)
	if state := u.bank.Peek(own); state.Tokens >= float64(state.Capacity) {
		return Out{}, ErrBankFull
	}

	if !u.refills.SpendRefill(holder) {
		return Out{}, ErrNoRefill
	}
	_, _ = u.bank.Fill(own)

	price := u.pricer.Price(in.CountryID)
	keys := u.buckets.Keys(payer, price)
	states := make([]cpratelimit.State, len(keys))
	for i, key := range keys {
		states[i] = u.bank.Peek(key)
	}

	return Out{Budget: u.buckets.BudgetOf(clicks.Tightest(states), price), Held: u.refills.Held(holder)}, nil
}
