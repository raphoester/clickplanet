package moments_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/moments"
	"github.com/raphoester/clickplanet.lol-backend/internal/gamemaster/internal/season"
)

// Two arena battles back to back on Fri 30 Oct, 18:00 to 21:00 and 21:00 to 23:00: day 23 ends at 18:00, day 24 starts at 23:00.
var (
	firstStarts  = at(30, 18)
	secondStarts = at(30, 21)
	secondEnds   = at(30, 23)
	first        = moments.BattleRoundOf(early)
	second       = moments.BattleRoundOf(late)
)

func backToBack(known moments.Known) moments.Plan {
	return moments.NewPlan(calendar(season.Finale{}), cutoff, nil,
		battle(early, firstStarts, secondStarts), battle(late, secondStarts, secondEnds)).With(known)
}

func both(scored, closed []moments.RoundID) moments.Known {
	return knowing([]moments.Judgement{{Round: first, Mode: moments.Arena}, {Round: second, Mode: moments.Arena}},
		scored, closed)
}

func knowing(judged []moments.Judgement, scored, closed []moments.RoundID) moments.Known {
	return moments.NewKnown(moments.Kept{Scored: scored}, moments.ArenasKept{Judged: judged, Closed: closed})
}

func TestABattleIsJudgedAnArenaBattleWhenItsSetSaysSoUnlessItOpensItsBook(t *testing.T) {
	plan := backToBack(knowing(nil, nil, nil))

	running, ok := plan.Unjudged(firstStarts)
	require.True(t, ok)
	assert.Equal(t, moments.Judgement{Round: first, Mode: moments.Arena}, plan.Judge(running, true))
	assert.Equal(t, moments.Judgement{Round: first, Mode: moments.Plain}, plan.Judge(running, false))
	assert.Equal(t, "rush", string(running.Set))

	opener := moments.NewPlan(calendar(season.Finale{}), cutoff, nil, battle(early, season.Origin, season.Origin.Add(time.Hour)))
	atStart, ok := opener.Unjudged(season.Origin)
	require.True(t, ok)
	assert.Equal(t, moments.Plain, opener.Judge(atStart, true).Mode, "the season's reset blanked the map already")

	_, ok = backToBack(knowing([]moments.Judgement{{Round: first, Mode: moments.Plain}}, nil, nil)).Unjudged(firstStarts)
	assert.False(t, ok, "judged once")
}

func TestAnArenaOpensOnceTheRoundBeforeItIsScored(t *testing.T) {
	assert.Empty(t, backToBack(both(nil, nil)).ArenasDueAt(firstStarts))

	assert.Equal(t, []moments.ArenaCommand{
		{Action: moments.OpenArena, Arena: first, At: firstStarts, EndsAt: secondStarts},
	}, backToBack(both([]moments.RoundID{"season-0/day-23"}, nil)).ArenasDueAt(firstStarts.Add(3*time.Second)))
}

func TestAnArenaBattlesRoundOpensFromNothingOnceItsArenaOpenedAndItsArenaOpensNoMore(t *testing.T) {
	judged := []moments.Judgement{{Round: first, Mode: moments.Arena}}
	scored := moments.Kept{Scored: []moments.RoundID{"season-0/day-23"}}
	now := firstStarts.Add(5 * time.Second)
	open := moments.Command{Action: moments.OpenRound, Book: "season-0", Round: first, Times: 2, At: firstStarts, FromNothing: true}

	waiting := backToBack(moments.NewKnown(scored, moments.ArenasKept{Judged: judged}))
	opened := backToBack(moments.NewKnown(scored, moments.ArenasKept{Judged: judged, Opened: []moments.RoundID{first}}))

	assert.NotContains(t, waiting.DueAt(now), open, "the round waits for the blank map")
	assert.Contains(t, opened.DueAt(now), open)
	assert.Empty(t, opened.ArenasDueAt(now))
}

func TestAnArenaBattleWhoseArenaNeverOpensNeverOpensItsRoundAndClosesIt(t *testing.T) {
	plan := backToBack(both([]moments.RoundID{"season-0/day-23"}, nil))

	for at := firstStarts; at.Before(secondStarts); at = at.Add(time.Minute) {
		for _, command := range plan.DueAt(at) {
			assert.False(t, command.Action == moments.OpenRound && command.Round == first, "%s: %v", at, command)
		}
	}
	assert.Contains(t, plan.DueAt(secondStarts),
		moments.Command{Action: moments.CloseRound, Book: "season-0", Round: first, At: secondStarts})
}

func TestTheNextArenaWaitsForTheMapOfTheOneBefore(t *testing.T) {
	scored := []moments.RoundID{"season-0/day-23", first}

	assert.Equal(t, []moments.ArenaCommand{
		{Action: moments.CloseArena, Arena: first, At: secondStarts, EndsAt: secondStarts},
	}, backToBack(both(scored, nil)).ArenasDueAt(secondStarts.Add(3*time.Second)), "the second opens on the first's map")

	assert.Equal(t, []moments.ArenaCommand{
		{Action: moments.OpenArena, Arena: second, At: secondStarts, EndsAt: secondEnds},
	}, backToBack(both(scored, []moments.RoundID{first})).ArenasDueAt(secondStarts.Add(6*time.Second)))
}

func TestAnArenaClosesOnceItsBattleIsScoredAndUntilPlanetTellsItClosed(t *testing.T) {
	scored := []moments.RoundID{"season-0/day-23", first, second}
	plan := backToBack(both(scored, []moments.RoundID{first}))

	assert.Equal(t, []moments.ArenaCommand{
		{Action: moments.CloseArena, Arena: second, At: secondEnds, EndsAt: secondEnds},
	}, plan.ArenasDueAt(secondEnds.Add(time.Hour)))
	assert.Empty(t, backToBack(both(scored, []moments.RoundID{first, second})).ArenasDueAt(secondEnds.Add(time.Hour)))
}

func TestWhatComesAfterAnArenaBattleWaitsForItsMap(t *testing.T) {
	waiting := backToBack(both([]moments.RoundID{"season-0/day-23", first, second}, []moments.RoundID{first}))

	assert.True(t, waiting.Waits(secondEnds))
	assert.False(t, waiting.Waits(secondStarts.Add(-time.Second)))
	_, ok := waiting.DayAt(secondEnds.Add(time.Minute))
	assert.False(t, ok, "day 24 is not told before the map is back")
	for _, command := range waiting.DueAt(secondEnds.Add(time.Minute)) {
		assert.NotEqual(t, moments.OpenRound, command.Action, "%v", command)
	}

	back := backToBack(both([]moments.RoundID{"season-0/day-23", first, second}, []moments.RoundID{first, second}))
	day, ok := back.DayAt(secondEnds.Add(time.Minute))
	require.True(t, ok)
	assert.Equal(t, 24, int(day.Number))
	assert.Contains(t, back.DueAt(secondEnds.Add(time.Minute)),
		moments.Command{Action: moments.OpenRound, Book: "season-0", Round: "season-0/day-24", Times: 1, At: secondEnds})
}

func TestAPlanThatKnowsNoArenaOpensEveryRoundOnTime(t *testing.T) {
	plan := moments.NewPlan(calendar(season.Finale{}), cutoff, nil, battle(early, firstStarts, secondStarts))

	assert.Contains(t, plan.DueAt(firstStarts),
		moments.Command{Action: moments.OpenRound, Book: "season-0", Round: first, Times: 2, At: firstStarts})
}

func TestAPlanJudgedOpensTheBattleAsAPlanReadAfterTheJudgementWasKept(t *testing.T) {
	unjudged := backToBack(knowing(nil, []moments.RoundID{"season-0/day-23"}, nil))
	arena := moments.Judgement{Round: first, Mode: moments.Arena}

	judged := unjudged.Judged(arena)

	kept := backToBack(knowing([]moments.Judgement{arena}, []moments.RoundID{"season-0/day-23"}, nil))
	assert.Equal(t, kept.DueAt(firstStarts), judged.DueAt(firstStarts))
	assert.Equal(t, kept.ArenasDueAt(firstStarts), judged.ArenasDueAt(firstStarts))
	_, ok := unjudged.Unjudged(firstStarts)
	assert.True(t, ok, "the plan judged is a copy")
}

func TestAPlanThatKnowsNoArenaStaysSoOnceJudged(t *testing.T) {
	plan := moments.NewPlan(calendar(season.Finale{}), cutoff, nil, battle(early, firstStarts, secondStarts))

	judged := plan.Judged(moments.Judgement{Round: first, Mode: moments.Arena})

	assert.Equal(t, plan.DueAt(firstStarts), judged.DueAt(firstStarts))
}
