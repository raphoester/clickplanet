package clicks

import (
	"context"
	"fmt"
	"math"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// ThrottleConfig is the `rateLimiter:` block: one account's allowance, and how many of them its scope may spend.
type ThrottleConfig struct {
	cpratelimit.Config `koanf:",squash"`

	// ScopeMultiplier is the scope's burst and rate over one account's: many players can share one address.
	ScopeMultiplier float64
}

const defaultScopeMultiplier = 10

// Validate refuses a scope that holds less than one account: every account behind it would be throttled by the scope.
func (c ThrottleConfig) Validate() error {
	if math.IsNaN(c.ScopeMultiplier) || c.ScopeMultiplier < 0 || (c.ScopeMultiplier > 0 && c.ScopeMultiplier < 1) {
		return fmt.Errorf("rateLimiter.scopeMultiplier is %v: it must be 1 or more, or unset for %d",
			c.ScopeMultiplier, defaultScopeMultiplier)
	}

	return nil
}

// Buckets is the throttle policy: which buckets a payer spends from.
func (c ThrottleConfig) Buckets() Buckets {
	multiplier := c.ScopeMultiplier
	if multiplier <= 0 {
		multiplier = defaultScopeMultiplier
	}

	return Buckets{scopeMultiplier: multiplier}
}

type Buckets struct {
	scopeMultiplier float64
}

// Payer is who a click is charged to: the scope it comes from, and the account its token names, if any.
type Payer struct {
	Scope   string
	Account string
}

func PayerOf(ctx context.Context) Payer {
	return Payer{Scope: cpipscope.Of(cpctx.GetSourceIP(ctx)), Account: cpctx.GetAccount(ctx)}
}

// Keys is the account's bucket first, then the scope's at the multiplier. With no account it is the
// scope's bucket alone at one, as it was before accounts: a separate bucket, so a scale never changes under a key.
func (b Buckets) Keys(payer Payer) []cpratelimit.Key {
	if payer.Account == "" {
		return []cpratelimit.Key{{Name: payer.Scope, Scale: 1}}
	}

	return []cpratelimit.Key{
		{Name: "account:" + payer.Account, Scale: 1},
		{Name: "scope:" + payer.Scope, Scale: b.scopeMultiplier},
	}
}

// Boosted is the bucket a bonus widens: the first key, never the scope's shared one.
func (b Buckets) Boosted(payer Payer) string {
	return b.Keys(payer)[0].Name
}

// Tightest is the reading that allows the fewest clicks now, so a player behind a busy scope sees the real
// limit. Boosted is the first bucket's, which is the one a bonus widens.
func Tightest(states []cpratelimit.State) cpratelimit.State {
	tightest := states[0]
	for _, state := range states[1:] {
		if state.Tokens < tightest.Tokens || (state.Tokens == tightest.Tokens && state.Capacity < tightest.Capacity) {
			tightest = state
		}
	}
	tightest.Boosted = states[0].Boosted

	return tightest
}
