package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/scoring/internal/calendar"
)

var (
	seasonZeroStarts = time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	seasonZeroEnds   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	seasonOneStarts  = time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	seasonOneEnds    = time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)
	cutoff           = calendar.Config{DailyCutoff: 21 * time.Hour}
)

func twoSeasons() calendar.Calendar {
	return calendar.New(cutoff,
		calendar.Window{Number: 1, StartsAt: seasonOneStarts, EndsAt: seasonOneEnds},
		calendar.Window{Number: 0, StartsAt: seasonZeroStarts, EndsAt: seasonZeroEnds},
	)
}

func TestTheCurrentSeasonIsTheOneThatRuns(t *testing.T) {
	seasons := twoSeasons()

	season, ok := seasons.Current(seasonZeroEnds.Add(-time.Second))
	require.True(t, ok)
	assert.Equal(t, calendar.Season{
		Number:        0,
		StartsAt:      seasonZeroStarts,
		LastDayEndsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
		EndsAt:        seasonZeroEnds,
	}, season)

	_, ok = seasons.Current(seasonZeroEnds)
	assert.False(t, ok, "a season is over at its end, and none runs until the next starts")

	season, ok = seasons.Current(seasonOneStarts)
	require.True(t, ok)
	assert.Equal(t, calendar.Season{
		Number:        1,
		StartsAt:      seasonOneStarts,
		LastDayEndsAt: time.Date(2026, 12, 31, 21, 0, 0, 0, time.UTC),
		EndsAt:        seasonOneEnds,
	}, season)

	_, ok = seasons.Current(seasonZeroStarts.Add(-time.Second))
	assert.False(t, ok, "before its start")
	_, ok = seasons.Current(seasonOneEnds)
	assert.False(t, ok)
}

func TestNoSeasonToldIsNoSeason(t *testing.T) {
	_, ok := calendar.New(cutoff).Current(seasonZeroEnds)

	assert.False(t, ok)
}

func TestEveryCopyReadsTheSeasonsPlayedLast(t *testing.T) {
	seasons := calendar.New(cutoff, calendar.Window{Number: 0, StartsAt: seasonZeroStarts, EndsAt: seasonZeroEnds})
	copied := seasons

	seasons.Play(calendar.Window{Number: 0, StartsAt: seasonZeroStarts, EndsAt: seasonZeroEnds.AddDate(0, 0, 7)})

	season, ok := copied.Current(seasonZeroEnds)
	require.True(t, ok)
	assert.Equal(t, seasonZeroEnds.AddDate(0, 0, 7), season.EndsAt)
}

func TestASeasonThatEndsAtACutoffEndsItsLastDayThen(t *testing.T) {
	ends := time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

	season, ok := calendar.New(cutoff, calendar.Window{Number: 0, StartsAt: seasonZeroStarts, EndsAt: ends}).
		Current(time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC))

	require.True(t, ok)
	assert.Equal(t, ends, season.LastDayEndsAt)
}

func TestACutoffOutsideADayIsRefused(t *testing.T) {
	require.NoError(t, cutoff.Validate())
	for _, at := range []time.Duration{-time.Second, 24 * time.Hour} {
		require.ErrorContains(t, calendar.Config{DailyCutoff: at}.Validate(), "dailyCutoff")
	}
}

func TestAWindowEndsAfterItsStart(t *testing.T) {
	require.NoError(t, calendar.Window{StartsAt: seasonZeroStarts, EndsAt: seasonZeroEnds}.Check())
	require.ErrorIs(t, calendar.Window{StartsAt: seasonZeroEnds, EndsAt: seasonZeroEnds}.Check(), calendar.ErrInvalidSeason)
}

func TestTheLatestDecisionOverrules(t *testing.T) {
	kept := calendar.Decision{DecidedAt: seasonZeroStarts}

	assert.True(t, calendar.Decision{DecidedAt: seasonZeroStarts.Add(time.Second)}.Overrules(kept))
	assert.False(t, kept.Overrules(kept), "a command sent again changes nothing")
	assert.False(t, calendar.Decision{DecidedAt: seasonZeroStarts.Add(-time.Second)}.Overrules(kept))
}

func TestTheNextCutoffIsTheFirstEndOfADayAfterATime(t *testing.T) {
	seasons := twoSeasons()
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 10, day, hour, minute, 0, 0, time.UTC) }

	assert.Equal(t, at(30, 21, 0), seasons.CutoffAfter(at(30, 3, 0)))
	assert.Equal(t, at(30, 21, 0), seasons.CutoffAfter(at(30, 20, 59)))
	assert.Equal(t, at(31, 21, 0), seasons.CutoffAfter(at(30, 21, 0)))
	assert.Equal(t, at(31, 21, 0), seasons.CutoffAfter(at(30, 23, 0)))
}

func TestTheSeasonsEndedAreThoseWhoseEndIsPast(t *testing.T) {
	seasons := twoSeasons()

	assert.Empty(t, seasons.Ended(seasonZeroEnds.Add(-time.Second)))
	ended := seasons.Ended(seasonZeroEnds)
	require.Len(t, ended, 1)
	assert.Equal(t, calendar.Number(0), ended[0].Number)
	assert.Len(t, seasons.Ended(seasonOneEnds), 2)
}
