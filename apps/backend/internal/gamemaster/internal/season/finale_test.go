package season_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

var (
	rush        = season.Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3}
	finaleZero  = season.Snapshot{Number: 0, Length: zero.Length, Finale: rush}
	finaleStart = seasonZeroEnds.Add(-2 * time.Hour)
)

func TestAFinaleEndsAtTheSeasonsEndAndStartsItsLengthBefore(t *testing.T) {
	window := planned(finaleZero).Windows()[0]

	assert.Equal(t, rush, window.Finale)
	assert.Equal(t, time.Date(2026, 10, 31, 21, 0, 0, 0, time.UTC), window.FinaleStartsAt())
	assert.Equal(t, []schedule.Finale{{
		Set: "rush", Description: "Triple points.", StartsAt: finaleStart, EndsAt: seasonZeroEnds,
	}}, planned(finaleZero, season.Snapshot{Number: 1, Break: day, Length: 28 * day}).Finales())
}

func TestASeasonWithNoFinaleHasNone(t *testing.T) {
	assert.False(t, planned(zero).Windows()[0].Finale.Planned())
	assert.Empty(t, planned(zero).Finales())
}

func TestAFinaleFitsInItsSeasonWithABattlesRules(t *testing.T) {
	for name, finale := range map[string]season.Finale{
		"no length":          {Set: "rush", Times: 3},
		"longer":             {Length: 29 * day, Set: "rush", Times: 3},
		"no set's name":      {Length: time.Hour, Set: "Rush!", Times: 3},
		"no times":           {Length: time.Hour, Set: "rush"},
		"too many times":     {Length: time.Hour, Set: "rush", Times: 11},
		"a long description": {Length: time.Hour, Set: "rush", Times: 3, Description: schedule.Description(strings.Repeat("a", 301))},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := planned(zero).Added(season.Draft{Break: day, Length: 28 * day, Finale: finale}, now)
			require.ErrorIs(t, err, season.ErrInvalidSeason)
		})
	}

	added, err := planned(zero).Added(season.Draft{Break: day, Length: 28 * day, Finale: rush}, now)
	require.NoError(t, err)
	assert.Equal(t, rush, added.TakeSnapshot().Finale)
}

func TestAFinaleIsAddedToTheRunningSeasonBeforeItStarts(t *testing.T) {
	edited, err := planned(zero).Edited(0, season.Draft{Length: zero.Length, Finale: rush}, now)

	require.NoError(t, err)
	assert.Equal(t, finaleZero, edited.TakeSnapshot())

	_, err = planned(zero).Edited(0, season.Draft{Length: zero.Length, Finale: rush}, finaleStart)
	require.ErrorIs(t, err, season.ErrInvalidSeason, "a finale added now would have started")
}

func TestMovingTheEndBeforeTheFinaleStartsMovesTheFinale(t *testing.T) {
	edited, err := planned(finaleZero).Edited(0, season.Draft{Length: zero.Length + day, Finale: rush}, now)

	require.NoError(t, err)
	assert.Equal(t, finaleStart.Add(day), planned(edited.TakeSnapshot()).Windows()[0].FinaleStartsAt())

	_, err = planned(finaleZero).Edited(0, season.Draft{Length: zero.Length - 3*day - 10*time.Hour, Finale: rush}, now)
	require.ErrorIs(t, err, season.ErrInvalidSeason, "its finale would have started")
}

func TestOnceTheFinaleStartedTheSeasonsEndAndItsFinaleStay(t *testing.T) {
	during := finaleStart.Add(time.Minute)
	for name, draft := range map[string]season.Draft{
		"a later end":    {Length: zero.Length + time.Hour, Finale: rush},
		"an earlier end": {Length: zero.Length - time.Minute, Finale: rush},
		"another set":    {Length: zero.Length, Finale: season.Finale{Length: 2 * time.Hour, Set: "storm", Times: 3}},
		"no finale":      {Length: zero.Length},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := planned(finaleZero).Edited(0, draft, during)
			require.ErrorIs(t, err, season.ErrFinaleStarted)
		})
	}

	_, err := planned(finaleZero).Edited(0, season.Draft{Length: zero.Length, Finale: rush}, during)
	require.NoError(t, err, "nothing moves")
}

func TestNoBattleOverlapsAFinale(t *testing.T) {
	calendar := planned(finaleZero)
	battle := func(startsAt, endsAt time.Time) schedule.Entry {
		return schedule.EntrySnapshot{ID: schedule.EntryID(uuid.UUID{15: 1}), Kind: schedule.Battle, StartsAt: startsAt, EndsAt: endsAt, Times: 3}.Restore()
	}

	require.ErrorIs(t, calendar.Overlapped(battle(finaleStart, seasonZeroEnds)), season.ErrFinaleOverlap)
	require.ErrorIs(t, calendar.Overlapped(battle(finaleStart.Add(-time.Hour), finaleStart.Add(time.Second))), season.ErrFinaleOverlap)
	require.ErrorIs(t, calendar.Overlapped(battle(seasonZeroEnds.Add(-time.Second), seasonZeroEnds.Add(time.Hour))), season.ErrFinaleOverlap)
	require.NoError(t, calendar.Overlapped(battle(finaleStart.Add(-time.Hour), finaleStart)), "it ends as the finale starts")
	require.NoError(t, calendar.Overlapped(battle(seasonZeroEnds, seasonZeroEnds.Add(time.Hour))), "it starts as the season ends")
	require.NoError(t, calendar.Overlapped(schedule.EntrySnapshot{Kind: schedule.Plain, StartsAt: finaleStart, EndsAt: seasonZeroEnds}.Restore()),
		"the finale plays above a plain entry")
	require.NoError(t, planned(zero).Overlapped(battle(finaleStart, seasonZeroEnds)), "no finale")
}
