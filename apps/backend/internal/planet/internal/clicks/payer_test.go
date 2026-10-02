package clicks_test

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

func TestTheScopeMultiplierDefaultsToTenAndRefusesLessThanOneAccount(t *testing.T) {
	payer := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}

	assert.InDelta(t, 10.0, clicks.ThrottleConfig{}.Buckets().Keys(payer, clicks.Price{})[2].Scale, 1e-9)
	assert.InDelta(t, 3.0, clicks.ThrottleConfig{ScopeMultiplier: 3}.Buckets().Keys(payer, clicks.Price{})[2].Scale, 1e-9)

	assert.NoError(t, clicks.ThrottleConfig{}.Validate())
	assert.NoError(t, clicks.ThrottleConfig{ScopeMultiplier: 1}.Validate())
	for _, bad := range []float64{0.5, -1, math.NaN()} {
		assert.ErrorContains(t, clicks.ThrottleConfig{ScopeMultiplier: bad}.Validate(), "rateLimiter.scopeMultiplier")
	}
}

func TestASignedInAccountRefillsItsOwnBucketFasterAtTheSameSize(t *testing.T) {
	guest := clicks.Payer{Scope: "1.2.3.4", Account: "an-account"}
	linked := clicks.Payer{Scope: "1.2.3.4", Account: "an-account", Linked: true}

	assert.Equal(t, []cpratelimit.Key{
		{Name: "account:an-account", Scale: 1, Pace: 1},
		{Name: "guests:1.2.3.4", Scale: 1, Pace: 1},
		{Name: "scope:1.2.3.4", Scale: 10},
	}, clicks.ThrottleConfig{}.Buckets().Keys(guest, clicks.Price{}))
	assert.Equal(t, []cpratelimit.Key{{Name: "account:an-account", Scale: 1, Pace: 2}, {Name: "scope:1.2.3.4", Scale: 10}},
		clicks.ThrottleConfig{}.Buckets().Keys(linked, clicks.Price{}), "the same bucket, and none shared with guests")
	assert.InDelta(t, 3.0, clicks.ThrottleConfig{LinkedMultiplier: 3}.Buckets().Keys(linked, clicks.Price{})[0].Pace, 1e-9)
	assert.InDelta(t, 2.0, clicks.ThrottleConfig{}.Buckets().BudgetOf(guest, make([]cpratelimit.State, 3), clicks.Price{}).LinkedMultiplier, 1e-9,
		"every budget says what signing in is worth")

	assert.Equal(t, []cpratelimit.Key{{Name: "1.2.3.4", Scale: 1, Pace: 1}},
		clicks.ThrottleConfig{}.Buckets().Keys(clicks.Payer{Scope: "1.2.3.4", Linked: true}, clicks.Price{}), "no account is never linked")

	require.NoError(t, clicks.ThrottleConfig{LinkedMultiplier: 1}.Validate())
	for _, bad := range []float64{0.5, -1, math.NaN()} {
		assert.ErrorContains(t, clicks.ThrottleConfig{LinkedMultiplier: bad}.Validate(), "rateLimiter.linkedMultiplier")
	}
}

func TestABigCountrySlowsThePayersOwnRefillAndNotTheScopes(t *testing.T) {
	buckets := clicks.ThrottleConfig{}.Buckets()
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

func TestTheGuestScopeMultiplierDefaultsToOneAndRefusesLessThanOneAccount(t *testing.T) {
	payer := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}

	assert.InDelta(t, 1.0, clicks.ThrottleConfig{}.Buckets().Keys(payer, clicks.Price{})[1].Scale, 1e-9)
	assert.InDelta(t, 2.0, clicks.ThrottleConfig{GuestScopeMultiplier: 2}.Buckets().Keys(payer, clicks.Price{})[1].Scale, 1e-9)

	require.NoError(t, clicks.ThrottleConfig{GuestScopeMultiplier: 1}.Validate())
	for _, bad := range []float64{0.5, -1, math.NaN()} {
		assert.ErrorContains(t, clicks.ThrottleConfig{GuestScopeMultiplier: bad}.Validate(), "rateLimiter.guestScopeMultiplier")
	}
}

func TestANewAccountsOwnBucketEarnsItsBankFromWhenItWasMade(t *testing.T) {
	made := time.Date(2026, 10, 1, 13, 37, 0, 0, time.UTC)
	start := 10.0
	buckets := clicks.ThrottleConfig{NewAccountClicks: &start}.Buckets()

	guest := buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest", Created: made}, clicks.Price{})
	assert.Equal(t, cpratelimit.Key{Name: "account:a-guest", Scale: 1, Pace: 1, Since: made, Start: 10}, guest[0])
	assert.Zero(t, guest[1].Since, "the network's guests share a bucket nobody started")
	assert.Zero(t, guest[2].Since)

	linked := buckets.Bank(clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true, Created: made})
	assert.Equal(t, made, linked[0].Since, "a refill fills the same bucket")

	assert.Zero(t, buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}, clicks.Price{})[0].Since,
		"an account that does not say when it was made starts full")
	assert.Zero(t, clicks.ThrottleConfig{}.Buckets().Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest", Created: made}, clicks.Price{})[0].Since,
		"unset, every account starts full")

	zero := 0.0
	require.NoError(t, clicks.ThrottleConfig{NewAccountClicks: &zero}.Validate())
	for _, bad := range []float64{-1, math.NaN()} {
		assert.ErrorContains(t, clicks.ThrottleConfig{NewAccountClicks: &bad}.Validate(), "rateLimiter.newAccountClicks")
	}
}

func TestARefillFillsTheBankNeverTheScopes(t *testing.T) {
	buckets := clicks.ThrottleConfig{}.Buckets()

	assert.Equal(t, []cpratelimit.Key{{Name: "account:a-guest", Scale: 1, Pace: 1}, {Name: "guests:1.2.3.4", Scale: 1, Pace: 1}},
		buckets.Bank(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}))
	assert.Equal(t, []cpratelimit.Key{{Name: "account:a-player", Scale: 1, Pace: 2}},
		buckets.Bank(clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true}))
	assert.Equal(t, []cpratelimit.Key{{Name: "1.2.3.4", Scale: 1, Pace: 1}}, buckets.Bank(clicks.Payer{Scope: "1.2.3.4"}))
}

func TestTheBudgetIsTheTightestBucketAndSaysWhoShares(t *testing.T) {
	buckets := clicks.ThrottleConfig{}.Buckets()
	guest := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}
	linked := clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true}

	own := cpratelimit.State{Tokens: 8, Capacity: 10, PerSecond: 1}
	guests := cpratelimit.State{Tokens: 3, Capacity: 10, PerSecond: 1}
	scope := cpratelimit.State{Tokens: 4, Capacity: 100, PerSecond: 10}

	budget := buckets.BudgetOf(guest, []cpratelimit.State{own, guests, scope}, clicks.Price{})
	assert.Equal(t, guests, budget.State)
	assert.Equal(t, clicks.SharedWithGuests, budget.SharedWith)

	budget = buckets.BudgetOf(linked, []cpratelimit.State{own, scope}, clicks.Price{})
	assert.Equal(t, scope, budget.State)
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

func TestAPayerIsPricedByItsAccountsTallyOrItsScopesWithNoAccount(t *testing.T) {
	assert.Equal(t, clicks.AccountAllegianceKey("ada"), clicks.Payer{Scope: "2001:db8::/64", Account: "ada"}.AllegianceKey())
	assert.Equal(t, clicks.ScopeAllegianceKey("2001:db8::/64"), clicks.Payer{Scope: "2001:db8::/64"}.AllegianceKey())
}

func TestATakeCountsForItsAccountAndItsScope(t *testing.T) {
	assert.Equal(t,
		[]clicks.AllegianceKey{clicks.AccountAllegianceKey("ada"), clicks.ScopeAllegianceKey("2001:db8::/64")},
		clicks.Payer{Scope: "2001:db8::/64", Account: "ada"}.AllegianceKeys())
	assert.Equal(t, []clicks.AllegianceKey{clicks.ScopeAllegianceKey("2001:db8::/64")},
		clicks.Payer{Scope: "2001:db8::/64"}.AllegianceKeys())
	assert.NotEqual(t, clicks.AccountAllegianceKey("x"), clicks.ScopeAllegianceKey("x"), "an account and a scope never share a tally")
}
