package clicks_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

func TestTheScopeMultiplierDefaultsToTenAndRefusesLessThanOneAccount(t *testing.T) {
	payer := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}

	assert.InDelta(t, 10.0, clicks.ThrottleConfig{}.Buckets().Keys(payer, clicks.Price{})[1].Scale, 1e-9)
	assert.InDelta(t, 3.0, clicks.ThrottleConfig{ScopeMultiplier: 3}.Buckets().Keys(payer, clicks.Price{})[1].Scale, 1e-9)

	assert.NoError(t, clicks.ThrottleConfig{}.Validate())
	assert.NoError(t, clicks.ThrottleConfig{ScopeMultiplier: 1}.Validate())
	for _, bad := range []float64{0.5, -1, math.NaN()} {
		assert.ErrorContains(t, clicks.ThrottleConfig{ScopeMultiplier: bad}.Validate(), "rateLimiter.scopeMultiplier")
	}
}

func TestASignedInAccountRefillsItsOwnBucketFasterAtTheSameSize(t *testing.T) {
	guest := clicks.Payer{Scope: "1.2.3.4", Account: "an-account"}
	linked := clicks.Payer{Scope: "1.2.3.4", Account: "an-account", Linked: true}

	assert.Equal(t, []cpratelimit.Key{{Name: "account:an-account", Scale: 1, Pace: 1}, {Name: "scope:1.2.3.4", Scale: 10}},
		clicks.ThrottleConfig{}.Buckets().Keys(guest, clicks.Price{}))
	assert.Equal(t, []cpratelimit.Key{{Name: "account:an-account", Scale: 1, Pace: 2}, {Name: "scope:1.2.3.4", Scale: 10}},
		clicks.ThrottleConfig{}.Buckets().Keys(linked, clicks.Price{}), "the same bucket: signing in keeps the bank")
	assert.InDelta(t, 3.0, clicks.ThrottleConfig{LinkedMultiplier: 3}.Buckets().Keys(linked, clicks.Price{})[0].Pace, 1e-9)
	assert.InDelta(t, 2.0, clicks.ThrottleConfig{}.Buckets().BudgetOf(cpratelimit.State{Capacity: 10}, clicks.Price{}).LinkedMultiplier, 1e-9,
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
	assert.Zero(t, guest[1].Pace, "the scope's bucket keeps its plain rate")

	linked := buckets.Keys(clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true}, price)
	assert.InDelta(t, 2/1.5, linked[0].Pace, 1e-9)

	anonymous := buckets.Keys(clicks.Payer{Scope: "1.2.3.4"}, price)
	assert.InDelta(t, 1/1.5, anonymous[0].Pace, 1e-9)
}

func TestABonusSpeedsUpTheAccountsBucketOrTheScopesWithNoAccount(t *testing.T) {
	buckets := clicks.ThrottleConfig{}.Buckets()

	assert.Equal(t, "account:a-guest", buckets.Boosted(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}))
	assert.Equal(t, "account:a-player", buckets.Boosted(clicks.Payer{Scope: "1.2.3.4", Account: "a-player", Linked: true}))
	assert.Equal(t, "1.2.3.4", buckets.Boosted(clicks.Payer{Scope: "1.2.3.4"}))
}

func TestTheTightestBucketIsTheOneWithTheFewestTokens(t *testing.T) {
	account := cpratelimit.State{Tokens: 8, Capacity: 30, PerSecond: 3, Boosted: true}
	scope := cpratelimit.State{Tokens: 4, Capacity: 100, PerSecond: 10}

	assert.Equal(t, cpratelimit.State{Tokens: 4, Capacity: 100, PerSecond: 10, Boosted: true},
		clicks.Tightest([]cpratelimit.State{account, scope}), "boosted is the account's, whichever bucket is tighter")

	full := cpratelimit.State{Tokens: 10, Capacity: 10, PerSecond: 1}
	fullScope := cpratelimit.State{Tokens: 10, Capacity: 100, PerSecond: 10}
	assert.Equal(t, full, clicks.Tightest([]cpratelimit.State{fullScope, full}), "a tie goes to the smaller bucket")
}
