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

	// LinkedMultiplier is a linked account's refill rate over a guest's: signing in is worth clicking faster.
	// The bank is the same size.
	LinkedMultiplier float64
}

const (
	defaultScopeMultiplier  = 10
	defaultLinkedMultiplier = 2
)

// Validate refuses a scope that holds less than one account: every account behind it would be throttled by the scope.
// It refuses a linked account slower than a guest too: signing in would be a penalty.
func (c ThrottleConfig) Validate() error {
	if math.IsNaN(c.ScopeMultiplier) || c.ScopeMultiplier < 0 || (c.ScopeMultiplier > 0 && c.ScopeMultiplier < 1) {
		return fmt.Errorf("rateLimiter.scopeMultiplier is %v: it must be 1 or more, or unset for %d",
			c.ScopeMultiplier, defaultScopeMultiplier)
	}
	if math.IsNaN(c.LinkedMultiplier) || c.LinkedMultiplier < 0 || (c.LinkedMultiplier > 0 && c.LinkedMultiplier < 1) {
		return fmt.Errorf("rateLimiter.linkedMultiplier is %v: it must be 1 or more, or unset for %d",
			c.LinkedMultiplier, defaultLinkedMultiplier)
	}

	return nil
}

// Buckets is the throttle policy: which buckets a payer spends from.
func (c ThrottleConfig) Buckets() Buckets {
	scope := c.ScopeMultiplier
	if scope <= 0 {
		scope = defaultScopeMultiplier
	}
	linked := c.LinkedMultiplier
	if linked <= 0 {
		linked = defaultLinkedMultiplier
	}

	return Buckets{scopeMultiplier: scope, linkedMultiplier: linked}
}

type Buckets struct {
	scopeMultiplier  float64
	linkedMultiplier float64
}

// BudgetOf is a reading with the price of the country asked about, and what signing in is worth, so a
// client can advertise it.
func (b Buckets) BudgetOf(state cpratelimit.State, price Price) Budget {
	return Budget{State: state, Price: price, LinkedMultiplier: b.linkedMultiplier}
}

// Payer is who a click is charged to: the scope it comes from, the account its token names if any, and whether
// that account signed in with a provider.
type Payer struct {
	Scope   string
	Account string
	Linked  bool
}

func PayerOf(ctx context.Context) Payer {
	return Payer{Scope: cpipscope.Of(cpctx.GetSourceIP(ctx)), Account: cpctx.GetAccount(ctx), Linked: cpctx.GetLinked(ctx)}
}

// Keys is the payer's own bucket first, then the scope's at the multiplier. With no account it is the
// scope's bucket alone at one, as it was before accounts: a separate bucket, so a scale never changes under a key.
//
// The payer's own bucket refills at its pace for a click for this price: the linked multiplier for an account
// that signed in, divided by the country's slowdown. Its burst never moves, so the bank a player sees is the same
// whatever it plays and whether or not it signed in. The scope's bucket is a ceiling shared by players of every
// flag, so it refills at its plain rate.
func (b Buckets) Keys(payer Payer, price Price) []cpratelimit.Key {
	pace := 1 / max(price.Slowdown, 1)

	if payer.Account == "" {
		return []cpratelimit.Key{{Name: payer.Scope, Scale: 1, Pace: pace}}
	}

	if payer.Linked {
		pace *= b.linkedMultiplier
	}

	return []cpratelimit.Key{
		{Name: "account:" + payer.Account, Scale: 1, Pace: pace},
		{Name: "scope:" + payer.Scope, Scale: b.scopeMultiplier},
	}
}

// Own is the caller's own bucket, the one a refill fills: the first key, never the scope's shared one.
func (b Buckets) Own(payer Payer) cpratelimit.Key {
	return b.Keys(payer, Price{})[0]
}

// Tightest is the reading that allows the fewest clicks now, so a player behind a busy scope sees the real
// limit.
func Tightest(states []cpratelimit.State) cpratelimit.State {
	tightest := states[0]
	for _, state := range states[1:] {
		if state.Tokens < tightest.Tokens || (state.Tokens == tightest.Tokens && state.Capacity < tightest.Capacity) {
			tightest = state
		}
	}
	return tightest
}
