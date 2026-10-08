package fronts_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/fronts"
	"github.com/raphoester/clickplanet.lol-backend/internal/player/internal/players"
)

func take(t *testing.T, country, previous fronts.Country) fronts.Take {
	t.Helper()

	take, err := fronts.NewTake(players.AccountID{15: 1}, country, previous)
	require.NoError(t, err)
	return take
}

func TestATakeWithNoCountryIsRefused(t *testing.T) {
	_, err := fronts.NewTake(players.AccountID{15: 1}, "", "de")

	assert.ErrorIs(t, err, fronts.ErrNoCountry)
}

func TestATakeIsAgainstTheCountryThatHeldTheTile(t *testing.T) {
	against, ok := take(t, "fr", "de").Against()

	assert.True(t, ok)
	assert.Equal(t, fronts.Country("de"), against)
}

func TestATileNobodyHeldIsAgainstNobody(t *testing.T) {
	_, ok := take(t, "fr", "").Against()

	assert.False(t, ok)
}

func TestATileTheFlagAlreadyHeldIsAgainstNobody(t *testing.T) {
	_, ok := take(t, "fr", "fr").Against()

	assert.False(t, ok)
}

func TestEachTakeCountsForItsFlagAndAgainstTheTilesOwner(t *testing.T) {
	tally := fronts.TallyOf(nil, nil).
		WithTake(take(t, "fr", "de")).
		WithTake(take(t, "fr", "de")).
		WithTake(take(t, "fr", "")).
		WithTake(take(t, "it", "fr"))

	assert.Equal(t, map[fronts.Country]uint64{"fr": 3, "it": 1}, tally.PlaysFor())
	assert.Equal(t, map[fronts.Country]uint64{"de": 2, "fr": 1}, tally.PlaysAgainst())
}

func TestATallyWithATakeIsACopy(t *testing.T) {
	before := fronts.TallyOf(map[fronts.Country]uint64{"fr": 1}, nil)

	_ = before.WithTake(take(t, "fr", "de"))

	assert.Equal(t, fronts.TallyOf(map[fronts.Country]uint64{"fr": 1}, nil), before)
}

func TestAZeroIsNoTile(t *testing.T) {
	assert.Equal(t,
		fronts.TallyOf(map[fronts.Country]uint64{"fr": 2}, nil),
		fronts.TallyOf(map[fronts.Country]uint64{"fr": 2, "de": 0}, map[fronts.Country]uint64{"it": 0}))
}
