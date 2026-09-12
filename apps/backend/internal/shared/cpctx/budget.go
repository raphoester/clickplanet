package cpctx

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

type rateBudgetKey struct{}

// AddRateBudgetToContext hands the handler what the rate limiter had left after
// it let this call through. The interceptor decides the policy; the handler
// decides how to say it on the wire, and whether to say it at all.
func AddRateBudgetToContext(ctx context.Context, state cpratelimit.State) context.Context {
	return context.WithValue(ctx, rateBudgetKey{}, state)
}

// GetRateBudget reports the allowance left, and whether the call passed a rate
// limiter at all: a server configured without one leaves nothing behind.
func GetRateBudget(ctx context.Context) (cpratelimit.State, bool) {
	state, ok := ctx.Value(rateBudgetKey{}).(cpratelimit.State)
	return state, ok
}
