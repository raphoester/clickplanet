package season_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

var (
	seasonZeroStarts = time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	seasonZeroEnds   = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	now              = time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
)

func seasonZero() season.Season {
	return season.Snapshot{Number: 0, StartsAt: seasonZeroStarts, EndsAt: seasonZeroEnds}.Restore()
}

func later(days int) season.Draft {
	return season.Draft{StartsAt: seasonZeroEnds.AddDate(0, 0, days), EndsAt: seasonZeroEnds.AddDate(0, 0, days+28)}
}

func TestTheFirstSeasonIsNumberZero(t *testing.T) {
	added, err := season.NewCalendar().Added(later(1), now)

	require.NoError(t, err)
	assert.Equal(t, season.Number(0), added.TakeSnapshot().Number)
}

func TestAnAddedSeasonTakesTheNextNumber(t *testing.T) {
	added, err := season.NewCalendar(seasonZero()).Added(later(1), now)

	require.NoError(t, err)
	assert.Equal(t, season.Snapshot{Number: 1, StartsAt: later(1).StartsAt, EndsAt: later(1).EndsAt}, added.TakeSnapshot())
}

func TestASeasonEndsAfterItsStart(t *testing.T) {
	draft := later(1)
	draft.EndsAt = draft.StartsAt

	_, err := season.NewCalendar(seasonZero()).Added(draft, now)

	require.ErrorIs(t, err, season.ErrInvalidSeason)
}

func TestASeasonDoesNotOverlapTheOneBefore(t *testing.T) {
	_, err := season.NewCalendar(seasonZero()).Added(season.Draft{
		StartsAt: seasonZeroEnds.Add(-time.Hour), EndsAt: seasonZeroEnds.AddDate(0, 0, 7),
	}, now)

	require.ErrorIs(t, err, season.ErrInvalidSeason)
}

func TestASeasonMayStartAtTheEndOfTheOneBefore(t *testing.T) {
	_, err := season.NewCalendar(seasonZero()).Added(season.Draft{StartsAt: seasonZeroEnds, EndsAt: seasonZeroEnds.AddDate(0, 0, 7)}, now)

	require.NoError(t, err)
}

func TestAnAddedSeasonStartsAfterNow(t *testing.T) {
	_, err := season.NewCalendar().Added(season.Draft{StartsAt: now, EndsAt: now.Add(time.Hour)}, now)

	require.ErrorIs(t, err, season.ErrInvalidSeason)
}

func TestTheEndOfARunningSeasonMovesToATimeAfterNow(t *testing.T) {
	calendar := season.NewCalendar(seasonZero())

	moved, err := calendar.Edited(0, season.Draft{StartsAt: seasonZeroStarts, EndsAt: seasonZeroEnds.AddDate(0, 0, 7)}, now)
	require.NoError(t, err)
	assert.Equal(t, seasonZeroEnds.AddDate(0, 0, 7), moved.TakeSnapshot().EndsAt)

	_, err = calendar.Edited(0, season.Draft{StartsAt: seasonZeroStarts, EndsAt: now}, now)
	require.ErrorIs(t, err, season.ErrInvalidSeason)
}

func TestARunningSeasonKeepsItsStart(t *testing.T) {
	_, err := season.NewCalendar(seasonZero()).Edited(0, season.Draft{
		StartsAt: seasonZeroStarts.Add(time.Hour), EndsAt: seasonZeroEnds,
	}, now)

	require.ErrorIs(t, err, season.ErrStarted)
}

func TestAnEndedSeasonCannotBeEdited(t *testing.T) {
	_, err := season.NewCalendar(seasonZero()).Edited(0, season.Draft{
		StartsAt: seasonZeroStarts, EndsAt: seasonZeroEnds.AddDate(0, 0, 7),
	}, seasonZeroEnds)

	require.ErrorIs(t, err, season.ErrEnded)
}

func TestAnEditedSeasonDoesNotOverlapTheOneAfter(t *testing.T) {
	calendar := season.NewCalendar(seasonZero(), season.Snapshot{Number: 1, StartsAt: later(1).StartsAt, EndsAt: later(1).EndsAt}.Restore())

	_, err := calendar.Edited(0, season.Draft{StartsAt: seasonZeroStarts, EndsAt: later(2).StartsAt}, now)

	require.ErrorIs(t, err, season.ErrInvalidSeason)
}

func TestAnUnknownSeasonIsNotEdited(t *testing.T) {
	_, err := season.NewCalendar(seasonZero()).Edited(4, later(1), now)

	require.ErrorIs(t, err, season.ErrUnknownSeason)
}

func TestOnlyTheLastSeasonToComeIsDeleted(t *testing.T) {
	calendar := season.NewCalendar(
		seasonZero(),
		season.Snapshot{Number: 1, StartsAt: later(1).StartsAt, EndsAt: later(1).EndsAt}.Restore(),
		season.Snapshot{Number: 2, StartsAt: later(30).StartsAt, EndsAt: later(30).EndsAt}.Restore(),
	)

	require.ErrorIs(t, calendar.Deletable(0, now), season.ErrStarted)
	require.ErrorIs(t, calendar.Deletable(1, now), season.ErrNotLast)
	require.NoError(t, calendar.Deletable(2, now))
	require.ErrorIs(t, calendar.Deletable(3, now), season.ErrUnknownSeason)
}

func TestTheRunningAndTheNextSeasonAreReadAtATime(t *testing.T) {
	next := season.Snapshot{Number: 1, StartsAt: later(1).StartsAt, EndsAt: later(1).EndsAt}
	calendar := season.NewCalendar(next.Restore(), seasonZero())

	running, ok := calendar.RunningAt(now)
	require.True(t, ok)
	assert.Equal(t, season.Number(0), running.TakeSnapshot().Number)
	coming, ok := calendar.NextAfter(now)
	require.True(t, ok)
	assert.Equal(t, next, coming.TakeSnapshot())

	_, ok = calendar.RunningAt(seasonZeroEnds)
	assert.False(t, ok, "a season is over at its end, and the next one starts a day later")
	_, ok = calendar.NextAfter(next.StartsAt)
	assert.False(t, ok)
}

func TestTheStartedSeasonsAreTheEndedOnesAndTheOneThatRuns(t *testing.T) {
	next := season.Snapshot{Number: 1, StartsAt: later(1).StartsAt, EndsAt: later(1).EndsAt}
	calendar := season.NewCalendar(seasonZero(), next.Restore())

	assert.Empty(t, calendar.StartedBy(seasonZeroStarts.Add(-time.Second)))
	assert.Len(t, calendar.StartedBy(now), 1)
	assert.Len(t, calendar.StartedBy(next.EndsAt.Add(time.Hour)), 2)
}
