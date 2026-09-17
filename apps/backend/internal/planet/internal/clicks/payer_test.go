package clicks_test

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cpratelimit"
)

func TestTheScopeMultiplierDefaultsToTenAndRefusesLessThanOneAccount(t *testing.T) {
	payer := clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}

	assert.InDelta(t, 10.0, clicks.ThrottleConfig{}.Buckets().Keys(payer)[1].Scale, 1e-9)
	assert.InDelta(t, 3.0, clicks.ThrottleConfig{ScopeMultiplier: 3}.Buckets().Keys(payer)[1].Scale, 1e-9)

	assert.NoError(t, clicks.ThrottleConfig{}.Validate())
	assert.NoError(t, clicks.ThrottleConfig{ScopeMultiplier: 1}.Validate())
	for _, bad := range []float64{0.5, -1, math.NaN()} {
		assert.ErrorContains(t, clicks.ThrottleConfig{ScopeMultiplier: bad}.Validate(), "rateLimiter.scopeMultiplier")
	}
}

func TestABonusWidensTheAccountsBucketOrTheScopesWithNoAccount(t *testing.T) {
	buckets := clicks.ThrottleConfig{}.Buckets()

	assert.Equal(t, "account:a-guest", buckets.Boosted(clicks.Payer{Scope: "1.2.3.4", Account: "a-guest"}))
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
