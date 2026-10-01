// Package throttle_click is the per-caller click allowance, as a decorator over
// the click use case rather than an interceptor over the procedure.
//
// It sits outside antibot_click and inside every edge refusal, which is the
// order the chain has always had: a click refused for its address or its session
// must not also spend a token, or the retry that follows would come back
// throttled and the web app would show the wrong dialog; and a shadow-banned
// caller has to keep hitting the same refusals everyone else does, or it has
// been told.
package throttle_click

import (
	"context"
	"fmt"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click_usecase"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// Limiter spends from every bucket or from none, and reports what each holds afterwards. The
// reading comes back either way: a refused caller is the one that most needs to
// know when the next token lands.
type Limiter interface {
	TakeAll(n float64, keys ...cpratelimit.Key) (bool, []cpratelimit.State)
}

// Pricer says how much slower a payer gets its clicks back, from the flag it clicks for most.
type Pricer interface {
	PriceFor(ctx context.Context, payer clicks.Payer, country string) (clicks.Price, error)
}

func New(implementation click_usecase.IUseCase, limiter Limiter, pricer Pricer, buckets clicks.Buckets) *UseCase {
	return &UseCase{implementation: implementation, limiter: limiter, pricer: pricer, buckets: buckets}
}

type UseCase struct {
	implementation click_usecase.IUseCase
	limiter        Limiter
	pricer         Pricer
	buckets        clicks.Buckets
}

// Execute charges every bucket of the payer one token, sets the pace the account refills at from its
// main flag's price, and answers the tighter reading on both paths. The allowed one carries it because
// the client redraws the meter off the server's own numbers; the refused one
// carries it because that is the moment a client most needs to know how long to
// wait, and it has no success message to read it from.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	payer := clicks.PayerOf(ctx)
	price, err := u.pricer.PriceFor(ctx, payer, in.CountryID)
	if err != nil {
		return click_usecase.Out{}, fmt.Errorf("failed to price the click: %w", err)
	}

	allowed, states := u.limiter.TakeAll(1, u.buckets.Keys(payer, price)...)
	budget := u.buckets.BudgetOf(payer, states, price)

	if !allowed {
		return click_usecase.Out{Budget: budget, Limited: true}, clicks.ErrThrottled
	}

	out, err := u.implementation.Execute(ctx, in)
	out.Budget, out.Limited = budget, true

	return out, err
}
