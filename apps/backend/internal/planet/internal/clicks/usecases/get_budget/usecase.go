// Package get_budget reads a caller's click allowance without spending it. It
// is what a client that has just loaded asks once; every click afterwards
// answers with a fresh reading of its own.
package get_budget

import (
	"context"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// ClickBudgetReader reports an allowance under the same key the rate limiter
// spends it under. Peek creates no bucket: reading an allowance must not be a
// way to make the limiter remember a caller.
type ClickBudgetReader interface {
	Peek(key string) cpratelimit.State
}

// New takes a nil reader for a server that does not rate limit clicks; Execute
// then reports no allowance and a client shows none.
func New(budgets ClickBudgetReader) *UseCase {
	return &UseCase{budgets: budgets}
}

type UseCase struct {
	budgets ClickBudgetReader
}

// Execute derives the key the same way the throttle charges it. Deriving it
// anywhere else is how a caller is told about somebody else's bucket.
//
// It reports false when nothing is limiting clicks, which is not the same answer
// as an allowance of zero.
func (u *UseCase) Execute(ctx context.Context) (cpratelimit.State, bool) {
	if u.budgets == nil {
		return cpratelimit.State{}, false
	}

	return u.budgets.Peek(cpctx.RateLimitKey(ctx)), true
}
