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

// Pricer says how many tokens a click for a country costs.
type Pricer interface {
	Price(country string) clicks.Price
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

// Execute charges the account and its scope together, at the country's price, and answers the tighter
// reading on both paths. The allowed one carries it because
// the client redraws the meter off the server's own numbers; the refused one
// carries it because that is the moment a client most needs to know how long to
// wait, and it has no success message to read it from.
func (u *UseCase) Execute(ctx context.Context, in click_usecase.In) (click_usecase.Out, error) {
	price := u.pricer.Price(in.CountryID)

	allowed, states := u.limiter.TakeAll(price.Cost, u.buckets.Keys(clicks.PayerOf(ctx))...)
	state := clicks.Tightest(states)

	if !allowed {
		return click_usecase.Out{Budget: clicks.BudgetOf(state, price), Limited: true}, clicks.ErrThrottled
	}

	in.Boosted = state.Boosted

	out, err := u.implementation.Execute(ctx, in)
	out.Budget, out.Limited = clicks.BudgetOf(state, price), true

	return out, err
}
