package clicks

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// ThrottleConfig is the `rateLimiter:` block: one account's allowance, and how many of them its scope may spend.
type ThrottleConfig struct {
	cpratelimit.Config `koanf:",squash"`

	// ScopeMultiplier is the scope's burst and rate over one account's: many players can share one address.
	ScopeMultiplier float64

	// GuestScopeMultiplier is the burst and rate of the bucket every guest behind one scope shares: at 1, ten tabs are one bank.
	GuestScopeMultiplier float64

	// LinkedMultiplier is a linked account's refill rate over a guest's: signing in is worth clicking faster.
	// The bank is the same size.
	LinkedMultiplier float64

	// NewAccountClicks is the bank of an account just made, which earns the rest; unset, it starts full.
	NewAccountClicks *float64
}

const (
	defaultScopeMultiplier      = 10
	defaultGuestScopeMultiplier = 1
	defaultLinkedMultiplier     = 2
)

// Validate refuses a scope that holds less than one account: every account behind it would be throttled by the scope.
// It refuses a linked account slower than a guest too: signing in would be a penalty.
func (c ThrottleConfig) Validate() error {
	if start := c.NewAccountClicks; start != nil && (math.IsNaN(*start) || *start < 0) {
		return fmt.Errorf("rateLimiter.newAccountClicks is %v: it must be 0 or more, or unset for a full bank", *start)
	}

	for _, multiplier := range []struct {
		name         string
		value        float64
		defaultValue int
	}{
		{"scopeMultiplier", c.ScopeMultiplier, defaultScopeMultiplier},
		{"guestScopeMultiplier", c.GuestScopeMultiplier, defaultGuestScopeMultiplier},
		{"linkedMultiplier", c.LinkedMultiplier, defaultLinkedMultiplier},
	} {
		if math.IsNaN(multiplier.value) || multiplier.value < 0 || (multiplier.value > 0 && multiplier.value < 1) {
			return fmt.Errorf("rateLimiter.%s is %v: it must be 1 or more, or unset for %d",
				multiplier.name, multiplier.value, multiplier.defaultValue)
		}
	}

	return nil
}

// Buckets is the throttle policy: which buckets a payer spends from.
func (c ThrottleConfig) Buckets() Buckets {
	return Buckets{
		scopeMultiplier:  orDefault(c.ScopeMultiplier, defaultScopeMultiplier),
		guestMultiplier:  orDefault(c.GuestScopeMultiplier, defaultGuestScopeMultiplier),
		linkedMultiplier: orDefault(c.LinkedMultiplier, defaultLinkedMultiplier),
		newAccountClicks: c.NewAccountClicks,
	}
}

func orDefault(value float64, defaultValue int) float64 {
	if value <= 0 {
		return float64(defaultValue)
	}
	return value
}

type Buckets struct {
	scopeMultiplier  float64
	guestMultiplier  float64
	linkedMultiplier float64
	newAccountClicks *float64
}

// SharedWith is who else spends from a bucket: nobody, the scope's guests, or every player behind the scope.
type SharedWith int

const (
	SharedWithNobody SharedWith = iota
	SharedWithGuests
	SharedWithScope
)

// BudgetOf is the tightest of the payer's readings, in the order Keys named them, and who else spends from it.
func (b Buckets) BudgetOf(payer Payer, states []cpratelimit.State, price Price) Budget {
	i := tightest(states)

	return Budget{
		State:            states[i],
		SharedWith:       b.pools(payer, price)[i].sharedWith,
		Price:            price,
		LinkedMultiplier: b.linkedMultiplier,
	}
}

// Payer is who a click is charged to: the scope it comes from, the account its token names if any, whether
// that account signed in with a provider, and when it was made (zero when unknown).
type Payer struct {
	Scope   string
	Account string
	Linked  bool
	Created time.Time
}

func PayerOf(ctx context.Context) Payer {
	return Payer{
		Scope:   cpipscope.Of(cpctx.GetSourceIP(ctx)),
		Account: cpctx.GetAccount(ctx),
		Linked:  cpctx.GetLinked(ctx),
		Created: cpctx.GetAccountCreated(ctx),
	}
}

// Keys is the buckets a click spends from, the payer's own first.
func (b Buckets) Keys(payer Payer, price Price) []cpratelimit.Key {
	pools := b.pools(payer, price)

	keys := make([]cpratelimit.Key, len(pools))
	for i, pool := range pools {
		keys[i] = pool.key
	}

	return keys
}

// Bank is the buckets a refill fills: every one but the scope's, which every player behind it shares.
func (b Buckets) Bank(payer Payer) []cpratelimit.Key {
	var keys []cpratelimit.Key
	for _, pool := range b.pools(payer, Price{}) {
		if pool.sharedWith != SharedWithScope {
			keys = append(keys, pool.key)
		}
	}

	return keys
}

type pool struct {
	key        cpratelimit.Key
	sharedWith SharedWith
}

// pools is the payer's own bucket first, then a guest's scope's guests' at the guest multiplier, then the scope's at
// the scope multiplier. With no account it is the scope's bucket alone at one, as it was before accounts: a
// separate bucket, so a scale never changes under a key.
//
// The payer's own bucket refills at its pace for a click for this price: the linked multiplier for an account
// that signed in, divided by the country's slowdown. Its burst never moves, so the bank a player sees is the same
// whatever it plays and whether or not it signed in. The guests' bucket takes a guest's pace, or ten tabs on a big
// country would refill faster than one. The scope's bucket is a ceiling shared by players of every flag, so it
// refills at its plain rate.
func (b Buckets) pools(payer Payer, price Price) []pool {
	pace := 1 / max(price.Slowdown, 1)

	if payer.Account == "" {
		return []pool{{key: cpratelimit.Key{Name: payer.Scope, Scale: 1, Pace: pace}, sharedWith: SharedWithNobody}}
	}

	scope := pool{key: cpratelimit.Key{Name: "scope:" + payer.Scope, Scale: b.scopeMultiplier}, sharedWith: SharedWithScope}

	if payer.Linked {
		return []pool{{key: b.own(payer, pace*b.linkedMultiplier), sharedWith: SharedWithNobody}, scope}
	}

	return []pool{
		{key: b.own(payer, pace), sharedWith: SharedWithNobody},
		{key: cpratelimit.Key{Name: "guests:" + payer.Scope, Scale: b.guestMultiplier, Pace: pace}, sharedWith: SharedWithGuests},
		scope,
	}
}

// own is the account's bucket; a new account's earns its bank from when it was made.
func (b Buckets) own(payer Payer, pace float64) cpratelimit.Key {
	key := cpratelimit.Key{Name: "account:" + payer.Account, Scale: 1, Pace: pace}
	if b.newAccountClicks != nil && !payer.Created.IsZero() {
		key.Since, key.Start = payer.Created, *b.newAccountClicks
	}

	return key
}

// tightest is the reading that allows the fewest clicks now; on a tie the smaller bucket, then the payer's own.
func tightest(states []cpratelimit.State) int {
	i := 0
	for j, state := range states[1:] {
		if state.Tokens < states[i].Tokens || (state.Tokens == states[i].Tokens && state.Capacity < states[i].Capacity) {
			i = j + 1
		}
	}
	return i
}
