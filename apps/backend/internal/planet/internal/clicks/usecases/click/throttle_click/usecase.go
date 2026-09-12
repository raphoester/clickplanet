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
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks/usecases/click"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// Limiter spends a token and reports what the bucket holds afterwards. The
// reading comes back either way: a refused caller is the one that most needs to
// know when the next token lands.
type Limiter interface {
	Take(key string) (bool, cpratelimit.State)
}

func New(implementation click.IUseCase, limiter Limiter) *UseCase {
	return &UseCase{implementation: implementation, limiter: limiter}
}

type UseCase struct {
	implementation click.IUseCase
	limiter        Limiter
}

// Execute answers the reading on both paths. The allowed one carries it because
// the client redraws the meter off the server's own numbers; the refused one
// carries it because that is the moment a client most needs to know how long to
// wait, and it has no success message to read it from.
func (u *UseCase) Execute(ctx context.Context, in click.In) (click.Out, error) {
	allowed, state := u.limiter.Take(cpctx.RateLimitKey(ctx))
	if !allowed {
		return click.Out{Budget: state, Limited: true}, clicks.ErrThrottled
	}

	out, err := u.implementation.Execute(ctx, in)
	out.Budget, out.Limited = state, true

	return out, err
}
