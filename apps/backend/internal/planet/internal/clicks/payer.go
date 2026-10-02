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

type ThrottleConfig struct {
	cpratelimit.Config `koanf:",squash"`

	ScopeMultiplier float64

	GuestScopeMultiplier float64

	LinkedMultiplier float64

	NewAccountClicks *float64
}

const (
	defaultScopeMultiplier      = 10
	defaultGuestScopeMultiplier = 1
	defaultLinkedMultiplier     = 2
)

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

type SharedWith int

const (
	SharedWithNobody SharedWith = iota
	SharedWithGuests
	SharedWithScope
)

func (b Buckets) BudgetOf(payer Payer, states []cpratelimit.State, price Price) Budget {
	i := tightest(states)

	return Budget{
		State:            states[i],
		SharedWith:       b.pools(payer, price)[i].sharedWith,
		Price:            price,
		LinkedMultiplier: b.linkedMultiplier,
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

type pool struct {
	key        cpratelimit.Key
	sharedWith SharedWith
}

func (b Buckets) pools(payer Payer, price Price) []pool {
	pace := 1 / max(price.Slowdown, 1)

	if payer.Account == "" {
		// Its own key, not "scope:": a bucket's Scale is fixed when it is first made.
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

func (b Buckets) own(payer Payer, pace float64) cpratelimit.Key {
	key := cpratelimit.Key{Name: "account:" + payer.Account, Scale: 1, Pace: pace}
	if b.newAccountClicks != nil && !payer.Created.IsZero() {
		key.Since, key.Start = payer.Created, *b.newAccountClicks
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
