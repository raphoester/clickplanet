// Package get_budget_usecase reads a caller's click allowance without spending it. It
// is what a client that has just loaded asks once; every click afterwards
// answers with a fresh reading of its own.
package get_budget_usecase

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// ClickBudgetReader reports an allowance under the same keys the rate limiter
// spends it under. Peek creates no bucket: reading an allowance must not be a
// way to make the limiter remember a caller.
type ClickBudgetReader interface {
	Peek(key cpratelimit.Key) cpratelimit.State
}

type Pricer interface {
	Price(country string) clicks.Price
}

// New takes a nil reader for a server that does not rate limit clicks; Execute
// then reports no allowance and a client shows none.
func New(budgets ClickBudgetReader, pricer Pricer, buckets clicks.Buckets) *UseCase {
	return &UseCase{budgets: budgets, pricer: pricer, buckets: buckets}
}

type UseCase struct {
	budgets ClickBudgetReader
	pricer  Pricer
	buckets clicks.Buckets
}

// Execute derives the keys the same way the throttle charges them, and reports the tighter bucket. Deriving it
// anywhere else is how a caller is told about somebody else's bucket.
//
// It reports false when nothing is limiting clicks, which is not the same answer
// as an allowance of zero.
func (u *UseCase) Execute(ctx context.Context, country string) (clicks.Budget, bool) {
	if u.budgets == nil {
		return clicks.Budget{}, false
	}

	keys := u.buckets.Keys(clicks.PayerOf(ctx))
	states := make([]cpratelimit.State, len(keys))
	for i, key := range keys {
		states[i] = u.budgets.Peek(key)
	}

	return u.buckets.BudgetOf(clicks.Tightest(states), u.pricer.Price(country)), true
}
