package standings_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/standings"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	seasonOneEnds  = time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)
)

func twoSeasons() calendar.Calendar {
	return calendar.New(calendar.Config{List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds, Finale: 2 * time.Hour},
		{Number: 1, EndsAt: seasonOneEnds, Finale: 2 * time.Hour},
	}})
}

func TestATakeCountsInTheSeasonWhoseIntervalHoldsIt(t *testing.T) {
	for _, tc := range []struct {
		at     time.Time
		season calendar.Number
	}{
		{time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 0},
		{seasonZeroEnds.Add(-time.Nanosecond), 0},
		{seasonZeroEnds, 1},
		{seasonOneEnds.Add(-time.Nanosecond), 1},
	} {
		season, ok := standings.Take{At: tc.at}.Season(twoSeasons())

		require.True(t, ok, tc.at)
		assert.Equal(t, tc.season, season, tc.at)
	}
}

func TestATakeInNoSeasonCountsNowhere(t *testing.T) {
	_, ok := standings.Take{At: seasonOneEnds}.Season(twoSeasons())
	assert.False(t, ok)

	_, ok = standings.Take{At: seasonZeroEnds}.Season(calendar.New(calendar.Config{}))
	assert.False(t, ok, "an empty calendar has no season")
}

func TestAnAccountIDIsAUUIDThatIsNotNil(t *testing.T) {
	account, err := standings.AccountIDOf("0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11")
	require.NoError(t, err)
	assert.Equal(t, "0b6d4f7e-5d7c-4a36-9a51-3f1f8f0c2a11", account.String())

	for _, value := range []string{"", "not-an-account", "00000000-0000-0000-0000-000000000000"} {
		_, err := standings.AccountIDOf(value)
		assert.ErrorIs(t, err, standings.ErrInvalidAccount, value)
	}
}
