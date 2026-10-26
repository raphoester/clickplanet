package clicks

import (
	"context"
	"net/netip"
	"time"

	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpctx"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpipscope"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

// Throttle is what the rule set in force says of the click buckets.
type Throttle struct {
	PerSecond float64
	Burst     int

	ScopeMultiplier      float64
	V6ScopeMultiplier    float64
	GuestScopeMultiplier float64
	LinkedMultiplier     float64

	NewAccountClicks float64
}

type ThrottleInForce interface {
	Throttle() Throttle
}

func NewBuckets(rules ThrottleInForce) Buckets {
	return Buckets{rules: rules}
}

type Buckets struct {
	rules ThrottleInForce
}

type SharedWith int

const (
	SharedWithNobody SharedWith = iota
	SharedWithGuests
	SharedWithScope
)

func (b Buckets) BudgetOf(payer Payer, states []cpratelimit.State, price Price) Budget {
	i := tightest(states)

	// Read against the payer's own bank, which pools lists first: a bucket it shares is a ceiling, not a bigger bank.
	state := states[i]
	state.Capacity = states[0].Capacity

	return Budget{
		State:            state,
		SharedWith:       b.pools(payer, price)[i].sharedWith,
		Price:            price,
		LinkedMultiplier: b.rules.Throttle().LinkedMultiplier,
	}
}

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

func (b Buckets) Keys(payer Payer, price Price) []cpratelimit.Key {
	pools := b.pools(payer, price)

	keys := make([]cpratelimit.Key, len(pools))
	for i, pool := range pools {
		keys[i] = pool.key
	}

	return keys
}

func (b Buckets) Bank(payer Payer) []cpratelimit.Key {
	var keys []cpratelimit.Key
	for _, pool := range b.pools(payer, Price{}) {
		if pool.sharedWith != SharedWithScope {
			keys = append(keys, pool.key)
		}
	}

	return keys
}

func (b Buckets) Ceiling(payer Payer) []cpratelimit.Key {
	var keys []cpratelimit.Key
	for _, pool := range b.pools(payer, Price{}) {
		if pool.sharedWith == SharedWithScope {
			keys = append(keys, pool.key)
		}
	}

	return keys
}

type pool struct {
	key        cpratelimit.Key
	sharedWith SharedWith
}

func (b Buckets) pools(payer Payer, price Price) []pool {
	pace := 1 / max(price.Slowdown, 1)

	if payer.Account == "" {
		// Its own key, not "scope:": the two scale differently.
		return []pool{{key: cpratelimit.Key{Name: payer.Scope, Scale: 1, Pace: pace}, sharedWith: SharedWithNobody}}
	}

	throttle := b.rules.Throttle()
	scope := pool{key: cpratelimit.Key{Name: "scope:" + payer.Scope, Scale: throttle.scopeScale(payer.Scope)}, sharedWith: SharedWithScope}

	if payer.Linked {
		return []pool{{key: throttle.own(payer, pace*throttle.LinkedMultiplier), sharedWith: SharedWithNobody}, scope}
	}

	return []pool{
		{key: throttle.own(payer, pace), sharedWith: SharedWithNobody},
		{key: cpratelimit.Key{Name: "guests:" + payer.Scope, Scale: throttle.GuestScopeMultiplier, Pace: pace}, sharedWith: SharedWithGuests},
		scope,
	}
}

func (t Throttle) scopeScale(scope string) float64 {
	if prefix, err := netip.ParsePrefix(scope); err == nil && prefix.Addr().Is6() {
		return t.V6ScopeMultiplier
	}
	return t.ScopeMultiplier
}

func (t Throttle) own(payer Payer, pace float64) cpratelimit.Key {
	key := cpratelimit.Key{Name: "account:" + payer.Account, Scale: 1, Pace: pace}
	if !payer.Created.IsZero() {
		key.Since, key.Start = payer.Created, t.NewAccountClicks
	}

	return key
}

func tightest(states []cpratelimit.State) int {
	i := 0
	for j, state := range states[1:] {
		if state.Tokens < states[i].Tokens || (state.Tokens == states[i].Tokens && state.Capacity < states[i].Capacity) {
			i = j + 1
		}
	}
	return i
}
