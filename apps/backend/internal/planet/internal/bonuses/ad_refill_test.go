package bonuses_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/bonuses"
	"github.com/raphoester/clickplanet.lol-backend/internal/planet/internal/clicks"
	"github.com/raphoester/clickplanet.lol-backend/internal/shared/cptime"
)

var (
	guest     = clicks.Payer{Scope: "203.0.113.7", Account: "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11"}
	housemate = clicks.Payer{Scope: "203.0.113.7", Account: "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"}
	player    = clicks.Payer{Scope: "203.0.113.7", Account: "5e0f9a1b-2c3d-4e5f-8a9b-0c1d2e3f4a5b", Linked: true}
)

func hour(h int) time.Time {
	return time.Date(2026, 10, 30, h, 0, 0, 0, time.UTC)
}

func adRefills() (bonuses.AdRefills, *bonuses.Rules, *cptime.FixedClock) {
	rules := bonuses.TestRules()
	clock := cptime.NewFixedClock(hour(12).Add(34 * time.Minute))
	return bonuses.NewAdRefills(bonuses.NewMemoryGrants(), func() bonuses.Rules { return rules }, clock), &rules, clock
}

func availableAt(t *testing.T, refills bonuses.AdRefills, payer clicks.Payer) time.Time {
	t.Helper()

	at, err := refills.AvailableAt(t.Context(), payer)
	require.NoError(t, err)
	return at
}

func TestABankHasOneAdRefillInEachSpan(t *testing.T) {
	refills, _, clock := adRefills()
	assert.Equal(t, hour(12), availableAt(t, refills, guest), "a span of an hour starts on the hour")

	require.NoError(t, refills.Record(t.Context(), guest))

	assert.Equal(t, hour(13), availableAt(t, refills, guest))
	err := refills.Record(t.Context(), guest)
	require.ErrorIs(t, err, bonuses.ErrAdRefillUsed)
	assert.Equal(t, bonuses.AdRefillRefusal{Until: hour(13)}, err)

	clock.Advance(26 * time.Minute)
	assert.Equal(t, hour(13), availableAt(t, refills, guest))
	require.NoError(t, refills.Record(t.Context(), guest))
}

func TestTheGuestsBehindOneNetworkShareIt(t *testing.T) {
	refills, _, _ := adRefills()
	require.NoError(t, refills.Record(t.Context(), guest))

	assert.Equal(t, hour(13), availableAt(t, refills, housemate))
	require.ErrorIs(t, refills.Record(t.Context(), housemate), bonuses.ErrAdRefillUsed)
}

func TestASignedInPlayerHasItsOwnAndLeavesTheGuestsTheirs(t *testing.T) {
	refills, _, _ := adRefills()
	require.NoError(t, refills.Record(t.Context(), guest))

	assert.Equal(t, hour(12), availableAt(t, refills, player))
	require.NoError(t, refills.Record(t.Context(), player))

	refills, _, _ = adRefills()
	require.NoError(t, refills.Record(t.Context(), player))
	require.NoError(t, refills.Record(t.Context(), guest))
}

func TestAGuestThatSignsInKeepsTheOneItHad(t *testing.T) {
	refills, _, _ := adRefills()
	require.NoError(t, refills.Record(t.Context(), guest))

	signedIn := guest
	signedIn.Linked = true
	require.ErrorIs(t, refills.Record(t.Context(), signedIn), bonuses.ErrAdRefillUsed)
}

func TestALongerSpanThatStartsWithTheOldOneGivesNoSecondRefill(t *testing.T) {
	refills, rules, _ := adRefills()
	require.NoError(t, refills.Record(t.Context(), guest))

	rules.AdRefill.Every = 2 * time.Hour

	assert.Equal(t, hour(14), availableAt(t, refills, guest))
}

func TestAnAccountlessCallerMayHaveNone(t *testing.T) {
	refills, _, _ := adRefills()

	assert.Zero(t, availableAt(t, refills, clicks.Payer{Scope: "203.0.113.7"}))
}
