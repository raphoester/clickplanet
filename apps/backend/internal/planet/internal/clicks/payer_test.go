package clicks_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

func bucketsWith(change func(*clicks.Throttle)) clicks.Buckets {
	throttle := clicks.TestThrottle()
	change(&throttle)
	return clicks.NewBuckets(func() clicks.Throttle { return throttle })
}

func plainBuckets() clicks.Buckets {
	return bucketsWith(func(*clicks.Throttle) {})
}

func TestTheScopesBucketHoldsTheScopeMultipliersAccounts(t *testing.T) {
	payer := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}

	assert.InDelta(t, 10.0, plainBuckets().Keys(payer, clicks.Price{})[2].Scale, 1e-9)
	assert.InDelta(t, 3.0, bucketsWith(func(throttle *clicks.Throttle) { throttle.ScopeMultiplier = 3 }).Keys(payer, clicks.Price{})[2].Scale, 1e-9)
}

func TestTheBucketsFollowTheRulesInForce(t *testing.T) {
	payer := clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true}
	throttle := clicks.TestThrottle()
	buckets := clicks.NewBuckets(func() clicks.Throttle { return throttle })

	throttle.ScopeMultiplier = 4
	throttle.LinkedMultiplier = 3

	keys := buckets.Keys(payer, clicks.Price{})
	assert.InDelta(t, 3.0, keys[0].Pace, 1e-9)
	assert.InDelta(t, 4.0, keys[1].Scale, 1e-9)
	assert.InDelta(t, 3.0, buckets.BudgetOf(payer, make([]cpratelimit.State, 2), clicks.Price{}).LinkedMultiplier, 1e-9)
}

func TestAnIPv6NetworkHasAScopeMultiplierOfItsOwn(t *testing.T) {
	v6 := clicks.Payer{Scope: "2001:db8:1:2::/64", Account: "a-player", Linked: true}
	v4 := clicks.Payer{Scope: "192.0.2.1", Account: "a-player", Linked: true}

	capped := bucketsWith(func(throttle *clicks.Throttle) { throttle.V6ScopeMultiplier = 4 })
	assert.Equal(t, cpratelimit.Key{Name: "scope:2001:db8:1:2::/64", Scale: 4}, capped.Keys(v6, clicks.Price{})[1])
	assert.InDelta(t, 10.0, capped.Keys(v4, clicks.Price{})[1].Scale, 1e-9, "an IPv4 address can be a school's NAT")
	assert.InDelta(t, 4.0, capped.Keys(clicks.Payer{Scope: "2001:db8:1:2::/64", Account: "a-guest"}, clicks.Price{})[2].Scale, 1e-9)
}

func TestASignedInAccountRefillsItsOwnBucketFasterAtTheSameSize(t *testing.T) {
	guest := clicks.Payer{Scope: "1.2.3.4", Account: "an-account"}
	linked := clicks.Payer{Scope: "1.2.3.4", Account: "an-account", Linked: true}

	assert.Equal(t, []cpratelimit.Key{
		{Name: "account:an-account", Scale: 1, Pace: 1},
		{Name: "guests:1.2.3.4", Scale: 1, Pace: 1},
		{Name: "scope:1.2.3.4", Scale: 10},
	}, plainBuckets().Keys(guest, clicks.Price{}))
	assert.Equal(t, []cpratelimit.Key{{Name: "account:an-account", Scale: 1, Pace: 2}, {Name: "scope:1.2.3.4", Scale: 10}},
		plainBuckets().Keys(linked, clicks.Price{}), "the same bucket, and none shared with guests")
	assert.InDelta(t, 3.0, bucketsWith(func(throttle *clicks.Throttle) { throttle.LinkedMultiplier = 3 }).Keys(linked, clicks.Price{})[0].Pace, 1e-9)
	assert.InDelta(t, 2.0, plainBuckets().BudgetOf(guest, make([]cpratelimit.State, 3), clicks.Price{}).LinkedMultiplier, 1e-9,
		"every budget says what signing in is worth")

	assert.Equal(t, []cpratelimit.Key{{Name: "1.2.3.4", Scale: 1, Pace: 1}},
		plainBuckets().Keys(clicks.Payer{Scope: "1.2.3.4", Linked: true}, clicks.Price{}), "no account is never linked")
}

func TestABigCountrySlowsThePayersOwnRefillAndNotTheScopes(t *testing.T) {
	buckets := plainBuckets()
	price := clicks.Price{Slowdown: 1.5}

	guest := buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}, price)
	assert.InDelta(t, 1/1.5, guest[0].Pace, 1e-9)
	assert.InDelta(t, 1/1.5, guest[1].Pace, 1e-9, "ten tabs on a big country refill no faster than one")
	assert.Zero(t, guest[2].Pace, "the scope's bucket keeps its plain rate")

	linked := buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true}, price)
	assert.InDelta(t, 2/1.5, linked[0].Pace, 1e-9)

	anonymous := buckets.Keys(clicks.Payer{Scope: "1.2.3.4"}, price)
	assert.InDelta(t, 1/1.5, anonymous[0].Pace, 1e-9)
}

func TestTheGuestsBehindAnAddressShareTheGuestScopeMultipliersAccounts(t *testing.T) {
	payer := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}

	assert.InDelta(t, 1.0, plainBuckets().Keys(payer, clicks.Price{})[1].Scale, 1e-9)
	assert.InDelta(t, 2.0, bucketsWith(func(throttle *clicks.Throttle) { throttle.GuestScopeMultiplier = 2 }).Keys(payer, clicks.Price{})[1].Scale, 1e-9)
}

func TestANewAccountsOwnBucketEarnsItsBankFromWhenItWasMade(t *testing.T) {
	made := time.Date(2026, 10, 1, 13, 37, 0, 0, time.UTC)
	buckets := bucketsWith(func(throttle *clicks.Throttle) { throttle.Burst, throttle.NewAccountClicks = 60, 10 })

	guest := buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest", Created: made}, clicks.Price{})
	assert.Equal(t, cpratelimit.Key{Name: "account:a-guest", Scale: 1, Pace: 1, Since: made, Start: 10}, guest[0])
	assert.Zero(t, guest[1].Since, "the network's guests share a bucket nobody started")
	assert.Zero(t, guest[2].Since)

	linked := buckets.Bank(clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true, Created: made})
	assert.Equal(t, made, linked[0].Since, "a refill fills the same bucket")

	assert.Zero(t, buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}, clicks.Price{})[0].Since,
		"an account that does not say when it was made starts full")
}

func TestARefillFillsTheBankAndTheScopesIsItsCeiling(t *testing.T) {
	buckets := plainBuckets()
	guest := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}
	linked := clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true}

	assert.Equal(t, []cpratelimit.Key{{Name: "account:a-guest", Scale: 1, Pace: 1}, {Name: "guests:1.2.3.4", Scale: 1, Pace: 1}},
		buckets.Bank(guest))
	assert.Equal(t, []cpratelimit.Key{{Name: "account:a-player", Scale: 1, Pace: 2}}, buckets.Bank(linked))
	assert.Equal(t, []cpratelimit.Key{{Name: "1.2.3.4", Scale: 1, Pace: 1}}, buckets.Bank(clicks.Payer{Scope: "1.2.3.4"}))

	assert.Equal(t, []cpratelimit.Key{{Name: "scope:1.2.3.4", Scale: 10}}, buckets.Ceiling(guest))
	assert.Equal(t, []cpratelimit.Key{{Name: "scope:1.2.3.4", Scale: 10}}, buckets.Ceiling(linked))
	assert.Empty(t, buckets.Ceiling(clicks.Payer{Scope: "1.2.3.4"}), "no account spends from no shared bucket")
}

func TestTheBudgetIsTheTightestBucketAndSaysWhoShares(t *testing.T) {
	buckets := plainBuckets()
	guest := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}
	linked := clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true}

	own := cpratelimit.State{Tokens: 8, Capacity: 10, PerSecond: 1}
	guests := cpratelimit.State{Tokens: 3, Capacity: 10, PerSecond: 1}
	scope := cpratelimit.State{Tokens: 4, Capacity: 100, PerSecond: 10}

	budget := buckets.BudgetOf(guest, []cpratelimit.State{own, guests, scope}, clicks.Price{})
	assert.Equal(t, guests, budget.State)
	assert.Equal(t, clicks.SharedWithGuests, budget.SharedWith)

	budget = buckets.BudgetOf(linked, []cpratelimit.State{own, scope}, clicks.Price{})
	assert.Equal(t, cpratelimit.State{Tokens: 4, Capacity: 10, PerSecond: 10}, budget.State,
		"the scope's count and rate, read against the player's own bank")
	assert.Equal(t, clicks.SharedWithScope, budget.SharedWith)

	fullScope := cpratelimit.State{Tokens: 8, Capacity: 100, PerSecond: 10}
	budget = buckets.BudgetOf(linked, []cpratelimit.State{own, fullScope}, clicks.Price{})
	assert.Equal(t, own, budget.State, "a tie goes to the smaller bucket")
	assert.Equal(t, clicks.SharedWithNobody, budget.SharedWith)

	budget = buckets.BudgetOf(guest, []cpratelimit.State{own, own, fullScope}, clicks.Price{})
	assert.Equal(t, clicks.SharedWithNobody, budget.SharedWith, "a guest alone on its network is not told it shares")

	budget = buckets.BudgetOf(clicks.Payer{Scope: "1.2.3.4"}, []cpratelimit.State{guests}, clicks.Price{})
	assert.Equal(t, clicks.SharedWithNobody, budget.SharedWith, "no account reads the only bucket it has")
}
