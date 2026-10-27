package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/seasons/internal/calendar"
)

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	seasonOneEnds  = time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)
)

func twoSeasons() calendar.Config {
	return calendar.Config{DailyCutoff: 21 * time.Hour, List: []calendar.Entry{
		{Number: 0, EndsAt: seasonZeroEnds},
		{Number: 1, EndsAt: seasonOneEnds},
	}}
}

func TestTheCurrentSeasonIsTheFirstThatHasNotEnded(t *testing.T) {
	seasons := calendar.New(twoSeasons())

	season, ok := seasons.Current(seasonZeroEnds.Add(-time.Second))
	require.True(t, ok)
	assert.Equal(t, calendar.Season{
		Number:        0,
		LastDayEndsAt: time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC),
		EndsAt:        seasonZeroEnds,
	}, season)

	season, ok = seasons.Current(seasonZeroEnds.Add(time.Second))
	require.True(t, ok)
	assert.Equal(t, calendar.Season{
		Number:        1,
		LastDayEndsAt: time.Date(2026, 12, 31, 21, 0, 0, 0, time.UTC),
		EndsAt:        seasonOneEnds,
	}, season)

	season, ok = seasons.Current(seasonOneEnds.Add(-time.Second))
	require.True(t, ok)
	assert.Equal(t, calendar.Number(1), season.Number)

	_, ok = seasons.Current(seasonOneEnds.Add(time.Second))
	assert.False(t, ok)
}

func TestASeasonIsOverAtTheInstantItEnds(t *testing.T) {
	season, ok := calendar.New(twoSeasons()).Current(seasonZeroEnds)

	require.True(t, ok)
	assert.Equal(t, calendar.Number(1), season.Number)
}

func TestAnEmptyListHasNoSeason(t *testing.T) {
	require.NoError(t, calendar.Config{}.Validate())

	_, ok := calendar.New(calendar.Config{}).Current(seasonZeroEnds)

	assert.False(t, ok)
}

func TestTwoSeasonsAreAccepted(t *testing.T) {
	require.NoError(t, twoSeasons().Validate())
}

func TestNumbersThatDoNotCountUpFromZeroAreRefused(t *testing.T) {
	config := twoSeasons()
	config.List[0].Number = 1
	config.List[1].Number = 2
	require.ErrorContains(t, config.Validate(), "list[0].number is 1")

	config = twoSeasons()
	config.List[1].Number = 2
	require.ErrorContains(t, config.Validate(), "list[1].number is 2")
}

func TestEndsThatDoNotGoUpAreRefused(t *testing.T) {
	config := twoSeasons()
	config.List[1].EndsAt = seasonZeroEnds
	require.ErrorContains(t, config.Validate(), "list[1].endsAt")

	config = twoSeasons()
	config.List[0].EndsAt = time.Time{}
	require.ErrorContains(t, config.Validate(), "list[0].endsAt")
}

func TestASeasonThatEndsAtACutoffEndsItsLastDayThen(t *testing.T) {
	config := twoSeasons()
	config.List[0].EndsAt = time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC)

	season, ok := calendar.New(config).Current(time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC))

	require.True(t, ok)
	assert.Equal(t, config.List[0].EndsAt, season.LastDayEndsAt)
}

func TestACutoffOutsideADayIsRefused(t *testing.T) {
	for _, cutoff := range []time.Duration{-time.Second, 24 * time.Hour} {
		config := twoSeasons()
		config.DailyCutoff = cutoff
		require.ErrorContains(t, config.Validate(), "dailyCutoff")
	}
}

func TestASeasonThatHoldsNoEndOfADayIsRefused(t *testing.T) {
	config := twoSeasons()
	config.List[1].EndsAt = time.Date(2026, 11, 1, 20, 0, 0, 0, time.UTC)

	require.ErrorContains(t, config.Validate(), "list[1].endsAt is 2026-11-01T20:00:00Z: a season holds the end of a day")
}

func TestTheNextCutoffIsTheFirstEndOfADayAfterATime(t *testing.T) {
	seasons := calendar.New(twoSeasons())
	at := func(day, hour, minute int) time.Time { return time.Date(2026, 10, day, hour, minute, 0, 0, time.UTC) }

	assert.Equal(t, at(30, 21, 0), seasons.CutoffAfter(at(30, 3, 0)))
	assert.Equal(t, at(30, 21, 0), seasons.CutoffAfter(at(30, 20, 59)))
	assert.Equal(t, at(31, 21, 0), seasons.CutoffAfter(at(30, 21, 0)))
	assert.Equal(t, at(31, 21, 0), seasons.CutoffAfter(at(30, 23, 0)))
}

func TestTheSeasonsEndedAreThoseWhoseEndIsPast(t *testing.T) {
	seasons := calendar.New(twoSeasons())

	assert.Empty(t, seasons.Ended(seasonZeroEnds.Add(-time.Second)))
	ended := seasons.Ended(seasonZeroEnds)
	require.Len(t, ended, 1)
	assert.Equal(t, calendar.Number(0), ended[0].Number)
	assert.Len(t, seasons.Ended(seasonOneEnds), 2)
}
