package ctxutil

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/kernel/ratelimit"
)

type clickBudgetKey struct{}

// AddClickBudgetToContext hands the handler what the rate limiter had left
// after it let this call through. The interceptor decides the policy; the
// handler decides how to say it on the wire.
func AddClickBudgetToContext(ctx context.Context, state ratelimit.State) context.Context {
	return context.WithValue(ctx, clickBudgetKey{}, state)
}

// GetClickBudget reports the allowance left, and whether the call passed a rate
// limiter at all: a server configured without one leaves nothing behind.
func GetClickBudget(ctx context.Context) (ratelimit.State, bool) {
	state, ok := ctx.Value(clickBudgetKey{}).(ratelimit.State)
	return state, ok
}
