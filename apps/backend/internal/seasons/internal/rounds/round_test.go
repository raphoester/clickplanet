package rounds_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/rounds"
)

func utc(day, hour, minute, second int) time.Time {
	return time.Date(2026, 10, day, hour, minute, second, 0, time.UTC)
}

var twoSeasons = calendar.New(calendar.Config{List: []calendar.Entry{
	{Number: 0, EndsAt: utc(31, 23, 0, 0), Finale: 2 * time.Hour},
	{Number: 1, EndsAt: time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC), Finale: 2 * time.Hour},
}})

func TestADayRoundEndsAtTheTimeOfDayTheFinaleStarts(t *testing.T) {
	for _, tc := range []struct {
		at   time.Time
		ends time.Time
	}{
		{at: utc(31, 20, 59, 59), ends: utc(31, 21, 0, 0)},
		{at: utc(30, 21, 0, 0), ends: utc(31, 21, 0, 0)},
		{at: utc(30, 20, 59, 59), ends: utc(30, 21, 0, 0)},
		{at: utc(9, 3, 0, 0), ends: utc(9, 21, 0, 0)},
	} {
		round, ok := rounds.Current(twoSeasons, tc.at)

		require.True(t, ok, tc.at)
		assert.Equal(t, rounds.Round{Season: 0, EndsAt: tc.ends}, round, tc.at)
	}
}

func TestTheFinaleIsARoundOfItsOwnToTheEndOfTheSeason(t *testing.T) {
	for _, at := range []time.Time{utc(31, 21, 0, 0), utc(31, 22, 59, 59)} {
		round, ok := rounds.Current(twoSeasons, at)

		require.True(t, ok, at)
		assert.Equal(t, rounds.Round{Season: 0, EndsAt: utc(31, 23, 0, 0), Finale: true}, round, at)
	}
}

func TestTheNextSeasonsFirstRoundStartsWhenTheFinaleEnds(t *testing.T) {
	round, ok := rounds.Current(twoSeasons, utc(31, 23, 0, 0))

	require.True(t, ok)
	assert.Equal(t, rounds.Round{Season: 1, EndsAt: time.Date(2026, 11, 1, 21, 0, 0, 0, time.UTC)}, round)
}

func TestThereIsNoRoundAfterTheLastSeason(t *testing.T) {
	_, ok := rounds.Current(twoSeasons, time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC))

	assert.False(t, ok)
}
