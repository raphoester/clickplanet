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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/toll"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// Limiter spends tokens and reports what the bucket holds afterwards. The
// reading comes back either way: a refused caller is the one that most needs to
// know when the next token lands.
type Limiter interface {
	TakeN(key string, n float64) (bool, cpratelimit.State)
}

// Pricer says how many tokens a click for a country costs.
type Pricer interface {
	Price(country string) toll.Price
}

func New(implementation click.IUseCase, limiter Limiter, pricer Pricer) *UseCase {
	return &UseCase{implementation: implementation, limiter: limiter, pricer: pricer}
}

type UseCase struct {
	implementation click.IUseCase
	limiter        Limiter
	pricer         Pricer
}

// Execute answers the reading on both paths. The allowed one carries it because
// the client redraws the meter off the server's own numbers; the refused one
// carries it because that is the moment a client most needs to know how long to
// wait, and it has no success message to read it from.
func (u *UseCase) Execute(ctx context.Context, in click.In) (click.Out, error) {
	price := u.pricer.Price(in.CountryID)

	allowed, state := u.limiter.TakeN(cpctx.RateLimitKey(ctx), price.Cost)
	if !allowed {
		return click.Out{Budget: toll.Of(state, price), Limited: true}, clicks.ErrThrottled
	}

	out, err := u.implementation.Execute(ctx, in)
	out.Budget, out.Limited = toll.Of(state, price), true

	return out, err
}
