package season_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

const day = 24 * time.Hour

var (
	seasonZeroEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	now            = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	zero           = season.Snapshot{Number: 0, Length: seasonZeroEnds.Sub(season.Origin)}
	month          = season.Draft{Break: day, Length: 28 * day}
)

func planned(snapshots ...season.Snapshot) season.Calendar {
	seasons := make([]season.Season, 0, len(snapshots))
	for _, snapshot := range snapshots {
		seasons = append(seasons, snapshot.Restore())
	}
	return season.NewCalendar(seasons...)
}

func TestEachSeasonStartsItsBreakAfterTheEndOfTheOneBefore(t *testing.T) {
	calendar := planned(season.Snapshot{Number: 1, Break: day, Length: 28 * day}, zero)

	assert.Equal(t, []season.Window{
		{Number: 0, StartsAt: season.Origin, EndsAt: seasonZeroEnds},
		{Number: 1, StartsAt: seasonZeroEnds.Add(day), EndsAt: seasonZeroEnds.Add(29 * day)},
	}, calendar.Windows())
}

func TestAnAddedSeasonTakesTheNextNumberAfterTheLast(t *testing.T) {
	added, err := planned(zero).Added(month, now)

	require.NoError(t, err)
	assert.Equal(t, season.Snapshot{Number: 1, Break: day, Length: 28 * day}, added.TakeSnapshot())
}

func TestASeasonWithNoBreakStartsAtTheEndOfTheOneBefore(t *testing.T) {
	added, err := planned(zero).Added(season.Draft{Length: 28 * day}, now)

	require.NoError(t, err)
	assert.Equal(t, seasonZeroEnds, planned(zero, added.TakeSnapshot()).Windows()[1].StartsAt)
}

func TestASeasonHasALengthAndNoBreakBelowNone(t *testing.T) {
	_, err := planned(zero).Added(season.Draft{Break: day}, now)
	require.ErrorIs(t, err, season.ErrInvalidSeason)

	_, err = planned(zero).Added(season.Draft{Break: -time.Hour, Length: day}, now)
	require.ErrorIs(t, err, season.ErrInvalidSeason)
}

func TestAnAddedSeasonStartsAfterNow(t *testing.T) {
	_, err := planned(zero).Added(month, seasonZeroEnds.Add(day))

	require.ErrorIs(t, err, season.ErrInvalidSeason, "the last season ended long ago: the break is too short")
}

func TestMovingTheEndOfTheRunningSeasonMovesTheSeasonsAfterIt(t *testing.T) {
	calendar := planned(zero, season.Snapshot{Number: 1, Break: day, Length: 28 * day})

	moved, err := calendar.Edited(0, season.Draft{Length: zero.Length + 7*day}, now)
	require.NoError(t, err)

	windows := planned(moved.TakeSnapshot(), season.Snapshot{Number: 1, Break: day, Length: 28 * day}).Windows()
	assert.Equal(t, seasonZeroEnds.Add(7*day), windows[0].EndsAt)
	assert.Equal(t, seasonZeroEnds.Add(8*day), windows[1].StartsAt)
}

func TestTheEndOfTheRunningSeasonMovesOnlyToATimeAfterNow(t *testing.T) {
	_, err := planned(zero).Edited(0, season.Draft{Length: now.Sub(season.Origin)}, now)

	require.ErrorIs(t, err, season.ErrInvalidSeason)
}

func TestARunningSeasonKeepsItsBreak(t *testing.T) {
	_, err := planned(zero).Edited(0, season.Draft{Break: time.Hour, Length: zero.Length}, now)

	require.ErrorIs(t, err, season.ErrStarted)
}

func TestASeasonToComeMayChangeItsBreak(t *testing.T) {
	calendar := planned(zero, season.Snapshot{Number: 1, Break: day, Length: 28 * day})

	_, err := calendar.Edited(1, season.Draft{Break: 3 * day, Length: 7 * day}, now)

	require.NoError(t, err)
}

func TestAnEndedSeasonCannotBeEdited(t *testing.T) {
	_, err := planned(zero).Edited(0, season.Draft{Length: zero.Length + 7*day}, seasonZeroEnds)

	require.ErrorIs(t, err, season.ErrEnded)
}

func TestAnUnknownSeasonIsNotEdited(t *testing.T) {
	_, err := planned(zero).Edited(4, month, now)

	require.ErrorIs(t, err, season.ErrUnknownSeason)
}

func TestOnlyTheLastSeasonToComeIsDeleted(t *testing.T) {
	calendar := planned(zero, season.Snapshot{Number: 1, Break: day, Length: 28 * day}, season.Snapshot{Number: 2, Length: day})

	require.ErrorIs(t, calendar.Deletable(0, now), season.ErrStarted)
	require.ErrorIs(t, calendar.Deletable(1, now), season.ErrNotLast)
	require.NoError(t, calendar.Deletable(2, now))
	require.ErrorIs(t, calendar.Deletable(3, now), season.ErrUnknownSeason)
}

func TestTheRunningAndTheNextSeasonAreReadAtATime(t *testing.T) {
	calendar := planned(zero, season.Snapshot{Number: 1, Break: day, Length: 28 * day})

	running, ok := calendar.RunningAt(now)
	require.True(t, ok)
	assert.Equal(t, season.Number(0), running.Number)
	next, ok := calendar.NextAfter(now)
	require.True(t, ok)
	assert.Equal(t, season.Window{Number: 1, StartsAt: seasonZeroEnds.Add(day), EndsAt: seasonZeroEnds.Add(29 * day)}, next)

	_, ok = calendar.RunningAt(seasonZeroEnds)
	assert.False(t, ok, "a season is over at its end, and the break runs")
	_, ok = calendar.NextAfter(next.StartsAt)
	assert.False(t, ok)
}

func TestTheStartedSeasonsAreTheEndedOnesAndTheOneThatRuns(t *testing.T) {
	calendar := planned(zero, season.Snapshot{Number: 1, Break: day, Length: 28 * day})

	assert.Empty(t, calendar.StartedBy(season.Origin.Add(-time.Second)))
	assert.Len(t, calendar.StartedBy(now), 1)
	assert.Len(t, calendar.StartedBy(seasonZeroEnds.Add(30*day)), 2)
}
