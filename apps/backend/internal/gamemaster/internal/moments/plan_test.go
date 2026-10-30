package moments_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/schedule"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

var (
	cutoff     = season.Cutoff(21 * time.Hour)
	seasonEnds = time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	finaleAt   = seasonEnds.Add(-2 * time.Hour)
	rush       = season.Finale{Length: 2 * time.Hour, Set: "rush", Description: "Triple points.", Times: 3}
	early      = schedule.EntryID(uuid.UUID{15: 1})
	late       = schedule.EntryID(uuid.UUID{15: 2})
	last       = schedule.EntryID(uuid.UUID{15: 3})
)

func calendar(finale season.Finale) season.Calendar {
	return season.NewCalendar(season.Snapshot{Number: 0, Length: seasonEnds.Sub(season.Origin), Finale: finale}.Restore())
}

func battle(id schedule.EntryID, startsAt, endsAt time.Time) schedule.Entry {
	return schedule.EntrySnapshot{ID: id, Set: "rush", Kind: schedule.Battle, StartsAt: startsAt, EndsAt: endsAt, Times: 2}.Restore()
}

func at(day, hour int) time.Time {
	return time.Date(2026, 10, day, hour, 0, 0, 0, time.UTC)
}

func TestASeasonWithNoFinaleClosesWithNoTiebreak(t *testing.T) {
	plan := moments.NewPlan(calendar(season.Finale{}), cutoff, nil)

	ended, err := plan.BookClosed("season-0", seasonEnds)

	require.NoError(t, err)
	assert.Equal(t, moments.SeasonEnded{Season: 0, Book: "season-0", EndedAt: seasonEnds}, ended)
	assert.Equal(t, moments.RoundID(""), plan.DueAt(seasonEnds)[len(plan.DueAt(seasonEnds))-1].Tiebreak)
}

func TestABattleThatRunsToTheSeasonsEndIsCutThereAndIsNoFinale(t *testing.T) {
	plan := moments.NewPlan(calendar(season.Finale{}), cutoff, nil, battle(early, finaleAt, seasonEnds.Add(time.Hour)))

	cut, ok := plan.BattleOf(early)

	require.True(t, ok)
	assert.Equal(t, seasonEnds, cut.EndsAt)
	assert.False(t, cut.EndsSeason)
	assert.Equal(t, moments.RoundID(early.String()), cut.Round())
	_, ok = plan.FinaleAt(finaleAt)
	assert.False(t, ok)
}

func TestTheFinaleIsTheSeasonsBattleNamedByItsSeason(t *testing.T) {
	plan := moments.NewPlan(calendar(rush), cutoff, nil, battle(early, finaleAt.Add(-5*time.Hour), finaleAt.Add(-3*time.Hour)))

	finale, ok := plan.FinaleAt(finaleAt)

	require.True(t, ok)
	want := moments.Battle{
		Season: 0, Number: 2, Times: 3, Description: "Triple points.", StartsAt: finaleAt, EndsAt: seasonEnds, EndsSeason: true,
	}
	assert.Equal(t, want, finale)
	assert.Equal(t, moments.RoundID("season-0/finale"), finale.Round())
	_, ok = plan.FinaleAt(finaleAt.Add(-time.Second))
	assert.False(t, ok, "it has not started")
	_, ok = plan.BattleOf(schedule.EntryID{})
	assert.False(t, ok, "the finale is no entry's")
	assert.Equal(t, []moments.Battle{want}, plan.FinalesAfter(finaleAt.Add(-time.Second)))
	assert.Empty(t, plan.FinalesAfter(finaleAt))

	ended, err := plan.RoundEnded("season-0/finale", seasonEnds)
	require.NoError(t, err)
	assert.Equal(t, moments.BattleEnded{Battle: want, EndedAt: seasonEnds}, ended)

	closed, err := plan.BookClosed("season-0", seasonEnds)
	require.NoError(t, err)
	assert.Equal(t, moments.SeasonEnded{
		Season: 0, Book: "season-0", Finale: "season-0/finale", FinaleBattle: &moments.BattleEnded{Battle: want, EndedAt: seasonEnds},
		EndedAt: seasonEnds,
	}, closed)
}

func TestTheFinaleOfASeasonWithNoneNamesNothing(t *testing.T) {
	_, err := moments.NewPlan(calendar(season.Finale{}), cutoff, nil).RoundEnded("season-0/finale", seasonEnds)

	require.ErrorIs(t, err, moments.ErrUnknownRound)
}

func TestTheFinaleFollowsTheSeasonsEnd(t *testing.T) {
	moved := season.NewCalendar(season.Snapshot{
		Number: 0, Length: seasonEnds.Add(24 * time.Hour).Sub(season.Origin), Finale: rush,
	}.Restore())

	finale, ok := moments.NewPlan(moved, cutoff, nil).FinaleAt(finaleAt.Add(24 * time.Hour))

	require.True(t, ok)
	assert.Equal(t, seasonEnds.Add(24*time.Hour), finale.EndsAt)
}

func TestADayKeepsWhatTheBattlesAtItsStartAndItsEndLeaveIt(t *testing.T) {
	plan := moments.NewPlan(calendar(rush), cutoff, nil, battle(early, at(30, 18), at(30, 21)), battle(late, at(30, 21), at(30, 23)))

	before, ok := plan.DayAt(at(30, 17))
	require.True(t, ok)
	assert.Equal(t, season.Day{Season: 0, Number: 23, StartsAt: at(29, 21), EndsAt: at(30, 18)}, before)
	_, ok = plan.DayAt(at(30, 18))
	assert.False(t, ok, "the battle that ends at the cutoff runs")
	_, ok = plan.DayAt(at(30, 22))
	assert.False(t, ok, "the battle that starts at the cutoff runs")
	after, ok := plan.DayAt(at(30, 23))
	require.True(t, ok)
	assert.Equal(t, season.Day{Season: 0, Number: 24, StartsAt: at(30, 23), EndsAt: at(31, 21)}, after)
	assert.Len(t, plan.DaysOf(0), 24)
	assert.Contains(t, plan.DueAt(at(30, 18)), moments.Command{
		Action: moments.CloseRound, Book: "season-0", Round: "season-0/day-23", At: at(30, 18),
	})
}

func TestBattlesThatFitBetweenTheDaysClashWithNothing(t *testing.T) {
	plan := moments.NewPlan(calendar(rush), cutoff, nil,
		battle(early, at(30, 18), at(30, 21)), battle(late, at(30, 21), at(30, 23)), battle(last, at(31, 18), finaleAt),
		battle(schedule.EntryID(uuid.UUID{15: 4}), seasonEnds, seasonEnds.Add(time.Hour)),
		schedule.EntrySnapshot{Kind: schedule.Plain, StartsAt: finaleAt, EndsAt: seasonEnds}.Restore(),
	)

	require.NoError(t, plan.Clashes(at(29, 12)), "a plain entry plays under the finale, and a battle after the season is none")
	require.NoError(t, moments.NewPlan(calendar(season.Finale{}), cutoff, nil, battle(early, at(28, 14), at(28, 16))).Clashes(at(29, 12)),
		"a battle that ended is not judged")
}

func TestABattleThatDoesNotFitBetweenTheDaysClashes(t *testing.T) {
	for name, c := range map[string]struct {
		finale  season.Finale
		battles []schedule.Entry
		want    error
	}{
		"at no cutoff":         {battles: []schedule.Entry{battle(early, at(30, 14), at(30, 16))}, want: moments.ErrOffCutoff},
		"a finale at none":     {finale: season.Finale{Length: 3 * time.Hour, Set: "rush", Times: 3}, want: moments.ErrOffCutoff},
		"over the finale":      {finale: rush, battles: []schedule.Entry{battle(early, finaleAt, at(31, 22))}, want: moments.ErrOverlap},
		"over another battle":  {battles: []schedule.Entry{battle(early, at(30, 18), at(30, 21)), battle(late, at(30, 20), at(30, 21))}, want: moments.ErrOverlap},
		"over a whole day":     {battles: []schedule.Entry{battle(early, at(29, 21), at(30, 22))}, want: moments.ErrDayTaken},
		"two over a whole day": {battles: []schedule.Entry{battle(early, at(29, 21), at(30, 9)), battle(late, at(30, 9), at(30, 21))}, want: moments.ErrDayTaken},
	} {
		t.Run(name, func(t *testing.T) {
			err := moments.NewPlan(calendar(c.finale), cutoff, nil, c.battles...).Clashes(at(29, 12))

			require.ErrorIs(t, err, c.want)
		})
	}
}
