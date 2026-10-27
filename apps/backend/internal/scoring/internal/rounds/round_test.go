package rounds_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/rounds"
)

func utc(day, hour, minute, second int) time.Time {
	return time.Date(2026, 10, day, hour, minute, second, 0, time.UTC)
}

var twoSeasons = calendar.New(calendar.Config{DailyCutoff: 21 * time.Hour}, calendar.Window{Number: 0, StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: utc(31, 23, 0, 0)}, calendar.Window{Number: 1, StartsAt: utc(31, 23, 0, 0), EndsAt: time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC)})

func TestADayEndsAtTheNextCutoff(t *testing.T) {
	for _, tc := range []struct {
		at   time.Time
		ends time.Time
	}{
		{at: utc(31, 20, 59, 59), ends: utc(31, 21, 0, 0)},
		{at: utc(30, 21, 0, 0), ends: utc(31, 21, 0, 0)},
		{at: utc(30, 20, 59, 59), ends: utc(30, 21, 0, 0)},
		{at: utc(9, 3, 0, 0), ends: utc(9, 21, 0, 0)},
	} {
		round, ok := rounds.Day(twoSeasons, tc.at)

		require.True(t, ok, tc.at)
		assert.Equal(t, rounds.Round{Season: 0, EndsAt: tc.ends, Times: 1}, round, tc.at)
	}
}

func TestNoDayRunsFromTheLastCutoffToTheSeasonsEnd(t *testing.T) {
	for _, at := range []time.Time{utc(31, 21, 0, 0), utc(31, 22, 59, 59)} {
		_, ok := rounds.Day(twoSeasons, at)

		assert.False(t, ok, at)
	}
}

func TestADayThatEndsWithTheSeasonIsItsFinale(t *testing.T) {
	seasons := calendar.New(calendar.Config{DailyCutoff: 21 * time.Hour}, calendar.Window{Number: 0, StartsAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC), EndsAt: utc(31, 21, 0, 0)})

	round, ok := rounds.Day(seasons, utc(31, 20, 0, 0))

	require.True(t, ok)
	assert.Equal(t, rounds.Round{Season: 0, EndsAt: utc(31, 21, 0, 0), Finale: true, Times: 1}, round)
}

func TestTheNextSeasonsFirstDayStartsWhenTheSeasonEnds(t *testing.T) {
	round, ok := rounds.Day(twoSeasons, utc(31, 23, 0, 0))

	require.True(t, ok)
	assert.Equal(t, rounds.Round{Season: 1, EndsAt: time.Date(2026, 11, 1, 21, 0, 0, 0, time.UTC), Times: 1}, round)
}

func TestThereIsNoDayAfterTheLastSeason(t *testing.T) {
	_, ok := rounds.Day(twoSeasons, time.Date(2026, 11, 30, 23, 0, 0, 0, time.UTC))

	assert.False(t, ok)
}

var battle = rounds.EntryID(uuid.MustParse("0192f5a0-7c00-7000-8000-000000000031"))

func TestARoundOfItsOwnIsItsEntrysWindow(t *testing.T) {
	round, ok := rounds.Own(twoSeasons, battle, 3, utc(20, 18, 0, 0), utc(20, 20, 0, 0), utc(20, 18, 0, 1))

	require.True(t, ok)
	assert.Equal(t, rounds.Round{Season: 0, EndsAt: utc(20, 20, 0, 0), Own: true, Entry: battle, StartsAt: utc(20, 18, 0, 0), Times: 3}, round)
}

func TestARoundOfItsOwnThatEndsWithTheSeasonIsItsFinale(t *testing.T) {
	round, ok := rounds.Own(twoSeasons, battle, 3, utc(31, 21, 0, 0), utc(31, 23, 0, 0), utc(31, 21, 0, 0))

	require.True(t, ok)
	assert.True(t, round.Finale)
}

func TestARoundOfItsOwnIsCutAtTheEndOfTheSeasonItStartsIn(t *testing.T) {
	round, ok := rounds.Own(twoSeasons, battle, 3, utc(31, 22, 0, 0), time.Date(2026, 11, 1, 2, 0, 0, 0, time.UTC), utc(31, 22, 0, 0))

	require.True(t, ok)
	assert.Equal(t, rounds.Round{Season: 0, EndsAt: utc(31, 23, 0, 0), Finale: true, Own: true, Entry: battle, StartsAt: utc(31, 22, 0, 0), Times: 3}, round)
}

func TestARoundOfItsOwnHeardOfOnceItEndedIsNotPlayed(t *testing.T) {
	_, ok := rounds.Own(twoSeasons, battle, 3, utc(20, 18, 0, 0), utc(20, 20, 0, 0), utc(20, 20, 0, 0))

	assert.False(t, ok)
}
